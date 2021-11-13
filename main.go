package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512

	// Client ID Header for WebSocket events
	wsClientIdHeader = "x-ws-client-id"

	// Auth Token header for identifying user
	authTokenHeader = "x-auth-token"

	// Namespace is used for handling multiple todo lists
	namespaceHeader = "x-namespace"

	// Used to separate different task lists
	globalNamespace = "global"

	// Use to get request context
	groceriesRequestContextKey = "groceries"
)

var (
	bind          = flag.String("bind", ":8080", "listen address")
	kvhost        = flag.String("kvhost", "localhost:6379", "a redis compatible server address")
	usersFilePath = flag.String("users", "", "a json file with users and keys")
	env           = flag.String("env", "", "environment to run in (dev is good for frontend)")

	//go:embed static
	static embed.FS

	newline  = []byte{'\n'}
	space    = []byte{' '}
	upgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}

	users map[string]User
)

func main() {
	flag.Parse()

	users = map[string]User{
		"admin": {Username: "admin"},
	}
	if *usersFilePath != "" {
		usersFile, _ := ioutil.ReadFile(*usersFilePath)
		// otherwise unmarshalled values are added to the previously defined map
		users = map[string]User{}
		err := json.Unmarshal(usersFile, &users)
		if err != nil {
			log.Fatalf("Unable to unmarshal users file: %v", err)
		}
	}
	log.Println(users)
	pool := newRedisPool(*kvhost)
	defer pool.Close()
	hub := newHub()
	go hub.run()
	h := Handlers{pool: pool, hub: hub}

	fsys, err := fs.Sub(static, "static")
	if err != nil {
		log.Fatal(err)
	}
	fileServer := http.FileServer(http.FS(fsys))
	if *env == "dev" {
		fileServer = http.FileServer(http.Dir("static"))
	}

	itemsMux := http.NewServeMux()
	itemsMux.HandleFunc("/add", h.AddItemHandler)
	itemsMux.HandleFunc("/delete", h.DeleteItemHandler)
	itemsMux.HandleFunc("/edit", h.EditItemHandler)
	itemsMux.HandleFunc("/toggle", h.ToggleItemHandler)
	itemsMux.HandleFunc("/", h.ItemsHandler)

	mux := http.NewServeMux()
	mux.Handle("/items/", http.StripPrefix("/items", itemsMiddleware(itemsMux)))
	mux.HandleFunc("/ws", func(rw http.ResponseWriter, r *http.Request) {
		serveWS(hub, rw, r)
	})
	mux.Handle("/", fileServer)
	log.Fatal(http.ListenAndServe(*bind, addLogging(os.Stdout, mux)))
}

func itemsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		rc := getRequestContext(req)
		if ok := rc.isAuthorized(); !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ctx := req.Context()
		ctx = context.WithValue(ctx, "groceries", rc)
		groceriesRequest := req.Clone(ctx)
		next.ServeHTTP(w, groceriesRequest)
	})
}

type loggingHandler struct {
	writer  io.Writer
	handler http.Handler
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (h loggingHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	t := time.Now()
	url := *req.URL
	recorder := &statusRecorder{w, 200}
	h.handler.ServeHTTP(recorder, req)
	if req.MultipartForm != nil {
		req.MultipartForm.RemoveAll()
	}
	dur := time.Now().Sub(t)
	log.Printf("%s %s %d %s", dur.String(), req.Method, recorder.status, &url)
}

func addLogging(out io.Writer, h http.Handler) http.Handler {
	return loggingHandler{out, h}
}

func newRedisPool(kvhost string) *redis.Pool {
	return &redis.Pool{
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(fmt.Sprintf("redis://%s", kvhost))
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}
}

type User struct {
	Username string `json:"username"`
}

type RequestContext struct {
	User      *User
	Namespace string
}

func getRequestContext(r *http.Request) *RequestContext {
	authToken := r.Header.Get(authTokenHeader)
	user, _ := users[authToken]
	namespace := r.Header.Get(namespaceHeader)
	if namespace == "" {
		namespace = globalNamespace
	}
	return &RequestContext{
		User:      &user,
		Namespace: namespace,
	}
}

func (rc *RequestContext) isAuthorized() bool {
	if rc.User.Username != "" {
		return true
	}
	return false
}

// item:global:qwer-asdf-1234asdf - global keys
// item:user:global:zcxv-asdf-qwer - user specific namespaces
func (rc *RequestContext) buidlKey(uid string) string {
	key := "item"
	if rc.Namespace != globalNamespace {
		key = fmt.Sprintf("%s:%s", key, rc.User.Username)
	}
	key = fmt.Sprintf("%s:%s", key, rc.Namespace)
	// if rc.Namespace != globalNamespace {
	// 	key = fmt.Sprintf("%s:", rc.User.Username)
	// key = fmt.Sprintf("%s:", rc.Namespace)
	if uid != "" {
		key = fmt.Sprintf("%s:%s", key, uid)
	}
	return key
}

func (rc *RequestContext) buildKeyPattern() string {
	// to make sure we are not building pattern with item uid
	key := rc.buidlKey("")
	return fmt.Sprintf("%s:*", key)
}

type Handlers struct {
	pool *redis.Pool
	hub  *Hub
}

type Item struct {
	UID       string `json:"uid"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	IsChecked bool   `json:"is_checked"`
}

func (h *Handlers) ItemsHandler(w http.ResponseWriter, r *http.Request) {

	conn := h.pool.Get()
	defer conn.Close()

	rc := r.Context().Value(groceriesRequestContextKey).(*RequestContext)
	keyPattern := rc.buildKeyPattern()
	log.Println(keyPattern)
	itemKeys, err := redis.ByteSlices(conn.Do("KEYS", keyPattern))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	items := make([]Item, 0)
	for _, itemKey := range itemKeys {
		var item Item
		result, _ := redis.Bytes(conn.Do("GET", itemKey))
		err = json.Unmarshal(result, &item)
		if err != nil {
			continue
		}
		items = append(items, item)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *Handlers) AddItemHandler(w http.ResponseWriter, r *http.Request) {
	conn := h.pool.Get()
	defer conn.Close()

	rc := r.Context().Value(groceriesRequestContextKey).(*RequestContext)

	uid := uuid.NewString()

	name := r.URL.Query().Get("name")
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	category := r.URL.Query().Get("category")

	item := Item{
		UID:       uid,
		Name:      name,
		Category:  category,
		IsChecked: false,
	}
	data, _ := json.Marshal(item)
	key := rc.buidlKey(uid)
	log.Println(key)
	conn.Do("SET", key, data)
	w.Write(data)

	msg, _ := json.Marshal(Message{ClientID: r.Header.Get(wsClientIdHeader), Type: "add", Data: item})
	h.hub.broadcast <- msg
}

func (h *Handlers) DeleteItemHandler(w http.ResponseWriter, r *http.Request) {
	rc := r.Context().Value(groceriesRequestContextKey).(*RequestContext)
	uid := r.URL.Query().Get("uid")
	if uid == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	conn := h.pool.Get()
	defer conn.Close()
	key := rc.buidlKey(uid)
	log.Println(key)
	itemRaw, _ := redis.Bytes(conn.Do("GET", key))
	if len(itemRaw) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var item Item
	_ = json.Unmarshal(itemRaw, &item)
	_, err := conn.Do("DEL", key)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	msg, _ := json.Marshal(Message{ClientID: r.Header.Get(wsClientIdHeader), Type: "delete", Data: item})
	h.hub.broadcast <- msg
}

func (h *Handlers) EditItemHandler(w http.ResponseWriter, r *http.Request) {
	rc := r.Context().Value(groceriesRequestContextKey).(*RequestContext)
	uid := r.URL.Query().Get("uid")
	if uid == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	name := r.URL.Query().Get("name")
	category := r.URL.Query().Get("category")

	conn := h.pool.Get()
	defer conn.Close()

	key := rc.buidlKey(uid)
	itemRaw, _ := redis.Bytes(conn.Do("GET", key))
	if len(itemRaw) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var item Item
	_ = json.Unmarshal(itemRaw, &item)
	item.Name = name
	item.Category = category
	updatedItem, _ := json.Marshal(item)
	conn.Do("SET", key, updatedItem)

	msg, _ := json.Marshal(Message{ClientID: r.Header.Get(wsClientIdHeader), Type: "edit", Data: item})
	h.hub.broadcast <- msg

}

func (h *Handlers) ToggleItemHandler(w http.ResponseWriter, r *http.Request) {
	rc := r.Context().Value(groceriesRequestContextKey).(*RequestContext)
	uid := r.URL.Query().Get("uid")
	if uid == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	conn := h.pool.Get()
	defer conn.Close()
	key := rc.buidlKey(uid)
	itemRaw, _ := redis.Bytes(conn.Do("GET", key))
	if len(itemRaw) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var item Item
	_ = json.Unmarshal(itemRaw, &item)
	item.IsChecked = !item.IsChecked
	updatedItem, _ := json.Marshal(item)
	conn.Do("SET", key, updatedItem)
	w.Header().Set("Content-Type", "application/json")
	w.Write(updatedItem)

	// Notify all connectede clients
	msg, _ := json.Marshal(Message{ClientID: r.Header.Get(wsClientIdHeader), Type: "toggle", Data: item})
	h.hub.broadcast <- msg
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			var msg Message
			_ = json.Unmarshal(message, &msg)
			for client := range h.clients {
				if client.id == msg.ClientID {
					continue
				}
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

type Message struct {
	ClientID string `json:"client_id"`
	Type     string `json:"type"`
	// Data has to be a marshalable json struct
	Data interface{} `json:"data"`
}

type Client struct {
	id   string
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.hub.broadcast <- message
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func serveWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	clientID := r.URL.Query().Get("client_id")
	client := &Client{id: clientID, hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

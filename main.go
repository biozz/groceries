package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
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
)

var (
	//go:embed static
	static embed.FS

	//go:embed index.html
	indexHTML string

	kvconn redis.Conn

	newline  = []byte{'\n'}
	space    = []byte{' '}
	upgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}

	users map[string]User
)

type User struct {
	Username string `json:"username"`
}

func main() {
	kvhost := "localhost:6379"
	bind := ":8080"
	users = map[string]User{
		"admin": {Username: "admin"},
	}
	if len(os.Args) > 3 {
		bind = os.Args[1]
		kvhost = os.Args[2]
		usersFile, _ := ioutil.ReadFile(os.Args[3])
		json.Unmarshal(usersFile, &users)
	}
	kvconn, err := redis.DialURL(fmt.Sprintf("redis://%s", kvhost))
	if err != nil {
		log.Fatalf("Unable to connect to kv store: %v", err)
	}
	defer kvconn.Close()
	hub := newHub()
	go hub.run()
	h := Handlers{kv: kvconn, hub: hub}

	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(http.FileServer(http.FS(static)))
	r.HandleFunc("/", IndexHandler)
	r.HandleFunc("/items", h.ItemsHandler)
	r.HandleFunc("/additem", h.AddItemHandler)
	r.HandleFunc("/deleteitem", h.DeleteItemHandler)
	r.HandleFunc("/edititem", h.EditItemHandler)
	r.HandleFunc("/toggleitem", h.ToggleItemHandler)
	r.HandleFunc("/ws", func(rw http.ResponseWriter, r *http.Request) {
		serveWS(hub, rw, r)
	})
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(bind, handlers.LoggingHandler(os.Stdout, r)))
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(indexHTML))
}

type Handlers struct {
	kv  redis.Conn
	hub *Hub
}

type Item struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	IsChecked bool   `json:"is_checked"`
}

func (h *Handlers) ItemsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: rewrite token getter to middleware
	authToken := r.Header.Get(authTokenHeader)
	_, ok := users[authToken]
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	itemKeys, err := redis.ByteSlices(h.kv.Do("KEYS", "item:*"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	items := make([]Item, 0)
	for _, itemKey := range itemKeys {
		var item Item
		result, _ := redis.Bytes(h.kv.Do("GET", itemKey))
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
	authToken := r.Header.Get(authTokenHeader)
	_, ok := users[authToken]
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	itemKeys, err := redis.ByteSlices(h.kv.Do("KEYS", "item:*"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	maxID := 1
	for _, itemKey := range itemKeys {
		itemKeyParts := strings.Split(string(itemKey), ":")
		id, _ := strconv.Atoi(itemKeyParts[1])
		if id > maxID {
			maxID = id
		}
	}
	newID := maxID + 1
	name := r.URL.Query().Get("name")
	category := r.URL.Query().Get("category")

	item := Item{
		ID:        newID,
		Name:      name,
		Category:  category,
		IsChecked: false,
	}
	data, _ := json.Marshal(item)
	h.kv.Do("SET", fmt.Sprintf("item:%d", newID), data)
	w.Write(data)

	msg, _ := json.Marshal(Message{ClientID: r.Header.Get(wsClientIdHeader), Type: "add", Data: item})
	h.hub.broadcast <- msg
}

func (h *Handlers) DeleteItemHandler(w http.ResponseWriter, r *http.Request) {
	authToken := r.Header.Get(authTokenHeader)
	_, ok := users[authToken]
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	itemID := r.URL.Query().Get("id")
	if itemID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	key := fmt.Sprintf("item:%s", itemID)
	itemRaw, _ := redis.Bytes(h.kv.Do("GET", key))
	if len(itemRaw) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var item Item
	_ = json.Unmarshal(itemRaw, &item)
	_, err := h.kv.Do("DEL", fmt.Sprintf("item:%s", itemID))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	msg, _ := json.Marshal(Message{ClientID: r.Header.Get(wsClientIdHeader), Type: "delete", Data: item})
	h.hub.broadcast <- msg
}

func (h *Handlers) EditItemHandler(w http.ResponseWriter, r *http.Request) {
	authToken := r.Header.Get(authTokenHeader)
	_, ok := users[authToken]
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	itemID := r.URL.Query().Get("id")
	if itemID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	name := r.URL.Query().Get("name")
	category := r.URL.Query().Get("category")
	key := fmt.Sprintf("item:%s", itemID)
	itemRaw, _ := redis.Bytes(h.kv.Do("GET", key))
	if len(itemRaw) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var item Item
	_ = json.Unmarshal(itemRaw, &item)
	item.Name = name
	item.Category = category
	updatedItem, _ := json.Marshal(item)
	h.kv.Do("SET", key, updatedItem)

	msg, _ := json.Marshal(Message{ClientID: r.Header.Get(wsClientIdHeader), Type: "edit", Data: item})
	h.hub.broadcast <- msg

}

func (h *Handlers) ToggleItemHandler(w http.ResponseWriter, r *http.Request) {
	authToken := r.Header.Get(authTokenHeader)
	_, ok := users[authToken]
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	itemID := r.URL.Query().Get("id")
	if itemID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	key := fmt.Sprintf("item:%s", itemID)
	itemRaw, _ := redis.Bytes(h.kv.Do("GET", key))
	if len(itemRaw) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var item Item
	_ = json.Unmarshal(itemRaw, &item)
	item.IsChecked = !item.IsChecked
	updatedItem, _ := json.Marshal(item)
	h.kv.Do("SET", key, updatedItem)
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

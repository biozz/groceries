package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
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

type User struct {
	Username string `json:"username"`
}

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
	mux.Handle("/items/", http.StripPrefix("/items", ItemsMiddleware(itemsMux)))
	mux.HandleFunc("/ws", func(rw http.ResponseWriter, r *http.Request) {
		serveWS(hub, rw, r)
	})
	mux.Handle("/", fileServer)
	log.Fatal(http.ListenAndServe(*bind, AddLogging(os.Stdout, mux)))
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

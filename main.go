package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

	// Namespace prefix can be "g" for "global" and "my" for "personal"
	namespacePrefixHeader = "x-namespace-prefix"

	// Use to get request context
	groceriesRequestContextKey RequestContextKey = "groceries"
)

type RequestContextKey string

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

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	users = map[string]User{
		"admin": {Username: "admin"},
	}
	if *usersFilePath != "" {
		usersFile, _ := ioutil.ReadFile(*usersFilePath)
		// otherwise unmarshalled values are added to the previously defined map
		users = map[string]User{}
		err := json.Unmarshal(usersFile, &users)
		if err != nil {
			log.Fatal().Err(err).Msg("Unable to unmarshal users file")
		}
	}
	pool := newRedisPool(*kvhost)
	defer pool.Close()
	hub := newHub()
	go hub.run()
	h := Handlers{pool: pool, hub: hub}

	fsys, err := fs.Sub(static, "static")
	if err != nil {
		log.Fatal().Err(err)
	}
	fileServer := http.FileServer(http.FS(fsys))

	// Various dev settings
	if *env == "dev" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		fileServer = http.FileServer(http.Dir("static"))
	}

	log.Print(users)

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
	err = http.ListenAndServe(*bind, AddLogging(os.Stdout, mux))
	log.Fatal().Err(err)
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

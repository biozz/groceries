package main

import (
	"encoding/json"
	"net/http"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
    "github.com/rs/zerolog/log"
)

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
	log.Print(keyPattern)
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
	log.Print(key)
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
	log.Print(key)
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

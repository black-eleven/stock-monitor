package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mu         sync.RWMutex
	quotes     sync.Map
	jwtSecret  string
}

func NewHub(jwtSecret string) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		jwtSecret:  jwtSecret,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			go h.sendSnapshot(client)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) sendSnapshot(client *Client) {
	var quotes []json.RawMessage
	h.quotes.Range(func(_, v interface{}) bool {
		quotes = append(quotes, v.(json.RawMessage))
		return true
	})
	if quotes == nil {
		quotes = []json.RawMessage{}
	}
	msg, _ := json.Marshal(map[string]interface{}{
		"type": "snapshot",
		"data": quotes,
	})
	select {
	case client.send <- msg:
	default:
	}
}

func (h *Hub) BroadcastQuote(quote interface{}) {
	data, _ := json.Marshal(quote)
	// Store in cache so new clients receive it in snapshot
	var tmp struct{ Code string `json:"code"` }
	json.Unmarshal(data, &tmp)
	if tmp.Code != "" {
		h.quotes.Store(tmp.Code, json.RawMessage(data))
	}
	msg, _ := json.Marshal(map[string]interface{}{
		"type": "quote",
		"data": json.RawMessage(data),
	})
	h.broadcast <- msg
}

func (h *Hub) BroadcastAlert(alert interface{}) {
	data, _ := json.Marshal(alert)
	msg, _ := json.Marshal(map[string]interface{}{
		"type": "alert",
		"data": json.RawMessage(data),
	})
	h.broadcast <- msg
}

func (h *Hub) validateToken(tokenStr string) bool {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.jwtSecret), nil
	})
	if err != nil {
		log.Printf("[WS] Token validation error: %v", err)
	}
	return err == nil && token.Valid
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" || !h.validateToken(token) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade error: %v", err)
		return
	}
	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}
	h.register <- client
	go client.writePump()
	go client.readPump()
}

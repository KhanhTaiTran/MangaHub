package websocket

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type ChatHub struct {
	clients    map[*websocket.Conn]*client
	rooms      map[string]map[*websocket.Conn]bool
	Broadcast  chan ChatMessage
	Register   chan *client
	Unregister chan *websocket.Conn
	mu         sync.RWMutex
}

type ChatMessage struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
	Room      string `json:"room,omitempty"`
}

type client struct {
	conn     *websocket.Conn
	userID   string
	username string
	room     string
	send     chan ChatMessage
}

func NewHub() *ChatHub {
	return &ChatHub{
		clients:    make(map[*websocket.Conn]*client),
		rooms:      make(map[string]map[*websocket.Conn]bool),
		Broadcast:  make(chan ChatMessage, 100),
		Register:   make(chan *client, 10),
		Unregister: make(chan *websocket.Conn, 10),
	}
}

func (h *ChatHub) Run() {
	for {
		select {
		case conn := <-h.Unregister:
			h.removeClient(conn)
		case c := <-h.Register:
			h.addClient(c)
		case msg := <-h.Broadcast:
			h.broadcastMessage(msg)
		}
	}
}

func (h *ChatHub) addClient(clientState *client) {
	if clientState.room == "" {
		clientState.room = "lobby"
	}

	h.clients[clientState.conn] = clientState
	if h.rooms[clientState.room] == nil {
		h.rooms[clientState.room] = make(map[*websocket.Conn]bool)
	}
	h.rooms[clientState.room][clientState.conn] = true

	h.Broadcast <- ChatMessage{
		UserID:    clientState.userID,
		Username:  "system",
		Message:   clientState.username + " joined",
		Timestamp: time.Now().Unix(),
		Room:      clientState.room,
	}
}

func (h *ChatHub) removeClient(conn *websocket.Conn) {
	clientState, ok := h.clients[conn]
	if !ok {
		return
	}

	delete(h.clients, conn)
	if roomClients, exists := h.rooms[clientState.room]; exists {
		delete(roomClients, conn)
		if len(roomClients) == 0 {
			delete(h.rooms, clientState.room)
		}
	}
	close(clientState.send)

	h.Broadcast <- ChatMessage{
		UserID:    clientState.userID,
		Username:  "system",
		Message:   clientState.username + " left",
		Timestamp: time.Now().Unix(),
		Room:      clientState.room,
	}
}

func (h *ChatHub) broadcastMessage(msg ChatMessage) {
	room := msg.Room
	if room == "" {
		room = "lobby"
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	//check if room exists before broadcasting
	roomClients, exists := h.rooms[room]
	if !exists {
		return // no clients in this room, skip broadcasting
	}

	for conn := range roomClients {
		// check if client still exists before sendingmessage
		clientState, ok := h.clients[conn]
		if !ok {
			continue // client not found, skip
		}

		select {
		case clientState.send <- msg:
		default:
			//close the channel if send fails and remove the client
			close(clientState.send)

			delete(h.clients, conn)
			delete(roomClients, conn)
		}
	}

	// check the length of room clients
	if len(roomClients) == 0 {
		delete(h.rooms, room) // remove empty room
	}
}

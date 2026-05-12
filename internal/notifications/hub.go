package notifications

import "sync"

// Notification is the payload delivered to web demo clients.
type Notification struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	MangaID   string `json:"manga_id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// Hub stores active SSE subscribers keyed by user ID.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[chan Notification]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[chan Notification]struct{})}
}

// Register adds a new subscriber channel for the given user ID and returns it
func (h *Hub) Register(userID string) chan Notification {
	ch := make(chan Notification, 8)
	if userID == "" {
		return ch
	}

	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[chan Notification]struct{})
	}
	h.clients[userID][ch] = struct{}{}
	h.mu.Unlock()

	return ch
}

// Unregister removes a subscriber channel for the given user ID and closes it
func (h *Hub) Unregister(userID string, ch chan Notification) {
	if userID == "" || ch == nil {
		return
	}

	h.mu.Lock()
	if clients, ok := h.clients[userID]; ok {
		if _, exists := clients[ch]; exists {
			delete(clients, ch)
			if len(clients) == 0 {
				delete(h.clients, userID)
			}
			close(ch)
		}
	}
	h.mu.Unlock()
}

// Broadcast sends a notification to the target users
// If targetUsers is empty, it broadcasts to all connected clients.
func (h *Hub) Broadcast(targetUsers []string, note Notification) {
	targets := h.snapshotTargets(targetUsers)
	for _, ch := range targets {
		select {
		case ch <- note:
		default:
			// Drop when the client is slow to keep the hub responsive.
		}
	}
}

// snapshotTargets returns a slice of channels for the given target user IDs
func (h *Hub) snapshotTargets(targetUsers []string) []chan Notification {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// if no specific targets, broadcast to all clients
	var targets []chan Notification
	if len(targetUsers) == 0 {
		for _, clientSet := range h.clients {
			for ch := range clientSet {
				targets = append(targets, ch)
			}
		}
		return targets
	}

	for _, userID := range targetUsers {
		if clientSet, ok := h.clients[userID]; ok {
			for ch := range clientSet {
				targets = append(targets, ch)
			}
		}
	}

	return targets
}

var hub = NewHub() // create a global hub instance

// Register registers a web demo client for a specific user.
func Register(userID string) chan Notification {
	return hub.Register(userID)
}

// Unregister removes a web demo client subscription.
func Unregister(userID string, ch chan Notification) {
	hub.Unregister(userID, ch)
}

// Broadcast sends a notification to the target users.
func Broadcast(targetUsers []string, note Notification) {
	hub.Broadcast(targetUsers, note)
}

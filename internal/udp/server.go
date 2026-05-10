package udp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// NotificationServer manages UDP client registrations and broadcasts notifications
type NotificationServer struct {
	Port       string
	conn       *net.UDPConn
	clients    map[string]*net.UDPAddr // userID -> addr
	addrIndex  map[string]string       // addr.String() -> userID
	mu         sync.RWMutex
	pending    map[string]map[string]time.Time
	pendingMu  sync.Mutex
	ackTimeout time.Duration
}

// Notification represents a manga-related event to be sent to clients
type Notification struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	MangaID   string `json:"manga_id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type inboundMessage struct {
	Type           string   `json:"type"`
	Token          string   `json:"token,omitempty"`        // use to auth
	TargetUsers    []string `json:"target_users,omitempty"` // optional list of user IDs to target (if empty, broadcast to all)
	EventType      string   `json:"event_type,omitempty"`
	MangaID        string   `json:"manga_id,omitempty"`
	Message        string   `json:"message,omitempty"`
	NotificationID string   `json:"notification_id,omitempty"`
}

type outboundMessage struct {
	Type         string        `json:"type"`
	Message      string        `json:"message,omitempty"`
	Notification *Notification `json:"notification,omitempty"`
}

// create a new notification server instance with the specified UDP port
func NewNotificationServer(port string) *NotificationServer {
	return &NotificationServer{
		Port:       port,
		clients:    make(map[string]*net.UDPAddr),
		addrIndex:  make(map[string]string),
		pending:    make(map[string]map[string]time.Time),
		ackTimeout: 3 * time.Second,
	}
}

// Run starts the UDP server loop and handles registrations and notifications
func (s *NotificationServer) Run(ctx context.Context) error {
	addr, err := net.ResolveUDPAddr("udp", s.Port)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	s.conn = conn

	buf := make([]byte, 4096)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)) // set read timeout (500ms) to allow periodic context check
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				select {
				case <-ctx.Done():
					return nil
				default:
					continue
				}
			}
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			log.Printf("UDP read error: %v", err)
			continue
		}

		// unmarshal incoming message and handle it
		var msg inboundMessage
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			log.Printf("UDP decode error: %v", err)
			continue
		}
		s.handleMessage(remote, msg)
	}
}

// helper func to handle the incoming message based on its type (register, unregister, notify, ack)
func (s *NotificationServer) handleMessage(remote *net.UDPAddr, msg inboundMessage) {
	switch strings.ToLower(msg.Type) {
	case "register":
		// register client with userID from JWT token
		userID, err := validateToken(msg.Token)
		if err != nil {
			s.sendTo(remote, outboundMessage{Type: "error", Message: "invalid token"})
			return
		}
		s.registerClient(userID, remote)
	case "unregister":
		s.unregisterClient(remote)
	case "notify":
		s.broadcastNotification(msg)
	case "ack":
		s.markAck(remote, msg.NotificationID)
	default:
		s.sendTo(remote, outboundMessage{Type: "error", Message: "unknown message type"})
	}
}

// registerClient adds a new client address to the server's registry and sends a confirmation message
func (s *NotificationServer) registerClient(userID string, addr *net.UDPAddr) {
	if userID == "" {
		return
	}
	key := addr.String()

	s.mu.Lock()
	if prev, ok := s.clients[userID]; ok {
		delete(s.addrIndex, prev.String())
	}
	s.clients[userID] = addr
	s.addrIndex[key] = userID
	s.mu.Unlock()

	s.sendTo(addr, outboundMessage{Type: "registered", Message: "ok"})
	log.Printf("UDP client registered: %s (userID: %s)", addr.String(), userID)
}

// unregisterClient removes a client address from the server's registry and sends a confirmation message
func (s *NotificationServer) unregisterClient(addr *net.UDPAddr) {
	key := addr.String()
	s.mu.Lock()
	if userID, ok := s.addrIndex[key]; ok {
		delete(s.clients, userID)
		delete(s.addrIndex, key)
	}
	s.mu.Unlock()

	s.sendTo(addr, outboundMessage{Type: "unregistered", Message: "ok"})
}

// broadcastNotification sends a manga-related notification to all registered clients and tracks pending ACKs for reliabilitys
func (s *NotificationServer) broadcastNotification(msg inboundMessage) {
	notifyType := strings.TrimSpace(msg.EventType)
	if notifyType == "" {
		notifyType = "chapter_release"
	}

	note := Notification{
		ID:        uuid.New().String(),
		Type:      notifyType,
		MangaID:   strings.TrimSpace(msg.MangaID),
		Message:   strings.TrimSpace(msg.Message),
		Timestamp: time.Now().Unix(),
	}

	payload := outboundMessage{Type: "notification", Notification: &note}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("UDP marshal error: %v", err)
		return
	}

	var targetClients []*net.UDPAddr
	if len(msg.TargetUsers) > 0 {
		targetClients = s.snapshotClientsForUsers(msg.TargetUsers)
	} else {
		targetClients = s.snapshotClients()
	}
	if len(targetClients) == 0 {
		log.Printf("UDP broadcast skipped: no target clients online")
		return
	}

	// send and tracking ACKs
	s.trackPending(note.ID, targetClients)
	for _, addr := range targetClients {
		if _, err := s.conn.WriteToUDP(data, addr); err != nil {
			log.Printf("UDP send error: %v", err)
			continue
		}
		log.Printf("UDP notification sent to %s for manga %s", addr.String(), note.MangaID)
	}

	go s.checkAckTimeout(note.ID, data)
}

// snapshotClients returns a slice of currently registered client addresses
func (s *NotificationServer) snapshotClients() []*net.UDPAddr {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := make([]*net.UDPAddr, 0, len(s.clients))
	for _, addr := range s.clients {
		clients = append(clients, addr)
	}
	return clients
}

func (s *NotificationServer) snapshotClientsForUsers(userIDs []string) []*net.UDPAddr {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]bool)
	clients := make([]*net.UDPAddr, 0, len(userIDs))
	for _, userID := range userIDs {
		if addr, ok := s.clients[userID]; ok {
			key := addr.String()
			if seen[key] {
				continue
			}
			seen[key] = true
			clients = append(clients, addr)
		}
	}
	return clients
}

// trackPending initializes the pending ACK tracking for a new notification by storing the client addresses that need to acknowledge it
func (s *NotificationServer) trackPending(notificationID string, clients []*net.UDPAddr) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()

	// initialize pending map for this notification ID with client addresses
	pending := make(map[string]time.Time, len(clients))
	for _, addr := range clients {
		pending[addr.String()] = time.Now()
	}
	s.pending[notificationID] = pending
}

// markAck removes a client address from the pending ACK tracking for a notification when an ACK is received
// and if all ACKs are received, it cleans up the pending entry
func (s *NotificationServer) markAck(addr *net.UDPAddr, notificationID string) {
	if notificationID == "" {
		return
	}

	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()

	// remove the acknowledging client from the pending map for this notification ID
	pending, ok := s.pending[notificationID]
	if !ok {
		return
	}
	delete(pending, addr.String())
	if len(pending) == 0 {
		delete(s.pending, notificationID)
	}
}

func (s *NotificationServer) checkAckTimeout(notificationID string, data []byte) {
	// set max attempts to 3 for retries
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		time.Sleep(s.ackTimeout) // wait 3 sec

		// check if there are still pending acks for this notification
		s.pendingMu.Lock()
		pending, ok := s.pending[notificationID]
		s.pendingMu.Unlock()

		// if no pending acks, we're done
		if !ok || len(pending) == 0 {
			return
		}

		// if there are still pending acks
		for clientIP := range pending {
			log.Printf("UDP ack timeout for %s. Retrying... (attempt %d/%d)", clientIP, attempt, maxRetries)

			//resovle client address and resend notification
			addr, err := net.ResolveUDPAddr("udp", clientIP)
			if err != nil {
				log.Printf("UDP resolve error for %s: %v", clientIP, err)
				continue
			}
			if _, err := s.conn.WriteToUDP(data, addr); err != nil {
				log.Printf("UDP resend error to %s: %v", clientIP, err)
			}
		}
	}

	// if after max retries there are still pending acks, log and give up
	s.pendingMu.Lock()
	pending, ok := s.pending[notificationID]
	if ok {
		delete(s.pending, notificationID) // clean up pending entry after max retries
	}
	s.pendingMu.Unlock()

	if ok && len(pending) > 0 {
		s.mu.Lock()
		for clientIP := range pending {
			if userID, exists := s.addrIndex[clientIP]; exists {
				delete(s.clients, userID)
				delete(s.addrIndex, clientIP)
				log.Printf("UDP client %s removed after %d failed attempts", clientIP, maxRetries)
			}
		}
		s.mu.Unlock()
	}
}

// sendTo sends a message to a specific UDP client address
func (s *NotificationServer) sendTo(addr *net.UDPAddr, msg outboundMessage) {
	if s.conn == nil {
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("UDP marshal error: %v", err)
		return
	}
	if _, err := s.conn.WriteToUDP(data, addr); err != nil {
		log.Printf("UDP send error: %v", err)
	}
}

// validateToken checks the JWT token and returns the user ID if valid
func validateToken(tokenString string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET is not set")
	}
	parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	// check for parsing errors and validate claims
	if err != nil || !parsed.Valid {
		return "", errors.New("invalid token")
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token claims")
	}
	userID, _ := claims["sub"].(string)
	if userID == "" {
		return "", errors.New("missing sub")
	}
	return userID, nil
}

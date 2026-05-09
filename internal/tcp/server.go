package tcp

import (
	"MangaHub/pkg/database"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type ProgressSyncServer struct {
	Port        string
	Connections map[string]*client
	Broadcast   chan ProgressUpdate
	register    chan *client
	unregister  chan *client
}

type ProgressUpdate struct {
	UserID    string `json:"user_id"`
	MangaID   string `json:"manga_id"`
	Chapter   int    `json:"chapter"`
	Status    string `json:"status"`
	ListName  string `json:"list_name"`
	Timestamp int64  `json:"timestamp"`
}

type client struct {
	id     string
	userID string
	conn   net.Conn
	send   chan outboundMessage
}

// Inbound messages from clients (auth, ping, progress)
type inboundMessage struct {
	Type     string `json:"type"`
	Token    string `json:"token,omitempty"`
	MangaID  string `json:"manga_id,omitempty"`
	Chapter  int    `json:"chapter,omitempty"`
	Status   string `json:"status,omitempty"`
	ListName string `json:"list_name,omitempty"`
}

type outboundMessage struct {
	Type     string          `json:"type"`
	Message  string          `json:"message,omitempty"`
	Progress *ProgressUpdate `json:"progress,omitempty"`
}

// Create a new progress sync server instance.
func NewProgressSyncServer(port string) *ProgressSyncServer {
	return &ProgressSyncServer{
		Port:        port,
		Connections: make(map[string]*client),
		Broadcast:   make(chan ProgressUpdate, 100),
		register:    make(chan *client, 10),
		unregister:  make(chan *client, 10),
	}
}

// run starts the TCP listener and the main dispatcher loop
func (s *ProgressSyncServer) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.Port)
	if err != nil {
		return err
	}
	defer listener.Close()
	// main dispatcher for registration and broadcasts
	go s.loop()
	// close listener on shutdown to unblock Accept
	go func() {
		<-ctx.Done()         // close all connections on shutdown
		_ = listener.Close() // cause Accept to return error and exit the loop
	}()

	// for each incoming connection, handle it in a separate goroutine
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done(): // if context is done, exit gracefully
				return nil
			default:
				log.Printf("TCP accept error: %v", err)
				continue
			}
		}
		// Handle each client connection concurrently
		go s.handleConnection(ctx, conn)
	}
}

// main server loop to manage client registration, unregistration, and broadcast fan-out
func (s *ProgressSyncServer) loop() {
	for {
		select {
		case c := <-s.register: // register new client connection
			s.Connections[c.id] = c

		case c := <-s.unregister: // unregister client connection
			if _, ok := s.Connections[c.id]; ok {
				delete(s.Connections, c.id)
				close(c.send)
			}

		case update := <-s.Broadcast: // broadcast progress update to all clients
			msg := outboundMessage{Type: "progress", Progress: &update}

			for id, c := range s.Connections {
				select {
				case c.send <- msg:
				default:
					delete(s.Connections, id)
					close(c.send)

					log.Printf("Client %s dropped due to full send channel", c.userID)
				}
			}
		}
	}
}

// handleConnection handles a single TCP client connection and message stream
func (s *ProgressSyncServer) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	// close connection if context is done (server shutdown)
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()
	// require auth within a short timeout to avoid idle connections
	_ = conn.SetReadDeadline(time.Now().Add(15 * time.Second))

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	// expect first message to be auth with JWT token
	var authMsg inboundMessage
	if err := decoder.Decode(&authMsg); err != nil {
		_ = encoder.Encode(outboundMessage{Type: "error", Message: "invalid auth message"})
		return
	}
	// validate auth message type and token
	if strings.ToLower(authMsg.Type) != "auth" || authMsg.Token == "" {
		_ = encoder.Encode(outboundMessage{Type: "error", Message: "auth required"})
		return
	}

	userID, err := validateToken(authMsg.Token)
	if err != nil {
		_ = encoder.Encode(outboundMessage{Type: "error", Message: "invalid token"})
		return
	}

	// clear read deadline after successful auth
	_ = conn.SetReadDeadline(time.Time{})
	clientID := uuid.New().String()
	// register the new client connection with the server
	c := &client{id: clientID, userID: userID, conn: conn, send: make(chan outboundMessage, 10)}

	s.register <- c
	_ = encoder.Encode(outboundMessage{Type: "ack", Message: "authenticated"})

	// Writer goroutine to serialize outbound messages.
	go writeLoop(ctx, c, encoder)

	for {
		var msg inboundMessage
		if err := decoder.Decode(&msg); err != nil {
			break
		}
		switch strings.ToLower(msg.Type) {
		case "ping":
			c.send <- outboundMessage{Type: "pong", Message: "pong"}
		case "progress":
			if msg.MangaID == "" {
				c.send <- outboundMessage{Type: "error", Message: "manga_id required"}
				continue
			}
			if msg.Chapter < 0 {
				msg.Chapter = 0
			}
			update := ProgressUpdate{
				UserID:    userID,
				MangaID:   msg.MangaID,
				Chapter:   msg.Chapter,
				Status:    normalizeStatus(msg.Status),
				ListName:  normalizeListName(msg.ListName),
				Timestamp: time.Now().Unix(),
			}
			if err := storeProgress(update); err != nil {
				c.send <- outboundMessage{Type: "error", Message: "progress not saved"}
			}
			// Broadcast to other clients after persistence attempt
			s.Broadcast <- update
		default:
			c.send <- outboundMessage{Type: "error", Message: "unknown message type"}
		}
	}
	// unregister client on disconnect
	s.unregister <- c
}

func writeLoop(ctx context.Context, c *client, encoder *json.Encoder) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			// Encode JSON frames as newline-delimited objects
			if err := encoder.Encode(msg); err != nil {
				return
			}
		}
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

// storeProgress upserts progress into list-based tables for consistency with HTTP API
func storeProgress(update ProgressUpdate) error {
	if update.UserID == "" || update.MangaID == "" {
		return errors.New("missing user or manga")
	}

	var exists bool
	if err := database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM manga WHERE id = ?)", update.MangaID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errors.New("manga not found")
	}

	listID, err := ensureUserList(update.UserID, update.ListName)
	if err != nil {
		return err
	}

	_, err = database.DB.Exec(`
		INSERT INTO user_list_items (
			user_id,
			list_id,
			manga_id,
			current_chapter,
			status,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, list_id, manga_id) DO UPDATE SET
			current_chapter = excluded.current_chapter,
			status = excluded.status,
			updated_at = excluded.updated_at
	`, update.UserID, listID, update.MangaID, update.Chapter, update.Status, time.Now())
	return err
}

// helper func to ensure user list exists or create it if not found
func ensureUserList(userID, listName string) (string, error) {
	var listID string
	err := database.DB.QueryRow(
		"SELECT id FROM user_lists WHERE user_id = ? AND name = ?",
		userID,
		listName,
	).Scan(&listID)
	if err == nil {
		return listID, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}

	listID = uuid.New().String()
	_, err = database.DB.Exec(
		"INSERT INTO user_lists (id, user_id, name, created_at) VALUES (?, ?, ?, ?)",
		listID,
		userID,
		listName,
		time.Now(),
	)
	if err != nil {
		return "", err
	}
	return listID, nil
}

// helper function to normalize list names
func normalizeListName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "reading"
	}
	return name
}

// helper function to normalize status values
func normalizeStatus(status string) string {
	status = strings.TrimSpace(status)
	switch status {
	case "reading", "completed", "dropped", "plan_to_read":
		return status
	case "":
		return "reading"
	default:
		return "reading"
	}
}

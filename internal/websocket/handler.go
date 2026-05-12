package websocket

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

// allow all CORS for websocket connections
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var hubOnce sync.Once
var hubInstance *ChatHub

func RegisterRoutes(r *gin.Engine) {
	hubOnce.Do(func() {
		hubInstance = NewHub()
		go hubInstance.Run()
	})

	r.GET("/ws/chat", func(c *gin.Context) {
		handleWebSocket(c, hubInstance)
	})
}

func handleWebSocket(c *gin.Context, hub *ChatHub) {
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token required"})
		return
	}

	userID, username, err := validateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	room := strings.TrimSpace(c.Query("room"))
	if room == "" {
		room = strings.TrimSpace(c.Query("manga_id"))
	}
	clientState := &client{
		conn:     conn,
		userID:   userID,
		username: username,
		room:     room,
		send:     make(chan ChatMessage, 16),
	}

	hub.Register <- clientState

	go writePump(conn, clientState)
	readPump(conn, hub, userID, username, room)
}

// readPump handles incoming messages from the client and broadcasts them to the hub
func readPump(conn *websocket.Conn, hub *ChatHub, userID, username, room string) {
	defer func() {
		hub.Unregister <- conn
		_ = conn.Close()
	}()

	conn.SetReadLimit(2048)
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var inbound struct {
			Message string `json:"message"`
		}
		if err := conn.ReadJSON(&inbound); err != nil {
			break
		}
		message := strings.TrimSpace(inbound.Message)
		if message == "" {
			continue
		}

		hub.Broadcast <- ChatMessage{
			UserID:    userID,
			Username:  username,
			Message:   message,
			Timestamp: time.Now().Unix(),
			Room:      room,
		}
	}
}

// writePump handles outgoing messages to the client and periodic pings to keep the connection alive
func writePump(conn *websocket.Conn, clientState *client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = conn.Close()
	}()

	for {
		select {
		case msg, ok := <-clientState.send:
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteJSON(msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func extractToken(c *gin.Context) string {
	if header := c.GetHeader("Authorization"); header != "" {
		if strings.HasPrefix(strings.ToLower(header), "bearer ") {
			return strings.TrimSpace(header[7:])
		}
		return strings.TrimSpace(header)
	}
	return strings.TrimSpace(c.Query("token"))
}

func validateToken(tokenString string) (string, string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", "", fmt.Errorf("JWT_SECRET is not set")
	}
	parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !parsed.Valid {
		return "", "", fmt.Errorf("invalid token")
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", fmt.Errorf("invalid token claims")
	}
	userID, _ := claims["sub"].(string)
	username, _ := claims["username"].(string)
	if userID == "" {
		return "", "", fmt.Errorf("missing sub")
	}
	if username == "" {
		username = "user"
	}
	return userID, username, nil
}

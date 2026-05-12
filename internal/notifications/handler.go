package notifications

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// RegisterRoutes wires the SSE notification endpoint.
func RegisterRoutes(r *gin.Engine) {
	r.GET("/api/notifications/stream", streamNotifications)
}

// streamNotifications handles Server-Sent Events (SSE) for real-time notifications.
func streamNotifications(c *gin.Context) {
	token := extractToken(c) //extract token from header or query
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token required"})
		return
	}

	userID, err := validateToken(token) //validate token and extract user ID
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// flushable response for SSE
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	// set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Status(http.StatusOK)

	ch := Register(userID)
	defer Unregister(userID, ch)

	keepAlive := time.NewTicker(25 * time.Second) // send a comment every 25s to keep connection alive
	defer keepAlive.Stop()

	for {
		select {
		case note, ok := <-ch:
			if !ok {
				return
			}
			payload, _ := json.Marshal(note)
			_, _ = fmt.Fprintf(c.Writer, "event: notification\n")
			_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
			flusher.Flush()
		case <-keepAlive.C:
			_, _ = fmt.Fprintf(c.Writer, ": ping\n\n")
			flusher.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}

// helper function to extract token from Authorization header or query parameter
func extractToken(c *gin.Context) string {
	if header := c.GetHeader("Authorization"); header != "" {
		lower := strings.ToLower(header)
		if strings.HasPrefix(lower, "bearer ") {
			return strings.TrimSpace(header[7:])
		}
		return strings.TrimSpace(header)
	}
	return strings.TrimSpace(c.Query("token"))
}

func validateToken(tokenString string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", fmt.Errorf("JWT_SECRET is not set")
	}
	parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !parsed.Valid {
		return "", fmt.Errorf("invalid token")
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}
	userID, _ := claims["sub"].(string)
	if userID == "" {
		return "", fmt.Errorf("missing sub")
	}
	return userID, nil
}

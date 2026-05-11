# MangaHub: System Architecture & Technical Report

## 1. Project Overview
MangaHub is a distributed, network-centric system designed for real-time comic tracking. The system demonstrates the practical application of five distinct network protocols (HTTP, TCP, UDP, WebSocket, and gRPC) within a microservices architecture. It is built entirely in Go (Golang) to leverage its native concurrency model (goroutines and channels).

## 2. System Architecture
The backend infrastructure is containerized using Docker and consists of four primary Go-based servers sharing a unified SQLite database:
1. **API Server (`:8080`):** Acts as the primary gateway, handling JWT authentication, RESTful routing, and WebSocket upgrades.
2. **TCP Sync Server (`:9000`):** Maintains persistent connections with clients to synchronize reading progress in real-time.
3. **UDP Notification Server (`:9001`):** A lightweight broadcaster that pushes chapter release alerts to active clients, utilizing an Application-Level Acknowledgment (ACK) mechanism.
4. **gRPC Internal Server (`:50051`):** Handles high-speed, binary-serialized requests for internal operations.

---

## 3. Core Workflows & Code Implementation

### 3.1. HTTP/REST & JWT Authentication
**Workflow:**
The HTTP layer is implemented using the `gin-gonic/gin` framework. When a user logs in via `POST /auth/login`, the server verifies the credentials and issues a JSON Web Token (JWT). For protected endpoints, a custom `JWTMiddleware` intercepts the request, extracts the Bearer token, validates the HMAC signature, and injects the `user_id` into the request context to enforce Role-Based Access Control (RBAC).

**Code Snippet - JWT Middleware:**
```go
func JWTMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Extract token from Authorization header
        tokenString := extractToken(c.GetHeader("Authorization"))
        
        // Validate token cryptographically
        claims, err := validateJWT(tokenString)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
            return
        }
        
        // Inject user context for subsequent handlers
        c.Set("user_id", claims["sub"])
        c.Next()
    }
}
```
### 3.2. Stateful TCP Synchronization
**Workflow:**
To handle continuous reading progress updates without HTTP header overhead, a dedicated TCP server was developed. Each incoming TCP connection is dispatched to its own goroutine. The server maintains a thread-safe map of active clients. When a progress update JSON is received, it updates the database and pushes the payload to a central Broadcast channel, which fans out the data to other connected devices.

**Code Snippet - Concurrent Fan-out Broadcasting:**

```Go
func (s *ProgressSyncServer) loop() {
    for {
        select {
        case update := <-s.Broadcast:
            msg := outboundMessage{Type: "progress", Progress: &update}
            for id, c := range s.Connections {
                select {
                case c.send <- msg: // Send to client's channel
                default:
                    // If channel is full, drop the client to prevent deadlocks
                    delete(s.Connections, id)
                    close(c.send)
                }
            }
        }
    }
}
```
### 3.3. Reliable UDP Notifications (RUDP)
**Workflow:**
UDP is inherently unreliable. To guarantee delivery for critical chapter release notifications, an Application-Level Acknowledgment (ACK) mechanism was engineered. The server broadcasts a datagram with a unique notification_id and adds the client to a pending list. The client must reply with an ack <id>. If no ACK is received within 3 seconds, the server retries up to 3 times before evicting the unresponsive client.

**Code Snippet - ACK Timeout & Retry Logic:**

```Go
func (s *NotificationServer) checkAckTimeout(notificationID string, data []byte) {
    maxRetries := 3
    for attempt := 1; attempt <= maxRetries; attempt++ {
        time.Sleep(3 * time.Second) // Wait for ACK
        
        s.pendingMu.Lock()
        pending, ok := s.pending[notificationID]
        s.pendingMu.Unlock()

        if !ok || len(pending) == 0 { return } // ACK received, exit routine

        // Resend to unresponsive clients
        for clientIP := range pending {
            addr, _ := net.ResolveUDPAddr("udp", clientIP)
            s.conn.WriteToUDP(data, addr)
        }
    }
    // Logic to evict client after 3 failed attempts
}
```
### 3.4. WebSocket Real-time Chat
**Workflow:**
The chat system implements the Actor Model using gorilla/websocket. A central ChatHub manages room states (map[string]map[*websocket.Conn]bool). To prevent zombie connections (silent network drops), a Ping/Pong Heartbeat mechanism runs in the background. The server pings clients every 30 seconds; if a Pong is not received within 60 seconds, the connection is forcefully garbage-collected.

**Code Snippet - Thread-safe Room Broadcasting:**

```Go
func (h *ChatHub) broadcastMessage(msg ChatMessage) {
    h.mu.Lock()
    defer h.mu.Unlock() // Prevent Race Conditions

    roomClients, exists := h.rooms[msg.Room]
    if !exists { return }

    for conn := range roomClients {
        clientState := h.clients[conn]
        clientState.send <- msg
    }
}
```
### 3.5. gRPC Internal Microservice
**Workflow:**
For internal server-to-server communication, gRPC is utilized due to its binary serialization (Protocol Buffers), which drastically reduces payload size and parsing time compared to REST. To maintain security, the gRPC implementation uses metadata.IncomingContext to extract the JWT token from the internal request headers, mimicking HTTP authorization.

**Code Snippet - JWT Extraction via gRPC Metadata:**

```Go
func extractUser(ctx context.Context) (string, string, error) {
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok { return "", "", fmt.Errorf("missing metadata") }

    var token string
    if values := md.Get("authorization"); len(values) > 0 {
        token = strings.TrimSpace(values[0])
    }
    
    // Validate token cryptographically
    userID, username, err := validateToken(token)
    return userID, username, err
}
```
## 4. Conclusion
MangaHub successfully fulfills the requirements of a modern net-centric application. By architecting a distributed backend in Go, the system demonstrates how different network protocols can be strategically combined to optimize performance, real-time capabilities, and system fault tolerance.
### #file:question.md

# Potential Mentor Q&A (Defense Preparation)

This document contains highly probable questions the mentor might ask during the project defense, along with the technical answers you should provide to secure maximum points.

### Q1: Why did you choose to build a custom "Reliable UDP" (ACK/Retry) instead of just using TCP for notifications?
**Your Answer:** "TCP is connection-oriented, which means it requires a 3-way handshake and maintains a continuous state, making it heavy for simple push notifications. UDP is connectionless and much faster for broadcasting to multiple users. By adding a simple Application-Level ACK and a 3-retry mechanism, we achieved the perfect balance: the lightweight broadcasting speed of UDP, with just enough reliability for critical system notifications, without the overhead of TCP."

### Q2: How did you handle Race Conditions when multiple clients chat or update progress at the exact same millisecond?
**Your Answer:** "In Go, Maps are not thread-safe by default. To prevent `concurrent map writes` panics, I implemented `sync.RWMutex` (Read-Write Mutex) in both the WebSocket `ChatHub` and the UDP server. Before modifying the client list or broadcasting a message, the system locks the state (`mu.Lock()`) and unlocks it using `defer mu.Unlock()`. Additionally, I utilized Go `channels` to queue messages, ensuring that data processing remains thread-safe."

### Q3: How is JWT Authentication handled differently across HTTP, WebSocket, and gRPC?
**Your Answer:** * **HTTP:** The JWT is sent normally via the `Authorization: Bearer` header.
* **WebSocket:** Since standard browser WebSockets don't allow setting custom headers during the handshake, I passed the JWT securely via a Query Parameter (`?token=...`) during the connection upgrade.
* **gRPC:** Instead of HTTP headers, gRPC uses context metadata. I injected the token into the `metadata.OutgoingContext` on the client CLI, and the server extracted it using `metadata.FromIncomingContext()`.

### Q4: I see you are using SQLite. Are there any concurrency limitations with this database in your current architecture?
**Your Answer:** "Yes, SQLite is a file-based database. While it is incredibly fast for reads, it locks the entire database file during write operations. Because our system has 4 different servers (API, TCP, UDP, gRPC) running concurrently via Docker, heavy simultaneous writes could trigger a `database is locked` error. For a production environment, migrating to a robust RDBMS like PostgreSQL would be the ideal next step to handle concurrent multi-process writes."

### Q5: How do you prevent "Zombie Connections" (clients that drop network silently) from eating up your server RAM in TCP and WebSockets?
**Your Answer:** "For WebSockets, I implemented a **Ping/Pong Heartbeat** mechanism. The server sends a Ping frame every 30 seconds. If the client doesn't reply with a Pong within a strict 60-second read deadline, the server assumes the client is dead, forcefully closes the socket, and garbage-collects the memory. A similar read deadline strategy is applied to the TCP connections."

### Q6: Why did you use gRPC for internal services instead of just reusing your HTTP REST APIs?
**Your Answer:** "REST uses JSON, which is text-based and requires CPU overhead to serialize and deserialize. gRPC uses **Protocol Buffers (Protobuf)**, which serializes data into a highly compressed binary format. Furthermore, gRPC operates over HTTP/2, allowing multiplexing. For internal microservices that need to communicate at extremely high speeds, gRPC is vastly more efficient than standard REST."

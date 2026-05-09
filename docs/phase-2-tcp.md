# MangaHub Phase 2 Documentation - TCP Progress Sync

## Scope
Phase 2 adds a TCP progress sync server that accepts concurrent connections, requires a JWT auth handshake, broadcasts progress updates, and writes progress to the database.

## Run Instructions
1. Set environment variables:
   - JWT_SECRET: required, same value as the API server
   - DB_PATH: optional, default is mangahub.db
   - TCP_PORT: optional, default is 9000
2. Start the TCP server:
   - go run ./cmd/tcp-server

## TCP Message Protocol
All messages are JSON objects, one per line.

### Auth (first message)
Client must authenticate before any other message.

Request:
{
  "type": "auth",
  "token": "<JWT>"
}

Response:
{
  "type": "ack",
  "message": "authenticated"
}

### Ping
Request:
{
  "type": "ping"
}

Response:
{
  "type": "pong",
  "message": "pong"
}

### Progress Update
Request:
{
  "type": "progress",
  "manga_id": "<id>",
  "chapter": 12,
  "status": "reading",
  "list_name": "reading"
}

Broadcast (to all clients):
{
  "type": "progress",
  "progress": {
    "user_id": "<user>",
    "manga_id": "<id>",
    "chapter": 12,
    "status": "reading",
    "list_name": "reading",
    "timestamp": 1710000000
  }
}

## HTTP API Integration
The HTTP endpoint PUT /users/progress notifies the TCP server in the background after updating the database. It uses a TCP auth handshake with the same JWT used in the HTTP request.

Environment variable for API server:
- TCP_SERVER_ADDR (default 127.0.0.1:9000 for local run, tcp-server:9000 for Docker)

## Testing
1. Start API server and TCP server.
2. Use Postman to log in and get a JWT token.
3. Run the TCP client:
   - go run ./cmd/tcp-client
4. Send auth and progress messages in the TCP client and watch for broadcasts.
5. Update progress via Postman and confirm the TCP client receives a progress broadcast.

## Notes
- The TCP server persists progress into user_list_items for consistency with the HTTP API.
- The server supports concurrent clients and drops slow connections if the send buffer is full.

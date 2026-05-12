# MangaHub Phase 2 Documentation - TCP Progress Sync

## Overview
Phase 2 adds a TCP progress sync server that accepts concurrent connections, requires a JWT auth handshake, broadcasts progress updates, and writes progress to the database.

## Architecture
- TCP server listens on port 9000 (configurable) and manages client sessions.
- First message must be `auth` with a JWT token.
- Progress updates are stored in `user_list_items` and broadcast to all clients.
- TCP notifications are triggered both from TCP clients and via HTTP `/users/progress`.

## Environment Variables
- JWT_SECRET: required for token validation
- TCP_PORT: TCP server port (default 9000)
- TCP_SERVER_ADDR: API -> TCP target (default 127.0.0.1:9000 for local, tcp-server:9000 for Docker)

## Message Protocol
All messages are JSON objects, one per line.

### Auth (first message)
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

Broadcast:
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

## HTTP Integration
- PUT /users/progress (JWT required) updates the DB and notifies the TCP server.

## Error Handling
- Invalid auth or token -> error message, connection closed.
- Unknown message types -> error message.
- Slow clients are dropped if their send buffer fills.

## Demo Guide (Docker)

### Demo Setup
1. Start services:
   ```
   docker compose up --build -d
   ```
2. Get a JWT token (see [docs/phase-1-api.md](docs/phase-1-api.md)). Save it as `<JWT>`.

### Demo Script (TCP + HTTP)
1) Open TCP client inside the TCP container
```bash
docker compose exec tcp-server go run ./cmd/tcp-client
```
Expected: `Connected to localhost:9000!`

2) Authenticate the TCP client
```json
{"type":"auth","token":"<JWT>"}
```
Expected: `ack` message with `authenticated`.

3) Ping check
```json
{"type":"ping"}
```
Expected: `pong` message.

4) Trigger progress update via HTTP
```bash
curl.exe -X PUT http://localhost:8080/users/progress ^
  -H "Authorization: Bearer <JWT>" ^
  -H "Content-Type: application/json" ^
  -d "{\"manga_id\":\"<MANGA_ID>\",\"list_name\":\"reading\",\"status\":\"reading\",\"current_chapter\":9}"
```
Expected: `200` with `Progress updated`. TCP client receives a broadcast.

5) Send progress directly from TCP client (optional)
```json
{"type":"progress","manga_id":"<MANGA_ID>","chapter":10,"status":"reading","list_name":"reading"}
```
Expected: TCP client receives a `progress` broadcast.

## Notes
- The TCP server writes progress to `user_list_items` to stay consistent with the HTTP API.
- Messages are newline-delimited JSON objects.

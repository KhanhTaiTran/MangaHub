# MangaHub API Documentation

## Overview
MangaHub exposes multiple APIs:
- HTTP REST API for authentication, manga catalog, and user library.
- WebSocket chat for real-time rooms.
- TCP progress sync for push updates.
- UDP notifications for chapter release events.
- gRPC internal services for service-to-service calls.

## Base URLs and Ports
- HTTP REST + WebSocket: http://localhost:8080
- TCP progress sync: tcp://localhost:9000
- UDP notifications: udp://localhost:9001
- gRPC: localhost:50051

## Authentication
- JWT is required for protected routes and socket protocols.
- Obtain a token via POST /auth/login and send it as:
  - HTTP: Authorization: Bearer <JWT>
  - TCP/UDP/WebSocket: token field in the auth/register message or query param.

## HTTP REST API

### Health
- GET /health
  - 200 OK if the server is healthy.

### Auth
- POST /auth/register
  - Body: { "username": "...", "password": "..." }
- POST /auth/login
  - Body: { "username": "...", "password": "..." }
  - Response: { "token": "<JWT>", ... }

### Manga
- GET /manga
  - Query params: q, author, genre, status, limit, offset
- GET /manga/:id

### User Library (JWT required)
- POST /users/library
  - Body: { "manga_id": "...", "list_name": "reading", "status": "reading", "current_chapter": 0 }
- GET /users/library
  - Query params: list_name (optional)
- PUT /users/progress
  - Body: { "manga_id": "...", "list_name": "reading", "status": "reading", "current_chapter": 0 }

### Notifications Trigger (JWT required)
- POST /api/manga/:id/notify
  - Body: { "message": "New chapter released" }

### Errors
- 400 for invalid input, 401 for missing/invalid token, 404 for not found, 500 for server errors.

## WebSocket Chat

- Endpoint: ws://localhost:8080/ws/chat
- Method: GET (upgrade)
- Query parameters:
  - token (required): JWT token
  - room (optional): room id; defaults to manga id or "lobby"
- Message format: JSON objects sent as text frames. The server broadcasts to clients in the same room.

## TCP Progress Sync

- Connect to tcp://localhost:9000
- Newline-delimited JSON messages.

### Auth (first message)
Request:
```
{
  "type": "auth",
  "token": "<JWT>"
}
```

Response:
```
{
  "type": "ack",
  "message": "authenticated"
}
```

### Ping
Request:
```
{
  "type": "ping"
}
```

Response:
```
{
  "type": "pong",
  "message": "pong"
}
```

### Progress Update
Request:
```
{
  "type": "progress",
  "manga_id": "<id>",
  "chapter": 12,
  "status": "reading",
  "list_name": "reading"
}
```

Broadcast:
```
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
```

## UDP Notifications

- Send JSON datagrams to udp://localhost:9001

### Register
Request:
```
{
  "type": "register",
  "token": "<JWT>"
}
```

Response:
```
{
  "type": "registered",
  "message": "ok"
}
```

### Unregister
Request:
```
{
  "type": "unregister"
}
```

Response:
```
{
  "type": "unregistered",
  "message": "ok"
}
```

### Notify (server broadcast)
Request:
```
{
  "type": "notify",
  "event_type": "chapter_release",
  "manga_id": "<id>",
  "message": "New chapter released",
  "target_users": ["user-1", "user-2"]
}
```

Broadcast:
```
{
  "type": "notification",
  "notification": {
    "id": "<id>",
    "type": "chapter_release",
    "manga_id": "<id>",
    "message": "New chapter released",
    "timestamp": 1710000000
  }
}
```

### ACK
Request:
```
{
  "type": "ack",
  "notification_id": "<id>"
}
```

## gRPC

- Proto: proto/mangahub.proto
- Services:
  - MangaService: GetManga, SearchManga, UpdateProgress
  - UserService: GetProfile, GetLibrary

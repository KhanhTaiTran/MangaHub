# Phase 4: WebSocket Chat System

## Overview
The WebSocket implementation in MangaHub provides a real-time chat system allowing users to discuss specific manga series in dedicated chat rooms. It uses the `gorilla/websocket` package and implements the Actor Model via a centralized `ChatHub` to manage concurrent client connections safely.

## Architecture
The WebSocket system consists of three main components:
1. **ChatHub (`hub.go`)**: A central registry that maintains active clients and broadcasts messages to specific rooms.
2. **Client (`hub.go`)**: Represents a single WebSocket connection, encapsulating the connection object, user details, and a buffered `send` channel.
3. **Pumps (`handler.go`)**:
    * `readPump`: Listens for incoming messages from the client and forwards them to the Hub.
    * `writePump`: Listens for messages on the client's `send` channel and writes them to the WebSocket connection. It also handles the Ping/Pong heartbeat mechanism.

## Connection Details

* **Endpoint:** `ws://localhost:8080/ws/chat`
* **Method:** HTTP `GET` (Connection Upgrade)
* **Authentication:** JWT Token required.

### Query Parameters
Since standard browser WebSocket APIs do not support setting custom headers (like `Authorization`), authentication is handled via query parameters.

| Parameter | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `token` | string | **Yes** | A valid JWT Token obtained from the `/auth/login` endpoint. |
| `room` | string | No | The chat room ID (usually the `manga_id`). Defaults to `manga_id` param or `"lobby"` if left empty. |

**Connection URL Example:**
```text
ws://localhost:8080/ws/chat?token=eyJhbGciOiJIUzI1NiIs...&room=one-piece
```
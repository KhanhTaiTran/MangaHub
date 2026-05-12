docker compose exec udp-server go run ./cmd/udp-client
# MangaHub Phase 2 Documentation - UDP Notifications

## Overview
The UDP notification system broadcasts chapter release messages to connected clients. Clients register with a JWT token, receive notifications, and send ACKs so the server can retry failed deliveries.

## Architecture
- UDP server listens on a single port and manages a registry of clients by `user_id`.
- Clients register with a JWT token to associate their UDP address with a user.
- Notifications can be triggered by HTTP or sent directly from UDP clients for testing.
- ACK tracking retries delivery and removes unresponsive clients.

## Data Flow
1. Client sends `register` with JWT token.
2. Server stores `user_id -> UDP address`.
3. HTTP trigger or UDP client sends `notify`.
4. Server broadcasts to target users (or all if none specified).
5. Clients send `ack` with `notification_id`.

## Message Protocol
All messages are JSON datagrams.

### Register (client -> server)
Request:
{
  "type": "register",
  "token": "<JWT>"
}

Response:
{
  "type": "registered",
  "message": "ok"
}

### Unregister
Request:
{
  "type": "unregister"
}

Response:
{
  "type": "unregistered",
  "message": "ok"
}

### Notify (server broadcast)
Request (server-triggered via HTTP or UDP):
{
  "type": "notify",
  "event_type": "chapter_release",
  "manga_id": "<id>",
  "message": "New chapter released",
  "target_users": ["user-1", "user-2"]
}

Broadcast:
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

### ACK (client -> server)
Request:
{
  "type": "ack",
  "notification_id": "<id>"
}

## Environment Variables
- UDP_PORT: server listen port (default 9001)
- UDP_SERVER_ADDR: API -> UDP target (default 127.0.0.1:9001 for local, udp-server:9001 for Docker)
- JWT_SECRET: used for validating register tokens

## HTTP Trigger Route
- POST /api/manga/:id/notify (JWT required)
- Body: { "message": "New chapter released" }

The server determines target users from `user_list_items` and sends `target_users` in the UDP payload.

## Error Handling
- Invalid JSON or unknown message type returns an `error` message.
- ACK timeout triggers retries (up to a few attempts) and may remove unresponsive clients.

## Demo Guide (Docker)
1. Enable the UDP service in docker-compose.yml if it is commented.
2. Start services:
   ```
   docker compose up --build -d
   ```
3. Get a JWT token (see [docs/phase-1-api.md](docs/phase-1-api.md)). Save it as `<JWT>`.
4. Open UDP client inside the UDP container:
   ```
   docker compose exec udp-server go run ./cmd/udp-client
   ```
5. Register the client:
   ```
   register <JWT>
   ```
6. Trigger a notification:
   ```
   curl.exe -X POST http://localhost:8080/api/manga/<MANGA_ID>/notify ^
     -H "Authorization: Bearer <JWT>" ^
     -H "Content-Type: application/json" ^
     -d "{\"message\":\"New chapter released\"}"
   ```
7. Confirm the client receives the notification and auto-ACKs it.

Bước 1: Bật Server và mở terminal xem log của Docker (docker compose logs -f udp-server).

Bước 2: Mở 1 terminal khác, chạy con udp-client lên và gõ lệnh đăng ký:
register <DÁN_TOKEN_VÀO_ĐÂY>

Bước 3 (Quan trọng): Bấm thẳng Ctrl + C để tắt cái terminal udp-client đó đi. (Lúc này Server vẫn đinh ninh là bạn đang online vì UDP không có kết nối thường trực như TCP).

Bước 4: Mở Postman (hoặc gọi HTTP API) bắn một thông báo mới cho User đó.

Bước 5: Quay sang nhìn màn hình Log của Server. Bạn sẽ thấy Server gọi mỏi mồm:

Đợi 3s... UDP ack timeout for 127.0.0.1:xxx. Retrying... (attempt 1/3)

Đợi 3s... UDP ack timeout for 127.0.0.1:xxx. Retrying... (attempt 2/3)

Đợi 3s... UDP ack timeout for 127.0.0.1:xxx. Retrying... (attempt 3/3)

UDP client 127.0.0.1:xxx removed after 3 failed attempts

👉 Kịch bản này show ra là ăn trọn 5 điểm Bonus phần RUDP ngay lập tức!

## Notes
- If `target_users` is empty, the server broadcasts to all registered clients.
- ACK retries are logged and slow clients are removed after repeated failures.

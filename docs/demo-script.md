# MangaHub Demo Script (Step-by-Step theo yeu cau)

## Muc tieu demo 
- HTTP: REST, JWT, error handling, CORS
- TCP: multi-connection, JSON protocol, goroutines, graceful termination
- UDP: client register, broadcast, error handling
- gRPC: proto, basic service, client-server, error handling
- WebSocket: upgrade, broadcast, lifecycle, client management
- Performance: 30-40 manga, search < 500ms, TCP 20-30 conn, WS 10-20 users
- Reliability: basic logging, graceful degradation

## Pre-demo checklist 
1. Build va warm up services:
   ```
   docker compose up --build -d
   ```
2. Verify API health:
   ```
   curl.exe -s http://localhost:8080/health
   ```
3. Mo demo UI tren browser:
   ```
   http://localhost:8080/demo/
   ```
4. Neu manga list rong, reseed 1 lan:
   ```
   docker compose run --rm api-server go run ./cmd/seed-mangadex
   ```

## Live demo script 

### Buoc 1: Gioi thieu kien truc 
- Noi: "MangaHub co REST API va cac giao thuc real-time (TCP, UDP, WebSocket), them gRPC cho internal services. SQLite la datastore chinh."
- Show: [docs/architecture.md](docs/architecture.md) diagram.

### Buoc 2: Start services + health check 
- Command:
  ```
  docker compose up -d
  curl.exe -s http://localhost:8080/health
  ```
- Expected: 200 OK hoac "ok".

### Buoc 3: HTTP REST + JWT + Error handling + CORS
- Register + login (JWT):
  ```
  $registerBody = '{"username":"demo_user","password":"pass12345"}'
  curl.exe -s -X POST http://localhost:8080/auth/register -H "Content-Type: application/json" -d $registerBody

  $login = curl.exe -s -X POST http://localhost:8080/auth/login -H "Content-Type: application/json" -d $registerBody
  $JWT = ($login | ConvertFrom-Json).token
  $JWT
  ```
- REST + auth check (401 khi thieu token):
  ```
  curl.exe -s http://localhost:8080/users/library
  ```
- Error handling (400 khi body thieu field):
  ```
  $badBody = '{"username":"only_user"}'
  curl.exe -s -X POST http://localhost:8080/auth/register -H "Content-Type: application/json" -d $badBody
  ```
- CORS (OPTIONS + Allow-Origin):
  ```
  curl.exe -i -X OPTIONS http://localhost:8080/manga -H "Origin: http://example.com" -H "Access-Control-Request-Method: GET"
  ```

### Buoc 4: Performance check 
- So luong manga (30-40 trong DB):
  ```
  $manga = curl.exe -s "http://localhost:8080/manga?limit=40"
  ($manga | ConvertFrom-Json).items.Count
  ```
- Search latency (muc tieu < 500ms):
  ```
  Measure-Command { curl.exe -s "http://localhost:8080/manga?limit=20" | Out-Null }
  ```

### Buoc 5: Library + progress 
- Lay 1 manga_id va cap nhat progress:
  ```
  $MANGA_ID = ($manga | ConvertFrom-Json).items[0].id
  $addBody = "{\"manga_id\":\"$MANGA_ID\",\"list_name\":\"reading\",\"status\":\"reading\",\"current_chapter\":1}"
  curl.exe -s -X POST http://localhost:8080/users/library -H "Authorization: Bearer $JWT" -H "Content-Type: application/json" -d $addBody

  $progressBody = "{\"manga_id\":\"$MANGA_ID\",\"list_name\":\"reading\",\"status\":\"reading\",\"current_chapter\":5}"
  curl.exe -s -X PUT http://localhost:8080/users/progress -H "Authorization: Bearer $JWT" -H "Content-Type: application/json" -d $progressBody

  curl.exe -s -H "Authorization: Bearer $JWT" http://localhost:8080/users/library
  ```

### Buoc 6: TCP progress sync 
- Mo 2 terminal TCP client (de demo concurrency 2-3 clients; neu can test 20-30 thi mo them):
  ```
  docker compose exec tcp-server go run ./cmd/tcp-client
  ```
- Trong tung client, auth + ping:
  ```
  {"type":"auth","token":"<JWT>"}
  {"type":"ping"}
  ```
- Trigger progress update (broadcast toi cac client):
  ```
  $progressBody = "{\"manga_id\":\"$MANGA_ID\",\"list_name\":\"reading\",\"status\":\"reading\",\"current_chapter\":9}"
  curl.exe -s -X PUT http://localhost:8080/users/progress -H "Authorization: Bearer $JWT" -H "Content-Type: application/json" -d $progressBody
  ```
- Graceful termination: dong client bang Ctrl+C, server se remove connection.

Run load test for search:
```
hey -n 2000 -c 50 "http://localhost:8080/manga?q=one&limit=20"
```
Check 95% latency in output; target is < 500ms.
For 100 users:
```
hey -n 4000 -c 100 "http://localhost:8080/manga?q=one&limit=20"
```
---
3) Open TCP client

- go run ./cmd/tcp-client
- Paste these in order (replace <JWT> and <MANGA_ID>):
4) Trigger from HTTP (Postman)

- POST /auth/login → copy token
- GET /manga?limit=1 → pick id
- PUT /users/progress with header:
  - Authorization: Bearer <JWT>
- Body: (in postman)
```
{"manga_id":"<MANGA_ID>","current_chapter":5,"status":"reading","list_name":"reading"}  
```
Expected TCP client output

{"type":"ack","message":"authenticated"}
{"type":"pong","message":"pong"}
A progress broadcast after the HTTP update.

### Buoc 7: UDP notifications 
- Mo 1-2 terminal UDP client:
  ```
  docker compose exec udp-server go run ./cmd/udp-client
  ```
- Dang ky client:
  ```
  register <JWT>
  ```
- Trigger notification (broadcast + ACK):
  ```
  $notifyBody = '{"message":"New chapter released"}'
  curl.exe -s -X POST http://localhost:8080/api/manga/$MANGA_ID/notify -H "Authorization: Bearer $JWT" -H "Content-Type: application/json" -d $notifyBody
  ```
- Error handling (optional): go sai JSON de thay response error tu server.

### Buoc 8: WebSocket chat 
- Mo 2-3 tab:
  - http://localhost:8080/demo/chat.html
- Trong moi tab:
  1) Paste JWT, click "Save Token".
  2) Room = manga_id hoac "demo".
  3) Click "Connect" va send message.
- Check: message broadcast cho cac tab, join/leave message khi Disconnect.
- Concurrency target: mo 10-20 tab neu can.

### Buoc 9: gRPC 
- Demo client-server + error handling:
  ```
  go run ./cmd/grpc-client search --q "one piece"
  go run ./cmd/grpc-client get --id $MANGA_ID
  go run ./cmd/grpc-client profile --token $JWT
  ```
- (Optional) thu token sai de thay error.

### Buoc 10: Reliability va logging 
- Xem log ngan gon:
  ```
  docker compose logs --tail=20 api-server
  ```
- Graceful degradation (optional): stop udp-server, HTTP/TCP/WS van hoat dong.
  ```
  docker compose stop udp-server
  docker compose start udp-server
  ```

## Fallbacks
- Neu API khong respond, chay `docker compose ps` va restart `docker compose up -d`.
- Neu token het han, re-run Buoc 3 de lay token moi.
- Neu manga list rong, reseed:
  ```
  docker compose run --rm api-server go run ./cmd/seed-mangadex
  ```

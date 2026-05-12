# MangaHub Phase 1 Documentation - HTTP REST API

## Overview
Phase 1 delivers the HTTP REST API foundation with authentication, manga search, and user library management. It includes SQLite schema initialization and seed loading for manga data.

## Architecture
- Gin router with JWT middleware protecting `/users/*` and `/api/*` routes.
- SQLite database with WAL mode for concurrency.
- Seed loader inserts manga data from JSON when the table is empty.

## Environment Variables
- JWT_SECRET: required for auth
- DB_PATH: optional, default is mangahub.db
- MANGA_SEED_PATH: optional, default is data/manga_seed.json

## Data Model Notes
- Manga genres are stored as JSON text in SQLite.
- User library uses `user_lists` and `user_list_items` to support multiple lists.
- `user_progress` and `user_notification_prefs` exist for future phases.

## REST Endpoints

### Auth
- POST /auth/register
  - Body: { "username": "...", "password": "..." }
- POST /auth/login
  - Body: { "username": "...", "password": "..." }

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

## Error Handling
- Invalid JSON or missing fields -> 400
- Unauthorized -> 401
- Not found -> 404
- Database errors -> 500

## Demo Guide (Docker)

### Demo Setup
1. Start services:
   ```
   docker compose up --build -d
   ```
2. Check health:
   ```
   curl.exe http://localhost:8080/health
   ```

### Demo Script (HTTP API)
1) Register user
```bash
curl.exe -X POST http://localhost:8080/auth/register ^
  -H "Content-Type: application/json" ^
  -d "{\"username\":\"demo_user\",\"password\":\"pass12345\"}"
```
Expected: `201` with user id and username.

2) Login and copy token
```bash
curl.exe -X POST http://localhost:8080/auth/login ^
  -H "Content-Type: application/json" ^
  -d "{\"username\":\"demo_user\",\"password\":\"pass12345\"}"
```
Expected: `200` with a `token`. Save it as `<JWT>`.

3) List manga and copy one id
```bash
curl.exe "http://localhost:8080/manga?limit=1"
```
Expected: `items` array with at least one manga. Copy `id` as `<MANGA_ID>`.

4) Add to library (JWT required)
```bash
curl.exe -X POST http://localhost:8080/users/library ^
  -H "Authorization: Bearer <JWT>" ^
  -H "Content-Type: application/json" ^
  -d "{\"manga_id\":\"<MANGA_ID>\",\"list_name\":\"reading\",\"status\":\"reading\",\"current_chapter\":1}"
```
Expected: `200` with `Added to library`.

5) Update progress (JWT required)
```bash
curl.exe -X PUT http://localhost:8080/users/progress ^
  -H "Authorization: Bearer <JWT>" ^
  -H "Content-Type: application/json" ^
  -d "{\"manga_id\":\"<MANGA_ID>\",\"list_name\":\"reading\",\"status\":\"reading\",\"current_chapter\":5}"
```
Expected: `200` with `Progress updated`.

6) View library (JWT required)
```bash
curl.exe -H "Authorization: Bearer <JWT>" http://localhost:8080/users/library
```
Expected: `items` with the manga and updated chapter.

## Notes
- If `/manga` returns empty items, regenerate seed data:
  ```
  docker compose run --rm api-server go run ./cmd/seed-mangadex
  ```
- JWT is required for all `/users/*` endpoints.

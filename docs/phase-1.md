# MangaHub Phase 1 Documentation

## Scope
Phase 1 delivers the HTTP REST API foundation with authentication, manga search, and user library management. It also includes SQLite schema initialization and seed loading for manga data.

## Run Instructions
1. Set environment variables:
   - JWT_SECRET: required for auth
   - DB_PATH: optional, default is mangahub.db
   - MANGA_SEED_PATH: optional, default is data/manga_seed.json
2. Start the API server:
   - go run ./cmd/api-server
3. Health check:
   - GET /health

## REST Endpoints

### Auth
- POST /auth/register
  - Body: { "username": "...", "password": "..." }
  - Creates a user and returns basic profile info.

- POST /auth/login
  - Body: { "username": "...", "password": "..." }
  - Returns JWT token and user info.

### Manga
- GET /manga
  - Query params:
    - q: search by title
    - author: filter by author
    - genre: filter by genre text
    - status: ongoing | completed | hiatus
    - limit: default 20 (max 100)
    - offset: default 0
  - Returns { items: [], count: N }

- GET /manga/:id
  - Returns a single manga item by id.

### User Library (JWT required)
- POST /users/library
  - Body: { "manga_id": "...", "list_name": "reading", "status": "reading", "current_chapter": 0 }
  - Adds or updates a manga entry in the user's list.

- GET /users/library
  - Query params:
    - list_name: optional, filter by list
  - Returns { items: [], count: N }

- PUT /users/progress
  - Body: { "manga_id": "...", "list_name": "reading", "status": "reading", "current_chapter": 0 }
  - Updates progress for a manga in a list.

## Data Model Notes
- Manga data is stored in the manga table with genres serialized as JSON text.
- User library uses user_lists and user_list_items to support multiple lists.
- user_progress and user_notification_prefs exist for future phases and are not used in Phase 1.

## Seed Data
- Seed file: data/manga_seed.json
- The API server will load the seed file automatically if the manga table is empty.

## Basic Error Handling
- Invalid JSON or missing fields returns 400.
- Unauthorized requests to protected endpoints return 401.
- Not found returns 404.
- Database errors return 500.

## Next Steps (Phase 2)
- Implement TCP progress sync, UDP notifications, WebSocket chat, and gRPC service.
- Add notification preferences API and wire it to UDP/WebSocket servers.

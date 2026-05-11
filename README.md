# 📚 MangaHub Backend API

![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)
![Gin Framework](https://img.shields.io/badge/Gin-Framework-00ADD8?style=flat)
![SQLite](https://img.shields.io/badge/SQLite-Database-003B57?style=flat&logo=sqlite)
![Docker](https://img.shields.io/badge/Docker-Containerized-2496ED?style=flat&logo=docker)
![CI/CD](https://img.shields.io/badge/CI%2FCD-GitHub_Actions-2088FF?style=flat&logo=github-actions)

MangaHub is a backend system for a manga tracking platform. Built in **Go (Golang)**, it focuses on clean modular structure, practical network programming, and a clear progression through the course phases.

## ✨ Key Features (Phase 1-3)

*   **🔒 Authentication & Security:** 
    *   User registration and login.
    *   Secure stateless authentication using **JWT (JSON Web Tokens)**.
    *   Password hashing and middleware protection for private routes.
*   **📖 Manga Catalog Management:** 
    *   Advanced search functionality with filters (Title, Author, Genre, Status).
    *   Optimized data fetching with limit/offset pagination.
*   **📂 User Library & Tracking:** 
    *   Add manga to personal reading lists (e.g., *Reading, Completed, Plan to Read*).
    *   Track and update reading progress (current chapters) using `UPSERT` mechanisms.
*   **🔁 TCP Progress Sync:**
    *   JWT-authenticated TCP server for progress updates.
    *   Broadcasts progress to connected clients.
*   **📣 UDP Notifications:**
    *   UDP server for chapter release notifications.
    *   Client registration and ACK retries for delivery confirmation.
*   **⚙️ DevOps & CI/CD:** 
    *   Containerized using **Docker & Docker Compose**.
    *   CI pipeline via **GitHub Actions** for build and test checks.

## 🛠 Tech Stack

*   **Language:** Go (1.25)
*   **Web Framework:** Gin (gin-gonic)
*   **Database:** SQLite3 (Configured with WAL mode for high concurrency)
*   **Infrastructure:** Docker, Docker Compose
*   **Automation:** GitHub Actions

## 📂 Project Structure

The project strictly follows the **Standard Go Project Layout** with Domain-Driven Design principles:
```text
mangahub/
├── .github/workflows/            # CI/CD 
├── cmd/
│   ├── api-server/main.go        # HTTP API server
│   ├── tcp-server/main.go        # TCP sync server 
│   ├── udp-client/main.go        # UDP test client
│   ├── udp-server/main.go        # UDP notification server 
│   └── grpc-server/main.go       # gRPC service server 
│
├── internal/
│   ├── auth/                     # Authentication logic
│   ├── manga/                    # Manga data management
│   ├── user/                     # User management
│   ├── tcp/                      # TCP server implementation 
│   ├── udp/                      # UDP server implementation 
│   ├── websocket/                # WebSocket chat implementation 
│   └── grpc/                     # gRPC service implementation 
│
├── pkg/
│   ├── models/                   # Data structures
│   ├── database/                 # Database utilities
│   └── utils/                    # Helper functions
│
├── proto/                        # Protocol buffer definitions
├── data/                         # JSON data files
├── docs/                         # Documentation
├── docker-compose.yml            # Development environment
└── README.md                     # Project guide
```

## 🚀 Getting Started

Prerequisites

* Docker & Docker Compose installed.
* Go installed (for local run).

Run with Docker **(Recommended)**
1. Clone the repository:
   ```
   git clone https://github.com/KhanhTaiTran/MangaHub.git
   ```
2. Start the services:
   ```
   docker compose up --build
   ```
   The API will be available at http://localhost:8080 and the database will be automatically seeded.
    Demo UI: http://localhost:8080/demo/

Run locally **(Phase 1-3)**
1. Set environment variables:
    - JWT_SECRET (required)
    - DB_PATH (optional, default mangahub.db)
    - MANGA_SEED_PATH (optional, default data/manga_seed.json)
    - TCP_SERVER_ADDR (optional, default 127.0.0.1:9000 for local API -> TCP notify)
    - UDP_SERVER_ADDR (optional, default 127.0.0.1:9001 for local API -> UDP notify)
2. Start the API server:
    ```
    go run ./cmd/api-server
    ```
    Demo UI: http://localhost:8080/demo/
3. Start the TCP server:
    ```
    go run ./cmd/tcp-server
    ```
4. Start the UDP server:
    ```
    go run ./cmd/udp-server
    ```
5. UDP test client (optional):
    ```
    go run ./cmd/udp-client
    ```
6. Start the gRPC server (Phase 5):
    ```
    go run ./cmd/grpc-server
    ```

## Documentation
- API Reference: [docs/api.md](docs/api.md)
- Setup Guide: [docs/setup.md](docs/setup.md)
- Architecture Overview: [docs/architecture.md](docs/architecture.md)
- Phase 1 (HTTP REST): [docs/phase-1-api.md](docs/phase-1-api.md)
- Phase 2 (TCP progress sync): [docs/phase-2-tcp.md](docs/phase-2-tcp.md)
- Phase 3 (UDP notifications): [docs/phase-3-udp.md](docs/phase-3-udp.md)
- Phase 4 (WebSocket chat): [docs/phase-4-ws.md](docs/phase-4-ws.md)
- Phase 5 (gRPC internal services): [docs/phase-5-grpc.md](docs/phase-5-grpc.md)
## 🎬 Demo Guides (Docker)
- Phase 1 (HTTP API): [docs/phase-1-api.md](docs/phase-1-api.md)
- Phase 2 (TCP progress sync): [docs/phase-2-tcp.md](docs/phase-2-tcp.md)
- Phase 3 (UDP notifications): [docs/phase-3-udp.md](docs/phase-3-udp.md)
- Phase 4 (Websocket Chat room): [docs/phase-4-ws.md](docs/phase-4-ws.md)
- Phase 5 (gRPC internal services): [docs/phase-5-grpc.md](docs/phase-5-grpc.md)
## 🗺 Roadmap

* [x] Phase 1: Core REST API, Auth, and Database integration.
* [x] Phase 2: TCP progress sync server.
* [x] Phase 3: UDP notifications system.
* [ ] Phase 4: WebSocket chat system.
* [x] Phase 5: gRPC internal services.

---

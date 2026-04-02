# MangaHub

## Project Structure

```text
mangahub/
├── cmd/
│   ├── api-server/main.go        # HTTP API server
│   ├── tcp-server/main.go        # TCP sync server
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

## Setup Instructions

1. Clone the repository.
2. Configure environment variables if needed.
3. Start services using `docker-compose up`.
4. Run specific servers from `cmd/` for development.


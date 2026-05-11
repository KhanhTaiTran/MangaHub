# MangaHub Setup Guide

## Prerequisites
- Go 1.25 (or the version in go.mod)
- Docker and Docker Compose (optional, recommended for quick start)
- protoc + Go plugins (optional, only if you regenerate gRPC stubs)

## Quick Start (Docker)
1. Start services:
   ```
   docker compose up --build
   ```
2. Endpoints:
   - HTTP API and WebSocket: http://localhost:8080
   - TCP: localhost:9000
   - UDP: localhost:9001
   - gRPC: localhost:50051
3. Demo UI (served by the API): http://localhost:8080/demo/

## Local Run
1. Set required environment variables (see table below).
2. Start the API server:
   ```
   go run ./cmd/api-server
   ```
3. Start supporting servers as needed:
   ```
   go run ./cmd/tcp-server
   go run ./cmd/udp-server
   go run ./cmd/grpc-server
   ```
4. Optional clients:
   ```
   go run ./cmd/tcp-client
   go run ./cmd/udp-client
   go run ./cmd/grpc-client <command>
   ```

## Environment Variables

| Name | Required | Default | Used By | Description |
| --- | --- | --- | --- | --- |
| JWT_SECRET | yes | - | all services | Secret for signing and validating JWTs |
| DB_PATH | no | mangahub.db | api-server, grpc-server | SQLite database path |
| MANGA_SEED_PATH | no | data/manga_seed.json | api-server, grpc-server | Seed data file |
| TCP_PORT | no | 9000 | tcp-server | TCP server listen port |
| TCP_SERVER_ADDR | no | 127.0.0.1:9000 (local) | api-server | API -> TCP target |
| UDP_PORT | no | 9001 | udp-server | UDP server listen port |
| UDP_SERVER_ADDR | no | 127.0.0.1:9001 (local) | api-server | API -> UDP target |
| GRPC_ADDR | no | :50051 | grpc-server | gRPC listen address |

For Docker Compose, override any values in docker-compose.yml if needed (for example, TCP_SERVER_ADDR or UDP_SERVER_ADDR).



## Notes
- On Windows PowerShell, use curl.exe for HTTP examples to avoid the Invoke-WebRequest alias.

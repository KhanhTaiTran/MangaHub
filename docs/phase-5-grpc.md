# Phase 5: gRPC Internal Service

This phase adds a simple gRPC service for internal communication.

## Proto Definitions

Proto file: [proto/mangahub.proto](../proto/mangahub.proto)

Services:
- MangaService (GetManga, SearchManga, UpdateProgress)
- UserService (GetProfile, GetLibrary)

Generated Go code lives in [proto/mangahubpb](../proto/mangahubpb).

If you need to re-generate the files:
```
protoc -I proto --go_out=. --go-grpc_out=. proto/mangahub.proto
```

## Run the gRPC Server

Set env vars (same as API):
- JWT_SECRET (required)
- DB_PATH (optional, default mangahub.db)
- MANGA_SEED_PATH (optional, default data/manga_seed.json)
- GRPC_ADDR (optional, default :50051)

Run:
```
go run ./cmd/grpc-server
```

## gRPC Client Demo

Examples (in a second terminal):
```
go run ./cmd/grpc-client search --q "one piece"

go run ./cmd/grpc-client get --id <manga_id>

go run ./cmd/grpc-client profile --token <jwt>

go run ./cmd/grpc-client library --token <jwt> --list reading

go run ./cmd/grpc-client progress --token <jwt> --manga <id> --chapter 10 --status reading
```

## Testing

Run the gRPC service tests:
```
go test ./internal/grpc -run Test
```

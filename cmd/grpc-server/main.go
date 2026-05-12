package main

import (
	grpcservice "MangaHub/internal/grpc"
	"MangaHub/pkg/database"
	"MangaHub/proto/mangahubpb"
	"log"
	"net"
	"os"

	gogrpc "google.golang.org/grpc"
)

func main() {
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	database.InitDB()
	defer func() {
		_ = database.Close()
	}()
	database.InitSchema()

	seedPath := os.Getenv("MANGA_SEED_PATH")
	if seedPath == "" {
		seedPath = "data/manga_seed.json"
	}
	if err := database.SeedMangaFromJSON(seedPath); err != nil {
		log.Printf("Seed skipped: %v", err)
	}

	addr := os.Getenv("GRPC_ADDR")
	if addr == "" {
		addr = ":50051"
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	server := gogrpc.NewServer()
	service := grpcservice.NewServer()
	mangahubpb.RegisterMangaServiceServer(server, service)
	mangahubpb.RegisterUserServiceServer(server, service)

	log.Printf("gRPC server listening on %s", addr)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

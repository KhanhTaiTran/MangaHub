package main

import (
	"MangaHub/internal/tcp"
	"MangaHub/pkg/database"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	database.InitDB()
	defer database.Close()
	database.InitSchema()

	// get TCP port from environment variable or default to 9000
	port := os.Getenv("TCP_PORT")
	if port == "" {
		port = "9000"
	}

	// set up context and signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// listen for OS interrupt signals to trigger graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		cancel()
	}()

	// create and run the TCP server
	server := tcp.NewProgressSyncServer(":" + port)
	log.Printf("Starting TCP Progress Sync server on port :%s", port)
	if err := server.Run(ctx); err != nil {
		log.Fatal("TCP server stopped: ", err)
	}
}

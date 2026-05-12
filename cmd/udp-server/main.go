package main

import (
	"MangaHub/internal/udp"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// get UDP port from environment variable or default to 9001
	port := os.Getenv("UDP_PORT")
	if port == "" {
		port = "9001"
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

	// create and run the UDP server
	server := udp.NewNotificationServer(":" + port)
	log.Printf("Starting UDP Notification server on port :%s", port)
	if err := server.Run(ctx); err != nil {
		log.Fatal("UDP server stopped: ", err)
	}
}

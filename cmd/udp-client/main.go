package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
)

func main() {
	// connect to the UDP server at localhost:9001
	serverAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9001")
	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Connected to UDP Server (localhost:9001)")

	// when client starts, it should immediately send a "register" message to the server
	conn.Write([]byte(`{"type":"register"}`))
	fmt.Println("Sent Register request...")

	// run a background goroutine to listen for incoming messages from the server
	go func() {
		buf := make([]byte, 4096)
		for {
			n, _, err := conn.ReadFromUDP(buf) // ReadFromUDP to get sender's address
			if err != nil {
				return
			}
			rawMsg := string(buf[:n]) // Convert bytes to string for logging
			fmt.Println("\n Server sent:", rawMsg)

			// check if the message is a notification
			var payload struct {
				Type         string `json:"type"`
				Notification struct {
					ID string `json:"id"`
				} `json:"notification"`
			}
			json.Unmarshal(buf[:n], &payload)

			// if it's a notification -> immediately send ACK back with ID
			if payload.Type == "notification" && payload.Notification.ID != "" {
				ackMsg := fmt.Sprintf(`{"type":"ack", "notification_id":"%s"}`, payload.Notification.ID)
				conn.Write([]byte(ackMsg))
				fmt.Println("Automatically sent ACK to server!")
			}
		}
	}()

	// main loop to read user input and send as notifications to the server
	fmt.Println("\nType a message (or 'exit' to quit):")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "exit" {
			break
		}
		// send a notify message to the server with the text and a dummy manga_id
		notifyMsg := fmt.Sprintf(`{"type":"notify", "message":"%s", "manga_id":"304ceac3-8cdb-4fe7-acf7-2b6ff7a60613"}`, text) // test notification with "attack on titan" manga ID
		conn.Write([]byte(notifyMsg))
	}
}

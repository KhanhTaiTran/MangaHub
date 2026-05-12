package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	addr := os.Getenv("UDP_SERVER_ADDR")
	if addr == "" {
		addr = "127.0.0.1:9001"
	}

	serverAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		fmt.Println("Resolve error:", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		fmt.Println("Dial error:", err)
		return
	}
	defer conn.Close()

	fmt.Printf("Connected to UDP server (%s)\n", addr)
	fmt.Println("Commands:")
	fmt.Println("  register <JWT>")
	fmt.Println("  unregister")
	fmt.Println("  notify <manga_id> <message>")
	fmt.Println("  ack <notification_id>")
	fmt.Println("  exit")

	go readLoop(conn)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" {
			return
		}

		parts := strings.SplitN(line, " ", 3)
		switch parts[0] {
		case "register":
			if len(parts) < 2 {
				fmt.Println("Usage: register <JWT>")
				continue
			}
			send(conn, map[string]any{"type": "register", "token": parts[1]})
		case "unregister":
			send(conn, map[string]any{"type": "unregister"})
		case "notify":
			if len(parts) < 3 {
				fmt.Println("Usage: notify <manga_id> <message>")
				continue
			}
			send(conn, map[string]any{
				"type":       "notify",
				"event_type": "chapter_release",
				"manga_id":   parts[1],
				"message":    parts[2],
			})
		case "ack":
			if len(parts) < 2 {
				fmt.Println("Usage: ack <notification_id>")
				continue
			}
			send(conn, map[string]any{"type": "ack", "notification_id": parts[1]})
		default:
			fmt.Println("Unknown command")
		}
	}
}

func readLoop(conn *net.UDPConn) {
	buf := make([]byte, 4096)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Read error:", err)
			return
		}
		fmt.Printf("Received: %s\n", string(buf[:n]))

		var payload struct {
			Type         string `json:"type"`
			Notification struct {
				ID string `json:"id"`
			} `json:"notification"`
		}
		if err := json.Unmarshal(buf[:n], &payload); err != nil {
			continue
		}
		if payload.Type == "notification" && payload.Notification.ID != "" {
			ack := map[string]any{"type": "ack", "notification_id": payload.Notification.ID}
			send(conn, ack)
			fmt.Println("Auto ACK sent")
		}
	}
}

func send(conn *net.UDPConn, payload map[string]any) {
	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Marshal error:", err)
		return
	}
	if _, err := conn.Write(data); err != nil {
		fmt.Println("Send error:", err)
	}
}

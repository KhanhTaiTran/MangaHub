package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	// connect to TCP server at localhost:9000
	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()
	fmt.Println("Connected to localhost:9000!")
	fmt.Println("Enter JSON string then press Enter to send (type 'exit' to quit):")

	// start a goroutine to read responses from the server
	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			fmt.Println("server response:", scanner.Text())
			fmt.Println("===========================================")
		}
	}()

	// read user input from stdin and send to server
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "exit" {
			break
		}

		// Send the JSON string you just typed + newline character (\n)
		conn.Write([]byte(text + "\n"))
	}
}

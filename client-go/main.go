package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	sc := NewStreamClient("localhost", 5000, true, "your_secret_token")

	sc.OnTCPMessage(func(message string) {
		fmt.Println("TCP message event:", message)
	})

	err := sc.Connect()
	if err != nil {
		fmt.Println("Failed to connect to the server:", err)
		os.Exit(1)
	}
	defer sc.Close()

	err = sc.SendTCPMessage("Hello from the client! TCP")
	if err != nil {
		fmt.Println("Failed to send TCP message:", err)
		os.Exit(1)
	}

	err = sc.SendUDPMessage([]byte("Hello, encrypted server! UDP"))
	if err != nil {
		fmt.Println("Failed to send UDP message:", err)
		os.Exit(1)
	}

	fmt.Println("Messages sent to server.")

	time.Sleep(15 * time.Second)
}
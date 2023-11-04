package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	sc := NewStreamClient("127.0.0.1", 5000, true, "your_secret_token")

	sc.OnConnected(func(message string) {
		fmt.Println("Connected to server.")
	})

	sc.OnDisconnected(func(message string) {
		fmt.Println("Disconnected from server.")
	})

	sc.OnTimeout(func(message string) {
		fmt.Println("Connection to server timed out.")
	})

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

	for i := 0; i < 100; i++ {
		err = sc.SendUDPMessage([]byte("Hello, encrypted server! UDP"))
		if err != nil {
			fmt.Println("Failed to send UDP message:", err)
			os.Exit(1)
		}
	}

	fmt.Println("Messages sent to server.")

	time.Sleep(15 * time.Second)
}

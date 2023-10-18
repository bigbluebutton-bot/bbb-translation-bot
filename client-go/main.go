package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	tcpclient := NewClient("localhost:5000", true)
	tcpclient.Secret_token = "your_secret_token"

	tcpclient.AddEventHandler("connected", func(message string) {
		fmt.Println("Connected event:", message)
	})

	tcpclient.AddEventHandler("disconnected", func(message string) {
		fmt.Println("Disconnected event:", message)
	})

	tcpclient.AddEventHandler("message", func(message string) {
		fmt.Println("Message event:", message)
	})

	tcpclient.AddEventHandler("timeout", func(message string) {
		fmt.Println("Timeout event:", message)
	})

	tcpclient.AddEventHandler("ping", func(message string) {
		fmt.Println("Ping event:", message)
	})

	err := tcpclient.Connect()
	if err != nil {
		fmt.Println("Failed to connect to the server:", err)
		os.Exit(1)
	}
	defer tcpclient.disconnect()

	err = tcpclient.Send("Hello from the client!")
	if err != nil {
		fmt.Println("Failed to send message:", err)
		os.Exit(1)
	}

	aesKey := []byte{
		0xe8, 0xab, 0x8d, 0x5d, 0x22, 0xfe, 0x15, 0xf0,
		0x4a, 0x48, 0x30, 0x7b, 0xd0, 0x6c, 0x10, 0xaa,
		0x84, 0x3c, 0x87, 0xab, 0x72, 0x8a, 0x24, 0x7d,
		0x94, 0x76, 0x4c, 0x9b, 0x6a, 0x64, 0x00, 0x34,
	}
	aesIV := []byte{
		0xe0, 0xab, 0x95, 0x32, 0x7e, 0x3a, 0xa1, 0x3d,
		0x20, 0x34, 0x62, 0xa8, 0x09, 0xa3, 0x71, 0x9e,
	}


	udpclient := NewUDPclient("localhost:5001", true, aesKey, aesIV)


	err = udpclient.Connect()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer udpclient.Close()

	err = udpclient.SendMessage([]byte("Hello, encrypted server!"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Message sent to server.")





	time.Sleep(15 * time.Second)
}
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"time"
)

const (
	serverAddr    = "localhost:5000"
	secretToken   = "your_secret_token"
	bufferSize    = 1024
)

func main() {
	// Connect to the server
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Println("Failed to connect to server:", err)
		return
	}
	defer conn.Close()

	// Receive server's RSA public key
	keyBuffer := make([]byte, bufferSize)
	n, err := conn.Read(keyBuffer)
	if err != nil {
		fmt.Println("Failed to read server public key:", err)
		return
	}
	serverPublicKeyPEM := keyBuffer[:n]
	block, _ := pem.Decode(serverPublicKeyPEM)
	serverPublicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		fmt.Println("Failed to parse server public key:", err)
		return
	}

	// Generate AES key and IV
	aesKey := make([]byte, 32) // 256-bit AES key
	aesIV := make([]byte, 16)  // AES IV
	rand.Read(aesKey)
	rand.Read(aesIV)
	fmt.Println("Generated AES Key:", aesKey)
	fmt.Println("Generated AES IV:", aesIV)	

	// Encrypt AES key and IV using server's RSA public key
	encryptedKeyIV, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, serverPublicKey.(*rsa.PublicKey), append(aesIV, aesKey...), nil)
	if err != nil {
		fmt.Println("Failed to encrypt AES key and IV:", err)
		return
	}
	// Send encrypted AES key and IV to server
	conn.Write(encryptedKeyIV)

	// Wait for message from server
	messageBuffer := make([]byte, bufferSize)
	n, err = conn.Read(messageBuffer)
	if err != nil {
		fmt.Println("Failed to read message from server:", err)
		return
	}
	encrypted_message := messageBuffer[:n]
	// Decrypt message with AES
	blockCipher, err := aes.NewCipher(aesKey)
	if err != nil {
		fmt.Println("Failed to create AES cipher:", err)
		return
	}
	stream := cipher.NewCFBDecrypter(blockCipher, aesIV)
	message := make([]byte, len(encrypted_message))
	stream.XORKeyStream(message, encrypted_message)

	fmt.Println(string(message))
	if string(message) != "OK" {
		fmt.Println("Server failed to decrypt AES key and IV")
		return
	}

	// Encrypt SECRET_TOKEN with AES and send to server for validation
	stream = cipher.NewCFBEncrypter(blockCipher, aesIV)
	encryptedToken := make([]byte, len(secretToken))
	stream.XORKeyStream(encryptedToken, []byte(secretToken))
	fmt.Println("plaintext bytes: ", []byte(secretToken))
	fmt.Println("encryptedToken: ", encryptedToken)

	conn.Write(encryptedToken)
	fmt.Println("Secret token sent to server!")

	go func() {
		// send ping every 4 seconds
		for {
			time.Sleep(4 * time.Second)
			
			stream := cipher.NewCFBEncrypter(blockCipher, aesIV)
			pingmsg := "PING"
			encryptedMessage := make([]byte, len(pingmsg))
			stream.XORKeyStream(encryptedMessage, []byte(pingmsg))
			conn.Write(encryptedMessage)
			fmt.Println("Ping send to server!")


			// wait for pong
			messageBuffer := make([]byte, bufferSize)
			n, err := conn.Read(messageBuffer)
			if err != nil {
				fmt.Println("Failed to read message from server:", err)
				return
			}
			encrypted_message := messageBuffer[:n]
			// Decrypt message with AES
			blockCipher, err := aes.NewCipher(aesKey)
			if err != nil {
				fmt.Println("Failed to create AES cipher:", err)
				return
			}
			stream = cipher.NewCFBDecrypter(blockCipher, aesIV)
			message := make([]byte, len(encrypted_message))
			stream.XORKeyStream(message, encrypted_message)

			fmt.Println(string(message))
			if string(message) != "PONG" {
				fmt.Println("Server failed to decrypt AES key and IV")
				return
			}
			fmt.Println("Pong received from server!")
		}
	}()

	time.Sleep(30 * time.Second)
}
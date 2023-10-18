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
	"sync"
	"time"
)

type Client struct {
	address     string
	connection  net.Conn
	eventLock   sync.Mutex
	msgSendLock sync.Mutex
	eventHandlers map[string][]func(message string)
	aesKey      []byte
	aesIV       []byte
	running     bool
	encryptionEnabled bool
	serverPublicKey   *rsa.PublicKey
	BufferSize  int
	PingTimeIntervall time.Duration
	StopChan chan bool

	Secret_token string
}

func NewClient(addr string, encryption bool) *Client {
	return &Client{
		address:      addr,
		connection:   nil,
		eventLock:    sync.Mutex{},
		msgSendLock:  sync.Mutex{},
		eventHandlers: make(map[string][]func(message string)),
		aesKey:       nil,
		aesIV:        nil,
		running:      false,
		encryptionEnabled: encryption,
		serverPublicKey: nil,
		BufferSize:   1024,
		PingTimeIntervall: 4 * time.Second,
		StopChan: make(chan bool),

		Secret_token: "",
	}
}

func (c *Client) Send(message string) error {
	c.msgSendLock.Lock()
	defer c.msgSendLock.Unlock()

	if c.encryptionEnabled {
		blockCipher, err := aes.NewCipher(c.aesKey)
		if err != nil {
			fmt.Println("Failed to create AES cipher:", err)
			return err
		}
		
		// Encrypt SECRET_TOKEN with AES and send to server for validation
		stream := cipher.NewCFBEncrypter(blockCipher, c.aesIV)
		encryptedToken := make([]byte, len(message))
		stream.XORKeyStream(encryptedToken, []byte(message))
		fmt.Println("plaintext bytes: ", []byte(message))
		fmt.Println("encryptedToken: ", encryptedToken)
		
		c.connection.Write(encryptedToken)
		fmt.Println("Secret token sent to server!")
	} else {
		_, err := c.connection.Write([]byte(message))
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) exchangeKeys() error {
	// Receive public key from server
	keyBuffer := make([]byte, c.BufferSize)
	n, err := c.connection.Read(keyBuffer)
	if err != nil {
		return err
	}
	serverPublicKeyPEM := keyBuffer[:n]
	block, _ := pem.Decode(serverPublicKeyPEM)
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}
	var ok bool
	c.serverPublicKey, ok = pubInterface.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("could not cast public key to *rsa.PublicKey")
	}
	fmt.Println("Received server public key:" + string(serverPublicKeyPEM) + "\n")

	// Erzeugung des AES-Schlüssels und IVs
	c.aesKey = make([]byte, 32) // 256-bit AES key
	c.aesIV = make([]byte, 16)  // AES IV
	rand.Read(c.aesKey)
	rand.Read(c.aesIV)
	fmt.Println("Generated AES Key:", c.aesKey)
	fmt.Println("Generated AES IV:", c.aesIV)

	// Verschlüsseln des AES-Schlüssels und IVs mit dem RSA-Schlüssel des Servers
	encryptedKeyIV, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, c.serverPublicKey, append(c.aesIV, c.aesKey...), nil)
	if err != nil {
		return err
	}
	// Senden des verschlüsselten AES-Schlüssels und IVs an den Server
	_, err = c.connection.Write(encryptedKeyIV)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Connect() error {
	var err error
	c.connection, err = net.Dial("tcp", c.address)
	if err != nil {
		return err
	}

	go c.receive()

	// Wait for OK message from server
	// Mutex to wait for event to be handled
	onmsgmutex := sync.Mutex{}
	onmsgmutex.Lock()
	var onmsg func(message string)
	onmsg = func(message string) {
		if message != "OK" {
			c.disconnect()
		}
		fmt.Println("Received message from server:", message)
		c.RemoveEventHandler("message", onmsg)
		onmsgmutex.Unlock()
	}
	c.AddEventHandler("message", onmsg)



	if c.encryptionEnabled {
		err := c.exchangeKeys()
		if err != nil {
			return err
		}
	}





	onmsgmutex.Lock()
	defer onmsgmutex.Unlock()
	// Send secret token to server
	if err := c.Send(c.Secret_token); err != nil {
		return err
	}

	// Start ping loop
	go c.sendPing()

	return nil
}

func (c *Client) disconnect() {
	c.connection.Close()
	c.emit("disconnected", "Disconnected from the server.")
}

func (c *Client) emit(eventType, message string) {
	c.eventLock.Lock()
	defer c.eventLock.Unlock()
	if handlers, ok := c.eventHandlers[eventType]; ok {
		for _, handler := range handlers {
			go handler(message)
		}
	}
}

func (c *Client) AddEventHandler(eventType string, handler func(message string)) {
	c.eventLock.Lock()
	defer c.eventLock.Unlock()
	c.eventHandlers[eventType] = append(c.eventHandlers[eventType], handler)
}

func (c *Client) RemoveEventHandler(eventType string, handlerToRemove func(message string)) {
	c.eventLock.Lock()
	defer c.eventLock.Unlock()
	if handlers, ok := c.eventHandlers[eventType]; ok {
		for i, handler := range handlers {
			if fmt.Sprintf("%p", handler) == fmt.Sprintf("%p", handlerToRemove) {
				c.eventHandlers[eventType] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
	}
}

// Receve messages from the server
func (c *Client) receive() {
	for {
		select {
		case <-c.StopChan:
			return
		default:
			messageBuffer := make([]byte, c.BufferSize)
			n, err := c.connection.Read(messageBuffer)
			if err != nil {
				c.emit("timeout", "Connection timed out.")
				return
			}
			message := messageBuffer[:n]
			if c.encryptionEnabled {
				blockCipher, err := aes.NewCipher(c.aesKey)
				if err != nil {
					fmt.Println("Failed to create AES cipher:", err)
					return
				}
				stream := cipher.NewCFBDecrypter(blockCipher, c.aesIV)
				decryptedMessage := make([]byte, len(message))
				stream.XORKeyStream(decryptedMessage, message)
				message = decryptedMessage
			}

			if string(message) == "PONG" {
				c.emit("ping", string(message))
				continue
			}

			c.emit("message", string(message))
		}
	}
}

// Will send a ping every PingTimeIntervall seconds
func (c *Client) sendPing() {
	for {
		select {
		case <-c.StopChan:
			return
		default:
			time.Sleep(c.PingTimeIntervall)
			err := c.Send("PING")
			if err != nil {
				fmt.Println("Failed to send ping:", err)
				return
			}
		}
	}
}

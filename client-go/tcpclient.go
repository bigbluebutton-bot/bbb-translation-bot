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

type status int
const (
	CONNECTED status = iota
	CONNECTING
	DISCONNECTED
	DISCONNECTING
)

type TCPclient struct {
	address           string
	connection        net.Conn
	msgSendLock       sync.Mutex
	aesKey            []byte
	aesIV             []byte
	running           bool
	encryptionEnabled bool
	serverPublicKey   *rsa.PublicKey
	BufferSize        int
	PingTimeIntervall time.Duration
	StopChan          chan bool

	Secret_token string

	status status

	messageEvent      *Event
	connectedEvent    *Event
	disconnectedEvent *Event
	timeoutEvent      *Event
	pingEvent         *Event

	messageEventQueue *Event
}

func NewTCPclient(addr string, encryption bool) *TCPclient {
	return &TCPclient{
		address:           addr,
		connection:        nil,
		msgSendLock:       sync.Mutex{},
		aesKey:            nil,
		aesIV:             nil,
		running:           false,
		encryptionEnabled: encryption,
		serverPublicKey:   nil,
		BufferSize:        1024,
		PingTimeIntervall: 4 * time.Second,
		StopChan:          make(chan bool),

		Secret_token: "",

		status: DISCONNECTED,

		messageEvent:      NewEvent(),
		connectedEvent:    NewEvent(),
		disconnectedEvent: NewEvent(),
		timeoutEvent:      NewEvent(),
		pingEvent:         NewEvent(),

		messageEventQueue: NewEvent(),
	}
}

func (c *TCPclient) GetAESkey() []byte {
	return c.aesKey
}

func (c *TCPclient) GetAESiv() []byte {
	return c.aesIV
}

func (c *TCPclient) Send(message string) error {
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
	} else {
		_, err := c.connection.Write([]byte(message))
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *TCPclient) exchangeKeys() error {
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

func (c *TCPclient) Connect() error {
	c.status = CONNECTING
	c.StopChan = make(chan bool)

	var err error
	c.connection, err = net.Dial("tcp", c.address)
	if err != nil {
		return err
	}

	c.running = true

	go c.receive()

	// Wait for OK message from server
	// Mutex to wait for event to be handled
	onmsgmutex := sync.Mutex{}
	onmsgmutex.Lock()
	var onmsg func(message string)
	onmsg = func(message string) {
		if message != "OK" {
			c.Close()
		}
		fmt.Println("Received message from server:", message)
		c.messageEvent.Remove(onmsg)
		onmsgmutex.Unlock()
	}
	c.messageEvent.Add(onmsg)

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


	// Add messageEventQueue to messageEvent
	c.messageEvent = c.messageEventQueue
	c.messageEventQueue = NewEvent()

	c.status = CONNECTED

	return nil
}

func (c *TCPclient) Close() {
	c.status = DISCONNECTING
	if !c.running {
		return
	}
	c.running = false
	close(c.StopChan)
	c.connection.Close()
	c.disconnectedEvent.Emit("Disconnected from the server.")
	c.status = DISCONNECTED
}

// Receve messages from the server
func (c *TCPclient) receive() {
	for {
		select {
		case <-c.StopChan:
			return
		default:
			messageBuffer := make([]byte, c.BufferSize)
			n, err := c.connection.Read(messageBuffer)
			if err != nil {
				c.timeoutEvent.Emit("Connection timed out.")
				c.Close()
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
				c.pingEvent.Emit(string(message))
				continue
			}

			fmt.Println("MSG", string(message))

			c.messageEvent.Emit(string(message))
		}
	}
}

// Will send a ping every PingTimeIntervall seconds
func (c *TCPclient) sendPing() {
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

func (c *TCPclient) OnConnected(handler func(message string)) {
	c.connectedEvent.Add(handler)
}

func (c *TCPclient) RemoveOnConnected(handler func(message string)) {
	c.connectedEvent.Remove(handler)
}

func (c *TCPclient) OnMessage(handler func(message string)) {
	if c.status != CONNECTED {
		c.messageEventQueue.Add(handler)
	} else {
		c.messageEvent.Add(handler)
	}
}

func (c *TCPclient) RemoveOnMessage(handler func(message string)) {
	c.messageEventQueue.Remove(handler)
	c.messageEvent.Remove(handler)
}

func (c *TCPclient) OnDisconnected(handler func(message string)) {
	c.disconnectedEvent.Add(handler)
}

func (c *TCPclient) RemoveOnDisconnected(handler func(message string)) {
	c.disconnectedEvent.Remove(handler)
}

func (c *TCPclient) OnTimeout(handler func(message string)) {
	c.timeoutEvent.Add(handler)
}

func (c *TCPclient) RemoveOnTimeout(handler func(message string)) {
	c.timeoutEvent.Remove(handler)
}

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"net"
)

type UDPclient struct {
	serverAddr string
	conn       *net.UDPConn
	aesKey     []byte
	aesIV      []byte
	Encrypted  bool

	messageEvent      *Event
	connectedEvent    *Event
	disconnectedEvent *Event
}

func NewUDPclient(serverAddr string, encrypted bool, aeskey, aesiv []byte) *UDPclient {
	return &UDPclient{
		serverAddr: serverAddr,
		aesKey:     aeskey,
		aesIV:      aesiv,
		Encrypted:  encrypted,

		messageEvent:      NewEvent(),
		connectedEvent:    NewEvent(),
		disconnectedEvent: NewEvent(),
	}
}

func (c *UDPclient) Connect() error {
	udpAddr, err := net.ResolveUDPAddr("udp", c.serverAddr)
	if err != nil {
		return fmt.Errorf("Error resolving address: %v", err)
	}

	c.conn, err = net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("Error dialing: %v", err)
	}
	return nil
}

func (c *UDPclient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *UDPclient) SendMessage(message []byte) error {
	if c.Encrypted {
		var err error
		message, err = c.encryptMessage(message)
		if err != nil {
			return fmt.Errorf("Error encrypting message: %v", err)
		}
	}

	_, err := c.conn.Write(message)
	if err != nil {
		return fmt.Errorf("Error sending message: %v", err)
	}

	return nil
}

func (c *UDPclient) encryptMessage(message []byte) ([]byte, error) {
	blockCipher, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(blockCipher, c.aesIV)
	encryptedMessage := make([]byte, len(message))
	stream.XORKeyStream(encryptedMessage, message)
	return encryptedMessage, nil
}

func (c *UDPclient) OnConnected(handler func(message string)) {
	c.connectedEvent.Add(handler)
}

func (c *UDPclient) RemoveOnConnected(handler func(message string)) {
	c.connectedEvent.Remove(handler)
}

func (c *UDPclient) OnMessage(handler func(message string)) {
	c.messageEvent.Add(handler)
}

func (c *UDPclient) RemoveOnMessage(handler func(message string)) {
	c.messageEvent.Remove(handler)
}

func (c *UDPclient) OnDisconnected(handler func(message string)) {
	c.disconnectedEvent.Add(handler)
}

func (c *UDPclient) RemoveOnDisconnected(handler func(message string)) {
	c.disconnectedEvent.Remove(handler)
}

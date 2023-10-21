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
}

func NewUDPclient(serverAddr string, encrypted bool, aeskey, aesiv []byte) *UDPclient {
	return &UDPclient{
		serverAddr: serverAddr,
		aesKey:     aeskey,
		aesIV:      aesiv,
		Encrypted:  encrypted,
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
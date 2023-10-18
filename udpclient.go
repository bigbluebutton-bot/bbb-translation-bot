package main

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"net"
	"os"
)


func main() {
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


	client := NewUDPclient("localhost:5001", aesKey, aesIV)


	err := client.Connect()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer client.Close()

	client.Encrypted = true // setting encryption to true
	err = client.SendMessage([]byte("Hello, encrypted server!"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Message sent to server.")
}










type UDPclient struct {
	serverAddr string
	conn       *net.UDPConn
	aesKey     []byte
	aesIV      []byte
	Encrypted  bool
}

func NewUDPclient(serverAddr string, aeskey, aesiv []byte) *UDPclient {
	return &UDPclient{
		serverAddr: serverAddr,
		aesKey:     aeskey,
		aesIV:      aesiv,
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
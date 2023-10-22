package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
)

type StreamClient struct {
	tcpClient  *TCPclient
	udpClient  *UDPclient
	serverHost string
	serverPort int

	connectedEvent *Event
}

func NewStreamClient(host string, port int, useEncryption bool, secretToken string) *StreamClient {
	tcpClient := NewTCPclient(fmt.Sprintf("%s:%d", host, port), useEncryption)
	tcpClient.Secret_token = secretToken

	return &StreamClient{
		tcpClient:  tcpClient,
		serverHost: host,
		serverPort: port,

		connectedEvent: NewEvent(),
	}
}

type messageTypeStruct struct {
	Type string `json:"type"`
}

type MessageType int

const (
	MESSAGE MessageType = iota
	INIT_UDPADDRESS
)

func (sc *StreamClient) getMessageType(message string) (MessageType, error) {
	var msgtype messageTypeStruct
	err := json.Unmarshal([]byte(message), &msgtype)
	if err != nil {
		return 0, err
	}

	switch msgtype.Type {
	case "msg":
		return MESSAGE, nil
	case "init_udpaddr":
		return INIT_UDPADDRESS, nil
	}

	return 0, errors.New("unknown message type")
}

func (sc *StreamClient) Connect() error {

	// Mutex to wait until the server has sent the UDP address
	onmsgmutex := sync.Mutex{}
	onmsgmutex.Lock()

	var init_udpserver func(message string)
	init_udpserver = func(message string) {
		msgtype, err := sc.getMessageType(message)
		if err != nil {
			fmt.Println("Failed to get message type:", err)
			return
		}

		fmt.Println("Message type:", msgtype)

		if msgtype == INIT_UDPADDRESS {
			type udpAddrStruct struct {
				Msg struct {
					UDP struct {
						Host       string `json:"host"`
						Port       int    `json:"port"`
						Encryption bool   `json:"encryption"`
					} `json:"udp"`
				} `json:"msg"`
			}
			var udpAddr udpAddrStruct
			err := json.Unmarshal([]byte(message), &udpAddr)
			if err != nil {
				fmt.Println("Failed to unmarshal UDP address:", err)
				sc.Close()
				return
			}

			udpClient := NewUDPclient(udpAddr.Msg.UDP.Host+":"+strconv.Itoa(udpAddr.Msg.UDP.Port), udpAddr.Msg.UDP.Encryption, sc.tcpClient.GetAESkey(), sc.tcpClient.GetAESiv())
			err = udpClient.Connect()
			if err != nil {
				fmt.Println("Failed to connect to UDP server:", err)
				sc.Close()
				return
			}

			sc.udpClient = udpClient
		}

		sc.tcpClient.RemoveOnMessage(init_udpserver)

		onmsgmutex.Unlock()
	}

	sc.tcpClient.OnMessage(init_udpserver)

	err := sc.tcpClient.Connect()
	if err != nil {
		return err
	}

	onmsgmutex.Lock()
	defer onmsgmutex.Unlock()

	sc.connectedEvent.Emit("connected")

	sc.OnTimeout(func(message string) {
		sc.Close()
	})
	sc.OnDisconnected(func(message string) {
		sc.Close()
	})

	return nil
}

func (sc *StreamClient) SendTCPMessage(msg string) error {
	return sc.tcpClient.Send(msg)
}

func (sc *StreamClient) SendUDPMessage(data []byte) error {
	if sc.udpClient == nil {
		return fmt.Errorf("UDP client has not been initialized")
	}
	return sc.udpClient.SendMessage(data)
}

func (sc *StreamClient) Write(p []byte) (int, error) {
	err := sc.SendUDPMessage(p)
	return len(p), err
}

func (sc *StreamClient) Close() {
	if sc.tcpClient != nil {
		sc.tcpClient.Close()
	}
	if sc.udpClient != nil {
		sc.udpClient.Close()
	}
}

func (sc *StreamClient) OnConnected(handler func(message string)) {
	sc.connectedEvent.Add(handler)
}

func (sc *StreamClient) RemoveOnConnected(handler func(message string)) {
	sc.connectedEvent.Remove(handler)
}

func (sc *StreamClient) OnTCPMessage(handler func(message string)) {
	sc.tcpClient.OnMessage(handler)
}

func (sc *StreamClient) RemoveOnTCPMessage(handler func(message string)) {
	sc.tcpClient.RemoveOnMessage(handler)
}

// on disconnected
func (sc *StreamClient) OnDisconnected(handler func(message string)) {
	sc.tcpClient.OnDisconnected(handler)
}

// remove on disconnected
func (sc *StreamClient) RemoveOnDisconnected(handler func(message string)) {
	sc.tcpClient.RemoveOnDisconnected(handler)
}

// on timeout
func (sc *StreamClient) OnTimeout(handler func(message string)) {
	sc.tcpClient.OnTimeout(handler)
}

// remove on timeout
func (sc *StreamClient) RemoveOnTimeout(handler func(message string)) {
	sc.tcpClient.RemoveOnTimeout(handler)
}

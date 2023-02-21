package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

//connect to a the python server over a socket connection
func main() {
    conn, err := net.Dial("tcp", "localhost:5000")
    if err != nil {
        log.Fatal(err)
    }
    //send a message to the server
    fmt.Fprintf(conn, "Hello, server!\n")
    //receive a message from the server
    message, _ := bufio.NewReader(conn).ReadString('\n')
    fmt.Print("Message from server: " + message)
}
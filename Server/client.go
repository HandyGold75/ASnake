package main

import (
	"bufio"
	"fmt"
	"net"
)

func someClient() {
	tcp, err := net.ResolveTCPAddr("tcp", "localhost:17530")
	if err != nil {
		fmt.Printf("Client | Error: %v\r\n", err)
		return
	}

	conn, err := net.DialTCP("tcp", nil, tcp)
	if err != nil {
		fmt.Printf("Client | Error: %v\r\n", err)
		return
	}
	defer conn.Close()

	_, err = conn.Write([]byte("Join\n"))
	if err != nil {
		fmt.Printf("Client | Error: %v\r\n", err)
		return
	}

	msg, err := bufio.NewReader(conn).ReadString('\n') // err EOF
	if err != nil {
		fmt.Printf("Client | Error: %v\r\n", err)
		return
	}
	fmt.Printf("Client | Received: %v\r\n", msg)
}

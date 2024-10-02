package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
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

	reader := bufio.NewReader(conn)

	msg, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, net.ErrClosed) {
			return
		}
		fmt.Printf("Client | Error (%v): %v\r\n", msg, err)
		return
	}
	msg = strings.ReplaceAll(msg, "\n", "")
	fmt.Printf("Client | Received: %v\r\n", msg)
	if msg != "Accept" {
		return
	}

	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			fmt.Printf("Client | Error (%v): %v\r\n", msg, err)
			break
		}
		msg = strings.ReplaceAll(msg, "\n", "")

		fmt.Printf("Client | Received: %v\r\n", msg)
		// time.Sleep(time.Second)
	}
}

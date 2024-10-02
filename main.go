package main

import (
	"ASnake/game"
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"golang.org/x/term"
)

var (
	originalTrm = &term.State{}

	onKeyEvent = func([]byte) {}
	stopping   = false
)

func listenKeys() {
	defer term.Restore(int(os.Stdin.Fd()), originalTrm)

	for !stopping {
		in := make([]byte, 3)
		_, err := os.Stdin.Read(in)
		if err != nil {
			panic(err)
		}
		onKeyEvent(in)
	}

}

func getConnection() (*net.TCPConn, error) {
	tcp, err := net.ResolveTCPAddr("tcp", "localhost:17530")
	if err != nil {
		fmt.Printf("Client | Error: %v\r\n", err)
		return &net.TCPConn{}, err
	}

	conn, err := net.DialTCP("tcp", nil, tcp)
	if err != nil {
		fmt.Printf("Client | Error: %v\r\n", err)
		return &net.TCPConn{}, err
	}
	defer conn.Close()

	_, err = conn.Write([]byte("Join\n"))
	if err != nil {
		fmt.Printf("Client | Error: %v\r\n", err)
		return &net.TCPConn{}, err
	}

	reader := bufio.NewReader(conn)

	msg, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, net.ErrClosed) {
			return &net.TCPConn{}, err
		}
		fmt.Printf("Client | Error (%v): %v\r\n", msg, err)
		return &net.TCPConn{}, err
	}
	msg = strings.ReplaceAll(msg, "\n", "")
	fmt.Printf("Client | Received: %v\r\n", msg)
	if msg != "Accept" {
		return &net.TCPConn{}, err
	}

	return conn, nil
}

func main() {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	originalTrm = oldState
	defer func() { stopping = true; term.Restore(int(os.Stdin.Fd()), originalTrm) }()

	gm, err := game.NewGame(originalTrm)
	if err != nil {
		panic(err)
	}
	// mn, err := menu.NewMenu(gm)
	// if err != nil {
	// 	panic(err)
	// }

	con, err := getConnection()
	if err != nil {
		panic(err)
	}
	gm.Config.Connection = con

	go listenKeys()

	// onKeyEvent = mn.HandleInput
	// if out := mn.Start(); out == "Start" {
	// 	onKeyEvent = gm.HandleInput
	// 	gm.Start()
	// }
	gm.Start()
}

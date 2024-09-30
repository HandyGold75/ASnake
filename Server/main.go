package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
)

type (
	Server struct {
		IP    string
		Port  uint16
		Pools []Pool
	}
)

var (
	maxPlayers = 4
)

func NewServer(ip string, port uint16) *Server {
	return &Server{
		IP:    ip,
		Port:  port,
		Pools: []Pool{},
	}
}

func (sv *Server) Run() error {
	listener, err := net.Listen("tcp", sv.IP+":"+strconv.FormatUint(uint64(sv.Port), 10))
	if err != nil {
		return err
	}
	defer listener.Close()
	fmt.Println("Listening on: ")

	for {
		con, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			con.Close()
			fmt.Printf("%s | Error: %v\r\n", con.RemoteAddr().String(), err)
			continue
		}
		go sv.handleConnection(con)
	}
}

func (sv *Server) handleConnection(con net.Conn) {
	fmt.Printf("Serving %s\n", con.RemoteAddr().String())

	msg, err := bufio.NewReader(con).ReadString('\n')
	if err != nil {
		fmt.Printf("Server | Error: %v\r\n", err)
		con.Close()
	}
	fmt.Printf("%s | Receive: %v\r\n", con.RemoteAddr().String(), msg)
	if string(msg) != "Join" {
		con.Close()
	}

	fmt.Printf("%s | Send: %v\r\n", con.RemoteAddr().String(), "Accept")
	con.Write([]byte("Accept\n"))

	for _, pool := range sv.Pools {
		if len(pool.Clients) >= maxPlayers && pool.Status == "waiting" {
			continue
		}

		pool.Clients = append(pool.Clients, *NewClient(con))
		return
	}

	pool := NewPool()
	pool.Clients = append(pool.Clients, *NewClient(con))
	sv.Pools = append(sv.Pools, *pool)
}

func main() {
	go NewServer("127.0.0.1", 17530).Run()

	someClient()

	<-make(chan bool)
}

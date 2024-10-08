package tcp

import (
	"ASnake/server/pool"
	"bufio"
	"fmt"
	"net"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

type (
	Server struct {
		IP    string
		Port  uint16
		Pools []*pool.Pool
	}
)

var (
	MaxClients = 2
)

func NewServer(ip string, port uint16) *Server {
	return &Server{
		IP:    ip,
		Port:  port,
		Pools: []*pool.Pool{},
	}
}

func (sv *Server) Run() error {
	listener, err := net.Listen("tcp", sv.IP+":"+strconv.FormatUint(uint64(sv.Port), 10))
	if err != nil {
		return err
	}
	defer listener.Close()
	fmt.Printf("\033[2K\rListening | " + sv.IP + ":" + strconv.FormatUint(uint64(sv.Port), 10) + "\n")

	go func() {
		for {
			clientLen := 0
			toRemove := []int{}
			for i, pl := range sv.Pools {
				if pl.Status == "stopped" {
					toRemove = append(toRemove, i)
					continue
				}
				clientLen += len(pl.Clients)
			}

			sort.Sort(sort.Reverse(sort.IntSlice(toRemove)))
			for _, i := range toRemove {
				sv.Pools = slices.Delete(sv.Pools, i, i+1)
			}

			fmt.Printf("\033[2K\rStats     | Pools: %v   Clients: %v", len(sv.Pools), clientLen)
			time.Sleep(time.Second * 1)
		}
	}()

	for {
		con, err := listener.Accept()
		if err != nil {
			con.Close()
			continue
		}
		go sv.handleConnection(con)
	}
}

func (sv *Server) handleConnection(con net.Conn) {
	fmt.Printf("\033[2K\rServing   | %s", con.RemoteAddr().String())
	defer fmt.Println()

	msg, err := bufio.NewReader(con).ReadString('\n')
	if err != nil {
		con.Close()
		return
	}
	msg = strings.ReplaceAll(msg, "\n", "")
	if string(msg) != "Join" {
		con.Close()
		return
	}

	con.Write([]byte("Accept\n"))

	for _, pl := range sv.Pools {
		if len(pl.Clients) >= MaxClients || pl.Status != "waiting" {
			continue
		}

		fmt.Printf("\033[2K\rAccepted  | %s", con.RemoteAddr().String())
		pl.Clients = append(pl.Clients, pool.NewClient(con, -1))
		return
	}

	pl, err := pool.NewPool(MaxClients)
	if err != nil {
		con.Close()
		return
	}

	fmt.Printf("\033[2K\rAccepted  | %s", con.RemoteAddr().String())
	pl.Clients = append(pl.Clients, pool.NewClient(con, -1))
	sv.Pools = append(sv.Pools, pl)
}

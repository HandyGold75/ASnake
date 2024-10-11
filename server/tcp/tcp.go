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

	"github.com/HandyGold75/GOLib/logger"
)

type (
	Server struct {
		Port  uint16
		Pools []*pool.Pool
		Lgr   *logger.Logger
	}
)

var (
	MaxClients = 4
)

func NewServer(ip string, port uint16) *Server {
	lgr := logger.New("ASnake.log")
	lgr.UseSeperators = false
	lgr.CharCountPerMsg = 16

	sv := &Server{
		Port:  port,
		Pools: []*pool.Pool{},
		Lgr:   lgr,
	}

	lgr.MessageCLIHook = func(msg string) {
		clientLen := 0
		for _, pl := range sv.Pools {
			if pl.Status == "stopped" {
				continue
			}
			clientLen += len(pl.Clients)
		}

		fmt.Printf("["+time.Now().Format(time.DateTime)+"] %-"+strconv.Itoa(sv.Lgr.CharCountVerbosity)+"v Pools: %v | Clients: %v      \r", "stats", len(sv.Pools), clientLen)
	}

	return sv
}

func (sv *Server) Run() error {
	listener, err := net.Listen("tcp", ":"+strconv.FormatUint(uint64(sv.Port), 10))
	if err != nil {
		return err
	}
	defer listener.Close()
	sv.Lgr.Log("medium", "Listening", ":"+strconv.FormatUint(uint64(sv.Port), 10))

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

			fmt.Printf("["+time.Now().Format(time.DateTime)+"] %-"+strconv.Itoa(sv.Lgr.CharCountVerbosity)+"v Pools: %v | Clients: %v      \r", "stats", len(sv.Pools), clientLen)
			time.Sleep(time.Second)
		}
	}()

	for {
		con, err := listener.Accept()
		if err != nil {
			sv.Lgr.Log("high", "Error", err)
			con.Close()
			continue
		}
		go sv.handleConnection(con)
	}
}

func (sv *Server) handleConnection(con net.Conn) {
	sv.Lgr.Log("low", "Serving", con.RemoteAddr().String())

	msg, err := bufio.NewReader(con).ReadString('\n')
	if err != nil {
		sv.Lgr.Log("medium", "Rejected", con.RemoteAddr().String())
		con.Close()
		return
	}
	msg = strings.ReplaceAll(msg, "\n", "")
	if string(msg) != "Join" {
		sv.Lgr.Log("medium", "Rejected", con.RemoteAddr().String())
		con.Close()
		return
	}

	con.Write([]byte("Accept\n"))

	for _, pl := range sv.Pools {
		if len(pl.Clients) >= MaxClients || (pl.Status != "initialized" && pl.Status != "waiting" && pl.Status != "started") {
			continue
		}

		sv.Lgr.Log("medium", "Accepted", con.RemoteAddr().String())
		pl.AddClient(&con)
		return
	}

	pl, err := pool.NewPool(sv.Lgr)
	if err != nil {
		sv.Lgr.Log("high", "Error", err)
		con.Close()
		return
	}

	sv.Lgr.Log("medium", "Accepted", con.RemoteAddr().String())
	pl.AddClient(&con)
	sv.Pools = append(sv.Pools, pl)
}

package main

import (
	"ASnake/game"
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"
)

type (
	Server struct {
		IP    string
		Port  uint16
		Pools []*Pool
	}

	Client struct {
		con net.Conn
		id  int
	}

	Pool struct {
		Clients []*Client
		Game    *game.Game
		Status  string
	}
)

var (
	maxPlayers = 4
)

func NewPool() (*Pool, error) {
	gm, err := game.NewGameNoTUI()
	if err != nil {
		return &Pool{}, err
	}
	gm.Config.PeaSpawnDelay = 2
	gm.Config.PeaSpawnLimit = 12
	gm.Config.PeaStartCount = 4

	p := &Pool{
		Clients: []*Client{},
		Game:    gm,
		Status:  "initialized",
	}

	go p.poolHandler()

	return p, nil
}

func (pool *Pool) poolHandler() {
	defer pool.stop()

	pool.wait()
	pool.start()
	pool.loop()
}

func (pool *Pool) wait() {
	queEndTime := time.Now().Add(time.Minute)

	pool.Status = "waiting"
	for pool.Status == "waiting" {
		if len(pool.Clients) == 0 {
			queEndTime = time.Now().Add(time.Minute)
		}

		if len(pool.Clients) >= maxPlayers || time.Now().After(queEndTime) {
			break
		}
		for _, client := range pool.Clients {
			client.con.Write([]byte(pool.Status + "\n"))
		}

		time.Sleep(time.Second * time.Duration(maxPlayers-len(pool.Clients)))
	}
}

func (pool *Pool) start() {
	pool.Status = "starting"

	pool.Game.State.Players = make([]game.Player, len(pool.Clients))
	for i, client := range pool.Clients {
		client.id = i
		pool.Game.State.Players[client.id] = game.Player{
			Crd: game.Cord{X: int(pool.Game.Screen.CurX / 2), Y: int(pool.Game.Screen.CurY / (len(pool.Clients) + client.id))},
			Dir: "right", CurDir: "right",
			TailCrds: []game.Cord{},
		}

		update := game.FirstUpdatePacket{
			ClientId:      client.id,
			Players:       pool.Game.State.Players,
			PeaCrds:       pool.Game.State.PeaCrds,
			StartTime:     pool.Game.State.StartTime,
			PlusOneActive: pool.Game.State.PlusOneActive,
			TpsTracker:    pool.Game.State.TpsTracker,
			CurX:          pool.Game.Screen.CurX, CurY: pool.Game.Screen.CurY,
		}
		data, err := json.Marshal(update)
		if err != nil {
			pool.Status = "stopping"
			return
		}

		client.con.Write(append(data, '\n'))
	}

	pool.inputHandler()

	for i := 0; i < pool.Game.Config.PeaStartCount; i++ {
		pool.Game.SpawnPea()
	}

	pool.Game.State.StartTime = time.Now()

	pool.Status = "started"
}

func (pool *Pool) stop() {
	pool.Status = "stopping"
	for _, client := range pool.Clients {
		client.con.Close()
	}
	pool.Clients = []*Client{}
	pool.Status = "stopped"
}

func (pool *Pool) loop() {
	updateFramePlayer := max(1, pool.Game.Config.TargetTPS-pool.Game.Config.PlayerSpeed)
	updateFramePea := max(1, pool.Game.Config.PeaSpawnDelay*pool.Game.Config.TargetTPS)
	updateFramePlusOne := max(1, pool.Game.Config.PlusOneDelay*pool.Game.Config.TargetTPS)

	for i := 1; pool.Status == "started"; i++ {
		t := time.Now()
		doSend := false

		if i%updateFramePlayer == 0 {
			doSend = true
			isOneAlive := false
			for _, client := range pool.Clients {
				if pool.Game.State.Players[client.id].IsGameOver {
					continue
				}
				isOneAlive = true
				pool.Game.UpdatePlayer(client.id)
			}
			if !isOneAlive {
				break
			}
		}

		if pool.Game.State.PlusOneActive && i%updateFramePlusOne == 0 {
			pool.Game.State.PlusOneActive = false
		}

		if i%updateFramePea == 0 {
			doSend = true
			pool.Game.State.PeaCrds = slices.DeleteFunc(pool.Game.State.PeaCrds, func(cord game.Cord) bool {
				val, err := pool.Game.Screen.GetColRow(cord.X, cord.Y)
				return err != nil || val != pool.Game.Objects.Pea
			})

			if len(pool.Game.State.PeaCrds) < pool.Game.Config.PeaSpawnLimit {
				pool.Game.SpawnPea()
			}
		}

		if doSend {
			if err := pool.sendUpdate(); err != nil {
				fmt.Printf("Server | Error: %v\r\n", err)
			}
		}

		time.Sleep((time.Second / time.Duration(pool.Game.Config.TargetTPS)) - time.Now().Sub(t))
		pool.Game.State.TpsTracker = int(time.Second/time.Now().Sub(t)) + 1

	}
}

func (pool *Pool) sendUpdate() error {
	update := game.UpdatePacket{
		Players:       pool.Game.State.Players,
		PeaCrds:       pool.Game.State.PeaCrds,
		PlusOneActive: pool.Game.State.PlusOneActive,
		TpsTracker:    pool.Game.State.TpsTracker,
	}

	data, err := json.Marshal(update)
	if err != nil {
		return err
	}
	fmt.Printf("Server | Send: %v\r\n", update)

	for _, client := range pool.Clients {
		client.con.Write(append(data, '\n'))
	}

	return nil
}

func (pool *Pool) inputHandler() {
	for _, client := range pool.Clients {
		go func(cl *Client) {
			reader := bufio.NewReader(cl.con)
			for pool.Status != "stopping" && pool.Status != "stopped" {
				msg, err := reader.ReadString('\n')
				if err != nil {
					if errors.Is(err, net.ErrClosed) {
						break
					}
					fmt.Printf("Server | Error: %v\r\n", err)
					break
				}
				msg = strings.ReplaceAll(msg, "\n", "")

				if !slices.Contains([]string{"up", "right", "down", "left"}, msg) {
					fmt.Printf("Server | Error: invalid dir '%v'\r\n", msg)
					continue
				}

				pool.Game.State.Players[cl.id].Dir = msg
			}

			fmt.Printf("Closing %s\n", cl.con.RemoteAddr().String())
			cl.con.Close()
			pool.Game.State.Players[cl.id].IsGameOver = true
		}(client)
	}
}

func NewServer(ip string, port uint16) *Server {
	return &Server{
		IP:    ip,
		Port:  port,
		Pools: []*Pool{},
	}
}

func (sv *Server) Run() error {
	listener, err := net.Listen("tcp", sv.IP+":"+strconv.FormatUint(uint64(sv.Port), 10))
	if err != nil {
		return err
	}
	defer listener.Close()
	fmt.Println("Listening on: " + sv.IP + ":" + strconv.FormatUint(uint64(sv.Port), 10))

	for {
		con, err := listener.Accept()
		if err != nil {
			con.Close()
			fmt.Printf("Server | Error: %v\r\n", err)
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
		return
	}
	msg = strings.ReplaceAll(msg, "\n", "")
	if string(msg) != "Join" {
		fmt.Printf("Server | Error: %v\r\n", msg)
		con.Close()
		return
	}

	con.Write([]byte("Accept\n"))

	for _, pool := range sv.Pools {
		if len(pool.Clients) >= maxPlayers || pool.Status != "waiting" {
			continue
		}

		pool.Clients = append(pool.Clients, &Client{con: con, id: -1})

		return
	}

	pool, err := NewPool()
	if err != nil {
		fmt.Printf("Server | Error: %v\r\n", err)
		con.Close()
		return
	}

	pool.Clients = append(pool.Clients, &Client{con: con, id: -1})
	sv.Pools = append(sv.Pools, pool)
}

func main() {
	NewServer("127.0.0.1", 17530).Run()
}

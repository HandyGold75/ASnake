package main

import (
	"ASnake/game"
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"time"
)

type (
	updatePacket struct {
		curDir string
	}

	Client struct {
		con net.Conn
		id  int
	}

	config struct {
		TargetTPS                                                              int
		PlayerSpeed, PeaSpawnDelay, PeaSpawnLimit, PeaStartCount, PlusOneDelay int
	}

	Pool struct {
		Clients []*Client
		Game    *game.Game
		Config  config
		Status  string
	}
)

func NewClient(con net.Conn) *Client {
	return &Client{
		con: con,
		id:  0,
	}

}

func NewPool() (*Pool, error) {
	gm, err := game.NewGameNoTUI()
	if err != nil {
		return &Pool{}, err
	}

	p := &Pool{
		Clients: []*Client{},
		Game:    gm,
		Config: config{
			TargetTPS:     30,
			PlayerSpeed:   24,
			PeaSpawnDelay: 5,
			PeaSpawnLimit: 3,
			PeaStartCount: 1,
			PlusOneDelay:  1,
		},
		Status: "initialized",
	}

	go p.poolHandler()

	return p, nil
}

func (pool *Pool) poolHandler() {
	fmt.Println("Pool handler started")
	defer pool.stop()

	pool.wait()
	pool.start()
	pool.loop()
}

func (pool *Pool) wait() {
	queEndTime := time.Now().Add(time.Second * 5)

	pool.Status = "waiting"
	for pool.Status == "waiting" {
		if len(pool.Clients) == 0 {
			queEndTime = time.Now().Add(time.Minute / 4)
		}

		if len(pool.Clients) >= maxPlayers || time.Now().After(queEndTime) {
			break
		}
		for _, client := range pool.Clients {
			fmt.Println("Server | Send: " + pool.Status)
			client.con.Write([]byte(pool.Status + "\n"))
			time.Sleep(time.Second * time.Duration(maxPlayers-len(pool.Clients)))
		}
	}
}

func (pool *Pool) start() {
	pool.Status = "starting"

	pool.Game.State.Players = make([]game.Player, len(pool.Clients))
	for i, client := range pool.Clients {
		client.id = i
		pool.Game.State.Players[client.id] = game.Player{
			Crd: game.Cord{X: int(pool.Game.Screen.CurX / 2), Y: int(pool.Game.Screen.CurY / (len(pool.Clients) + 1))},
			Dir: "right", CurDir: "right",
			TailCrds: []game.Cord{},
		}
	}

	pool.inputHandler()

	for i := 0; i < pool.Game.Config.PeaStartCount; i++ {
		pool.Game.SpawnPea()
	}

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

	for i := 1; pool.Status == "started"; i++ {
		t := time.Now()

		if i%updateFramePlayer == 0 {
			for _, client := range pool.Clients {
				if pool.Game.State.Players[client.id].IsGameOver {
					continue
				}
				pool.Game.UpdatePlayer(client.id)
			}
		}

		if i%updateFramePea == 0 {
			pool.Game.State.PeaCrds = slices.DeleteFunc(pool.Game.State.PeaCrds, func(cord game.Cord) bool {
				val, err := pool.Game.Screen.GetColRow(cord.X, cord.Y)
				return err != nil || val != pool.Game.Objects.Pea
			})

			if len(pool.Game.State.PeaCrds) < pool.Game.Config.PeaSpawnLimit {
				pool.Game.SpawnPea()
			}
		}

		if err := pool.sendUpdate(); err != nil {
			fmt.Printf("Server | Error: %v\r\n", err)
		}

		time.Sleep((time.Second / time.Duration(pool.Game.Config.TargetTPS)) - time.Now().Sub(t))
		pool.Game.State.TpsTracker = int(time.Second/time.Now().Sub(t)) + 1

		pool.Game.State.FpsTracker = pool.Game.State.TpsTracker
	}
}

func (pool *Pool) sendUpdate() error {
	data, err := json.Marshal(pool.Game.State)
	if err != nil {
		return err
	}

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

				update := updatePacket{}
				err = json.Unmarshal([]byte(msg), &update)
				if err != nil {
					fmt.Printf("Server | Error: %v\r\n", err)
					continue
				}
				pool.Game.State.Players[cl.id].CurDir = update.curDir
				fmt.Printf("Server | Dir: %v\r\n", update.curDir)
			}

			fmt.Println("Server | Send: " + pool.Status)
			client.con.Write([]byte(pool.Status + "\n"))

			fmt.Printf("Closing %s\n", cl.con.RemoteAddr().String())
			cl.con.Close()
			pool.Clients = slices.DeleteFunc(pool.Clients, func(client *Client) bool { return client.id == cl.id })

		}(client)
	}
}

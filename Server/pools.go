package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type (
	Cord struct{ X, Y int }

	updatePacket struct {
		playerDir string
	}

	playerState struct {
		PlayerCrd Cord
		PlayerDir string
		TailCrds  []Cord
	}

	Client struct {
		con net.Conn
		playerState
	}

	stats struct {
		startTime time.Time
	}

	config struct {
		TargetTPS                                                              int
		PlayerSpeed, PeaSpawnDelay, PeaSpawnLimit, PeaStartCount, PlusOneDelay int
	}

	Pool struct {
		Clients []Client
		Stats   stats
		Config  config
		Status  string
	}
)

func NewClient(con net.Conn) *Client {
	return &Client{
		con: con,
		playerState: playerState{
			PlayerCrd: Cord{0, 0},
			PlayerDir: "",
			TailCrds:  []Cord{},
		},
	}

}

func NewPool() *Pool {
	p := &Pool{
		Clients: []Client{},
		Stats: stats{
			startTime: time.Now(),
		},
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

	return p
}

func (pool *Pool) poolHandler() {
	pool.Status = "waiting"

	for pool.Status != "stopping" && pool.Status != "stopped" {
		queEndTime := time.Now().Add(time.Minute * time.Duration(5))
		for _, client := range pool.Clients {
			client.con.Write([]byte("Waiting\n"))

			if len(pool.Clients) >= maxPlayers || time.Now().After(queEndTime) {
				break
			}
			time.Sleep(time.Second * time.Duration(maxPlayers-len(pool.Clients)))
		}
		pool.start()
	}
}

func (pool *Pool) start() {
	pool.Status = "starting"

	pool.inputHandler()
	pool.Stats.startTime = time.Now()

	pool.Status = "started"
}

func (pool *Pool) stop() {
	pool.Status = "stopping"

	for _, client := range pool.Clients {
		client.con.Close()
	}

	pool.Status = "stopped"
}

func (pool *Pool) inputHandler() {
	for _, client := range pool.Clients {
		go func(cl *Client) {
			for pool.Status != "stopping" && pool.Status != "stopped" {
				msg, err := bufio.NewReader(cl.con).ReadString('\n')
				if err != nil {
					fmt.Printf("Server | Error: %v\r\n", err)
				}

				tmp := updatePacket{}
				err = json.Unmarshal([]byte(msg), &tmp)
				if err != nil {
					fmt.Printf("Server | Error: %v\r\n", err)
					continue
				}
				cl.playerState.PlayerDir = tmp.playerDir
				fmt.Printf("Server | Dir: %v\r\n", cl.playerState.PlayerDir)
			}
		}(&client)
		client.con.Write([]byte("Start\n"))
	}
}

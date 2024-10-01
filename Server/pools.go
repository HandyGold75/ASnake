package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"slices"
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
		id  int8
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
		id:  0,
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

	fmt.Println("Pool handler started")

	for pool.Status != "stopping" && pool.Status != "stopped" {
		queEndTime := time.Now().Add(time.Minute * time.Duration(5))

		for pool.Status == "waiting" {
			if len(pool.Clients) >= maxPlayers || time.Now().After(queEndTime) {
				break
			}
			for _, client := range pool.Clients {
				fmt.Println("Server | Send: Waiting")
				client.con.Write([]byte("Waiting\n"))
				// fmt.Fprintf(client.con, "Waiting\n")
				// bufio.NewWriter(client.con).WriteString("Watinggg\n")
				time.Sleep(time.Second * time.Duration(maxPlayers-len(pool.Clients)))
			}
		}
		pool.start()
	}

	pool.stop()
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
	pool.Clients = []Client{}

	pool.Status = "stopped"
}

func (pool *Pool) inputHandler() {
	for i, client := range pool.Clients {
		client.id = int8(i)

		go func(cl *Client) {
			reader := bufio.NewReader(cl.con)
			for pool.Status != "stopping" && pool.Status != "stopped" {
				msg, err := reader.ReadString('\n')
				if err != nil {
					if errors.Is(err, net.ErrClosed) {
						break
					}
					fmt.Printf("Server | Error: %v\r\n", err)
					continue
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

			fmt.Printf("Closing %s\n", cl.con.RemoteAddr().String())
			cl.con.Close()
			pool.Clients = slices.DeleteFunc(pool.Clients, func(client Client) bool { return client.id == cl.id })

		}(&client)

		client.con.Write([]byte("Start\n"))
	}
}

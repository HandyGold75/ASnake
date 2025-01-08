package pool

import (
	"ASnake/client/game"
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/HandyGold75/GOLib/logger"
)

type (
	Pool struct {
		Clients    map[string]*net.Conn
		Game       *game.Game
		MaxClients int
		Status     string
		Lgr        *logger.Logger
	}
)

func NewPool(lgr *logger.Logger) (*Pool, error) {
	gm, err := game.NewServer()
	if err != nil {
		return &Pool{}, err
	}

	p := &Pool{
		Clients: map[string]*net.Conn{},
		Game:    gm,
		Status:  "initialized",
		Lgr:     lgr,
	}

	go p.poolHandler()

	return p, nil
}

func (pool *Pool) AddClient(con *net.Conn) {
	id := (*con).RemoteAddr().String()
	if pool.Status == "initialized" || pool.Status == "waiting" {
		pool.Clients[id] = con
		pool.inputHandler(pool.Clients[id])

	} else if pool.Status == "started" {
		cord := game.Cord{X: rand.IntN(pool.Game.Screen.CurX-1) + 1, Y: rand.IntN(pool.Game.Screen.CurY-1) + 1}
		for i := 1; i <= 100; i++ {
			if i == 100 {
				pool.Lgr.Log("high", "Error", "No space left for new player")
				(*con).Close()
				return
			}

			valid := true
			for y := -2; y < 3; y++ {
				for x := -2; x < 3; x++ {
					val, _ := pool.Game.Screen.GetColRow(cord.X+x, cord.Y+y)
					if val != pool.Game.Objects.Empty && val != pool.Game.Objects.PlusOne {
						valid = false
						break
					}
				}
				if !valid {
					break
				}
			}
			if !valid {
				cord = game.Cord{X: rand.IntN(pool.Game.Screen.CurX-1) + 1, Y: rand.IntN(pool.Game.Screen.CurY-1) + 1}
			}
			break
		}

		pool.Clients[id] = con
		pool.inputHandler(pool.Clients[id])

		pool.Game.State.Players[id] = game.Player{
			Crd: game.Cord{X: cord.X, Y: cord.Y},
			Dir: "right", CurDir: "right",
			TailCrds: []game.Cord{},
		}

		update := game.FirstUpdatePacket{
			ClientId:      id,
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

		(*pool.Clients[id]).Write(append(data, '\n'))

	} else {
		pool.Lgr.Log("high", "Error", "Attempting add on stopped pool")
		(*con).Close()
	}
}

func (pool *Pool) DelClient(con *net.Conn) {
	id := (*con).RemoteAddr().String()
	cl, ok := pool.Clients[id]
	if !ok {
		pool.Lgr.Log("high", "Error", "Unable to disconnect client '"+id+"'")
		return
	}

	pool.Lgr.Log("medium", "Disconnecting", id)
	(*cl).Close()
	if _, ok := pool.Game.State.Players[id]; ok {
		playerState := pool.Game.State.Players[id]
		playerState.IsGameOver = true
		pool.Game.State.Players[id] = playerState
	}
	delete(pool.Clients, id)
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

		if len(pool.Clients) >= 2 || time.Now().After(queEndTime) {
			break
		}
		for _, client := range pool.Clients {
			(*client).Write([]byte(pool.Status + "\n"))
		}

		time.Sleep(time.Second * 3)
	}
}

func (pool *Pool) start() {
	pool.Status = "starting"

	pool.Game.State.Players = make(map[string]game.Player, len(pool.Clients))
	i := 0
	for id, client := range pool.Clients {
		startY := int(pool.Game.Screen.CurY / 2)
		if i%2 == 0 {
			startY += i
		} else {
			startY -= (i + 1)
		}

		pool.Game.State.Players[id] = game.Player{
			Crd: game.Cord{X: int(pool.Game.Screen.CurX / 2), Y: startY},
			Dir: "right", CurDir: "right",
			TailCrds: []game.Cord{},
		}

		update := game.FirstUpdatePacket{
			ClientId:      id,
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

		(*client).Write(append(data, '\n'))
		i++
	}

	for i := 0; i < pool.Game.Config.PeaStartCount; i++ {
		pool.Game.SpawnPea()
	}

	pool.Game.State.StartTime = time.Now()

	pool.Status = "started"
}

func (pool *Pool) stop() {
	pool.Status = "stopping"
	for _, client := range pool.Clients {
		pool.DelClient(client)
	}
	pool.Status = "stopped"
}

func (pool *Pool) loop() {
	updateFramePlayer := max(1, pool.Game.Config.TargetTPS/pool.Game.Config.PlayerSpeed)
	updateFramePea := max(1, pool.Game.Config.PeaSpawnDelay*pool.Game.Config.TargetTPS)
	updateFramePlusOne := max(1, pool.Game.Config.PlusOneDelay*pool.Game.Config.TargetTPS)

	for i := 1; pool.Status == "started"; i++ {
		t := time.Now()
		doSend := false

		if i%updateFramePlayer == 0 {
			doSend = true
			isOneAlive := false
			for id := range pool.Clients {
				if pool.Game.State.Players[id].IsGameOver {
					continue
				}
				isOneAlive = true
				pool.Game.UpdatePlayer(id)
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
				pool.Lgr.Log("high", "Error", err)
			}
		}

		time.Sleep((time.Second / time.Duration(pool.Game.Config.TargetTPS)) - time.Now().Sub(t))
		pool.Game.State.TpsTracker = int(time.Second/time.Now().Sub(t)) + 1

	}
}

func (pool *Pool) sendUpdate() error {
	if slices.Contains(os.Args, "-d") || slices.Contains(os.Args, "-debug") {
		fmt.Println(pool.Game.Screen.Rows)
	}

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

	for _, client := range pool.Clients {
		(*client).Write(append(data, '\n'))
	}

	return nil
}

func (pool *Pool) inputHandler(con *net.Conn) {
	go func() {
		defer func() {
			if pool.Status != "stopping" && pool.Status != "stopped" {
				pool.DelClient(con)
			}
		}()

		reader := bufio.NewReader(*con)
		for pool.Status != "stopping" && pool.Status != "stopped" {
			msg, err := reader.ReadString('\n')
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					break
				}
				break
			}
			msg = strings.ReplaceAll(msg, "\n", "")

			if !slices.Contains([]string{"up", "right", "down", "left"}, msg) {
				continue
			}

			id := (*con).RemoteAddr().String()
			playerState := pool.Game.State.Players[id]
			playerState.Dir = msg
			pool.Game.State.Players[id] = playerState
		}
	}()
}

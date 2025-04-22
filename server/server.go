package server

import (
	"ASnake/game"
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
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
		IP         string
		Port       uint16
		MaxClients int
		Pools      []*Pool
		Lgr        *logger.Logger
	}

	Pool struct {
		Clients    map[string]*net.Conn
		Game       *game.Game
		MaxClients int
		Status     string
		Lgr        *logger.Logger
	}
)

func NewServer(ip string, port uint16, maxClients int) *Server {
	lgr := logger.New("ASnake.log")
	lgr.UseSeperators = false
	lgr.CharCountPerPart = 16

	sv := &Server{
		IP:         ip,
		Port:       port,
		MaxClients: maxClients,
		Pools:      []*Pool{},
		Lgr:        lgr,
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
	listener, err := net.Listen("tcp", sv.IP+":"+strconv.FormatUint(uint64(sv.Port), 10))
	if err != nil {
		return err
	}
	defer listener.Close()
	sv.Lgr.Log("medium", "Listening", sv.IP+":"+strconv.FormatUint(uint64(sv.Port), 10))

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
			_ = con.Close()
			continue
		}
		go func() {
			sv.Lgr.Log("low", "Serving", con.RemoteAddr().String())

			msg, err := bufio.NewReader(con).ReadString('\n')
			if err != nil {
				sv.Lgr.Log("medium", "Failed", con.RemoteAddr().String())
				_ = con.Close()
				return
			}
			msg = strings.ReplaceAll(msg, "\n", "")
			if string(msg) != "Join" {
				sv.Lgr.Log("medium", "Rejected", con.RemoteAddr().String())
				_, _ = con.Write([]byte("Rejected\n"))
				_ = con.Close()
				return
			}

			_, err = con.Write([]byte("Accept\n"))
			if err != nil {
				sv.Lgr.Log("medium", "Failed", con.RemoteAddr().String())
				_ = con.Close()
				return
			}

			for _, pl := range sv.Pools {
				if len(pl.Clients) >= sv.MaxClients || (pl.Status != "initialized" && pl.Status != "waiting" && pl.Status != "started") {
					continue
				}

				sv.Lgr.Log("medium", "Accepted", con.RemoteAddr().String())
				pl.AddClient(&con)
				return
			}

			pl, err := NewPool(sv.MaxClients, sv.Lgr)
			if err != nil {
				sv.Lgr.Log("high", "Error", err)
				_ = con.Close()
				return
			}

			sv.Lgr.Log("medium", "Accepted", con.RemoteAddr().String())
			pl.AddClient(&con)
			sv.Pools = append(sv.Pools, pl)
		}()
	}
}

func NewPool(maxClients int, lgr *logger.Logger) (*Pool, error) {
	gm, err := game.NewGame(true)
	if err != nil {
		return &Pool{}, err
	}

	gm.Config.PeaSpawnDelay = max(1, 5-maxClients)
	gm.Config.PeaSpawnLimit = 4 * maxClients
	gm.Config.PeaStartCount = 2 * maxClients

	p := &Pool{
		Clients: map[string]*net.Conn{},
		Game:    gm,
		Status:  "initialized",
		Lgr:     lgr,
	}

	go p.start()

	return p, nil
}

func (pool *Pool) AddClient(con *net.Conn) {
	id := (*con).RemoteAddr().String()
	if pool.Status == "initialized" || pool.Status == "waiting" {
		pool.Clients[id] = con
		go pool.clientHandler(pool.Clients[id])

	} else if pool.Status == "started" {
		cord := [2]int{rand.IntN(pool.Game.Screen.CurX-1) + 1, rand.IntN(pool.Game.Screen.CurY-1) + 1}
		for i := 1; i <= 100; i++ {
			if i == 100 {
				pool.Lgr.Log("high", "Error", "No space left for new player")
				_ = (*con).Close()
				return
			}

			valid := true
			for y := -2; y < 3; y++ {
				for x := -2; x < 3; x++ {
					val, _ := pool.Game.Screen.GetColRow(cord[0]+x, cord[1]+y)
					if val != game.ObjEmpty && val != game.ObjPlusOne {
						valid = false
						break
					}
				}
				if !valid {
					break
				}
			}
			if !valid {
				cord = [2]int{rand.IntN(pool.Game.Screen.CurX-1) + 1, rand.IntN(pool.Game.Screen.CurY-1) + 1}
				continue
			}
			break
		}

		pool.Clients[id] = con
		go pool.clientHandler(pool.Clients[id])

		pool.Game.State.Players[id] = game.Player{
			Crd: cord,
			Dir: "right", CurDir: "right",
			TailCrds: [][2]int{},
		}

		update := game.FirstUpdatePacket{
			ClientId:  id,
			StartTime: pool.Game.StartTime,
			MaxX:      pool.Game.Screen.MaxX, MaxY: pool.Game.Screen.MaxY,
			State: game.GameState{
				Players:       pool.Game.State.Players,
				PeaCrds:       pool.Game.State.PeaCrds,
				PlusOneActive: pool.Game.State.PlusOneActive,
				TpsTracker:    pool.Game.State.TpsTracker,
			},
		}
		data, err := json.Marshal(update)
		if err != nil {
			pool.DelClient(pool.Clients[id])
			return
		}

		_, err = (*pool.Clients[id]).Write(append(data, '\n'))
		if err != nil {
			pool.DelClient(pool.Clients[id])
			return
		}

	} else {
		pool.Lgr.Log("high", "Error", "Attempting add on stopped pool")
		_ = (*con).Close()
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
	_ = (*cl).Close()
	if _, ok := pool.Game.State.Players[id]; ok {
		playerState := pool.Game.State.Players[id]
		playerState.IsGameOver = true
		pool.Game.State.Players[id] = playerState
	}
	delete(pool.Clients, id)
}

func (pool *Pool) start() {
	defer func() {
		pool.Status = "stopping"
		for _, client := range pool.Clients {
			pool.DelClient(client)
		}
		pool.Status = "stopped"
	}()

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
			_, err := (*client).Write([]byte(pool.Status + "\n"))
			if err != nil {
				pool.DelClient(client)
			}
		}

		time.Sleep(time.Second * 3)
	}

	pool.Status = "starting"
	pool.Game.State.Players = make(map[string]game.Player, len(pool.Clients))
	pool.Game.StartTime = time.Now()

	i := 0
	for id, client := range pool.Clients {
		startY := int(pool.Game.Screen.CurY / 2)
		if i%2 == 0 {
			startY += i
		} else {
			startY -= (i + 1)
		}

		pool.Game.State.Players[id] = game.Player{
			Crd: [2]int{int(pool.Game.Screen.CurX / 2), startY},
			Dir: "right", CurDir: "right",
			TailCrds: [][2]int{},
		}

		update := game.FirstUpdatePacket{
			ClientId:  id,
			StartTime: pool.Game.StartTime,
			MaxX:      pool.Game.Screen.MaxX, MaxY: pool.Game.Screen.MaxY,
			State: game.GameState{
				Players:       pool.Game.State.Players,
				PeaCrds:       pool.Game.State.PeaCrds,
				PlusOneActive: pool.Game.State.PlusOneActive,
				TpsTracker:    pool.Game.State.TpsTracker,
			},
		}
		data, err := json.Marshal(update)
		if err != nil {
			pool.DelClient(client)
		}

		_, err = (*client).Write(append(data, '\n'))
		if err != nil {
			pool.DelClient(client)
		}
		i++
	}

	for i := 0; i < pool.Game.Config.PeaStartCount; i++ {
		pool.Game.SpawnPea()
	}

	pool.Status = "started"

	updateFramePlayer := max(1, pool.Game.Config.TargetTPS/pool.Game.Config.PlayerSpeed)
	updateFramePea := max(1, pool.Game.Config.PeaSpawnDelay*pool.Game.Config.TargetTPS)
	updateFramePlusOne := max(1, pool.Game.Config.PlusOneDelay*pool.Game.Config.TargetTPS)

	for i := 1; pool.Status == "started"; i++ {
		now := time.Now()
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
			pool.Game.State.PeaCrds = slices.DeleteFunc(pool.Game.State.PeaCrds, func(cord [2]int) bool {
				val, err := pool.Game.Screen.GetColRow(cord[0], cord[1])
				return err != nil || val != game.ObjPea
			})

			if len(pool.Game.State.PeaCrds) < pool.Game.Config.PeaSpawnLimit {
				pool.Game.SpawnPea()
			}
		}

		if doSend {
			update := game.GameState{
				Players:       pool.Game.State.Players,
				PeaCrds:       pool.Game.State.PeaCrds,
				PlusOneActive: pool.Game.State.PlusOneActive,
				TpsTracker:    pool.Game.State.TpsTracker,
			}

			data, err := json.Marshal(update)
			if err != nil {
				pool.Lgr.Log("high", "Error", err)
			} else {
				for _, client := range pool.Clients {
					_, err = (*client).Write(append(data, '\n'))
					if err != nil {
						pool.DelClient(client)
					}
				}
			}
		}

		nowDiff := time.Since(now)
		time.Sleep((time.Second / time.Duration(pool.Game.Config.TargetTPS)) - nowDiff)
		pool.Game.State.TpsTracker = int(time.Second/nowDiff) + 1
	}
}

func (pool *Pool) clientHandler(con *net.Conn) {
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
}

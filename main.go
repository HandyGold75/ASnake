package main

import (
	"ASnake/game"
	"ASnake/server"
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/HandyGold75/GOLib/argp"
	"github.com/HandyGold75/GOLib/tui"
)

var args = argp.ParseArgs(struct {
	Help       bool   `switch:"h,-help" opts:"help"        help:"Another game of Snake."`
	Server     bool   `switch:"s,-server"                  help:"Start as a server instace."`
	IP         string `switch:"i,-ip" default:"0.0.0.0"    help:"Listen on this ip when started as server."`
	Port       uint16 `switch:"p,-port" default:"17530"    help:"Listen on this port when started as server."`
	MaxClients int    `switch:"m,-max-clients" default:"4" help:"Max amount of clients per pool."`
}{})

func mainMenu(gm *game.Game) (mode string, ipStr string, err error) {
	mode = ""

	tui.Defaults.Align = tui.AlignLeft
	mm := tui.NewMenuBulky("ASnake")

	sp := mm.Menu.NewMenu("SinglePlayer")
	sp.NewAction("Start", func() { mode = "singleplayer" })
	spLockFPSToTPS := sp.NewList("Low Performance", []string{"No", "Yes"})
	spTargetTPS := sp.NewDigit("Target TPS", 30, 0, 99999)
	spTargetFPS := sp.NewDigit("Target FPS", 60, 0, 99999)
	spPlayerSpeed := sp.NewDigit("Player Speed", 5, 0, 99999)
	spPeaSpawnDelay := sp.NewDigit("Spawn Delay", 5, 0, 99999)
	spPeaSpawnLimit := sp.NewDigit("Spawn Limit", 3, 0, 99999)
	spPeaStartCount := sp.NewDigit("Spawn Count", 1, 0, 99999)

	mp := mm.Menu.NewMenu("MultiPlayer")
	mp.NewAction("Connect", func() { mode = "multiplayer" })
	mpIP := mp.NewIPv4("IP", "84.25.253.77")
	mpPort := mp.NewDigit("Port", 17530, 0, 65535)

	if err := mm.Run(); err != nil {
		return mode, "", err
	}

	gm.Config.LockFPSToTPS = spLockFPSToTPS.Value() == "Yes"
	if gm.Config.TargetTPS, err = strconv.Atoi(spTargetTPS.Value()); err != nil {
		return mode, "", err
	}
	if gm.Config.TargetFPS, err = strconv.Atoi(spTargetFPS.Value()); err != nil {
		return mode, "", err
	}
	if gm.Config.PlayerSpeed, err = strconv.Atoi(spPlayerSpeed.Value()); err != nil {
		return mode, "", err
	}
	if gm.Config.PeaSpawnDelay, err = strconv.Atoi(spPeaSpawnDelay.Value()); err != nil {
		return mode, "", err
	}
	if gm.Config.PeaSpawnLimit, err = strconv.Atoi(spPeaSpawnLimit.Value()); err != nil {
		return mode, "", err
	}
	if gm.Config.PeaStartCount, err = strconv.Atoi(spPeaStartCount.Value()); err != nil {
		return mode, "", err
	}
	return mode, fmt.Sprintf("%v:%v", mpIP.Value(), mpPort.Value()), nil
}

func connect(gm *game.Game, ip string) error {
	fmt.Print("\r\033[0JConnecting")

	tcp, err := net.ResolveTCPAddr("tcp", ip)
	if err != nil {
		fmt.Print("\r\033[0JFailed\r\n")
		return err
	}

	conn, err := net.DialTCP("tcp", nil, tcp)
	if err != nil {
		fmt.Print("\r\033[0JFailed\r\n")
		return err
	}

	_, err = conn.Write([]byte("Join\n"))
	if err != nil {
		fmt.Print("\r\033[0JFailed\r\n")
		_ = conn.Close()
		return err
	}

	reader := bufio.NewReader(conn)

	msg, err := reader.ReadString('\n')
	if err != nil {
		fmt.Print("\r\033[0JFailed\r\n")
		_ = conn.Close()
		return err
	}
	msg = strings.ReplaceAll(msg, "\n", "")

	if msg != "Accept" {
		fmt.Print("\r\033[0J" + msg + "\r\n")
		return conn.Close()
	}

	gm.Config.Connection = conn

	fmt.Print("\r\033[0JJoined\r")

	readerConn := bufio.NewReader(gm.Config.Connection)
	msg, err = readerConn.ReadString('\n')
	if err != nil {
		fmt.Print("\r\033[0JFailed\r\n")
		return err
	}
	msg = strings.ReplaceAll(msg, "\n", "")

	for msg == "waiting" {
		fmt.Print("\r\033[0JWaiting for players\r")

		msg, err = readerConn.ReadString('\n')
		if err != nil {
			fmt.Print("\r\033[0JFailed\r\n")
			return err
		}
		msg = strings.ReplaceAll(msg, "\n", "")
	}

	fmt.Print("\r\033[0JStarting\r\n")

	update := game.FirstUpdatePacket{}
	err = json.Unmarshal([]byte(msg), &update)
	if err != nil {
		fmt.Print("\r\033[0JFailed\r\n")
		return err
	}

	gm.Config.ClientId = update.ClientId
	gm.StartTime = update.StartTime
	gm.Screen.MaxX, gm.Screen.MaxY = update.MaxX, update.MaxY
	gm.State.Players = update.State.Players
	gm.State.PeaCrds = update.State.PeaCrds
	gm.State.PlusOneActive = update.State.PlusOneActive
	gm.State.TpsTracker = update.State.TpsTracker

	return nil
}

func Run() {
	gm, err := game.NewGame(false)
	if err != nil {
		panic(err)
	}
	mode, ipStr, err := mainMenu(gm)
	if err != nil {
		panic(err)
	}

	switch mode {
	case "singleplayer":
		if err := gm.Start(); err != nil {
			panic(err)
		}
	case "multiplayer":
		if err := connect(gm, ipStr); err != nil {
			panic(err)
		}
		if err := gm.Start(); err != nil {
			panic(err)
		}
	}
}

func main() {
	if args.Server {
		if err := server.NewServer(args.IP, args.Port, args.MaxClients).Run(); err != nil {
			panic(err)
		}
		fmt.Println()
	} else {
		Run()
	}
}

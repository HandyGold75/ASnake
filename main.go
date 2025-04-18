package main

import (
	"ASnake/game"
	"ASnake/server"
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/HandyGold75/GOLib/argp"
	"github.com/HandyGold75/GOLib/tui"
	"golang.org/x/term"
)

var args = argp.ParseArgs(struct {
	Help   bool `switch:"h,-help"   opts:"help" help:"Another game of Snake."`
	Server bool `switch:"s,-server"             help:"Start as a server instace."`
}{})

func mainMenu(gm *game.Game) (mode string, ipStr string, err error) {
	mode = ""

	tui.Defaults.Align = tui.AlignLeft
	mm := tui.NewMenuBulky("ASnake")

	sp := mm.Menu.NewMenu("SinglePlayer")
	sp.NewAction("Start", func() { mode = "singleplayer" })
	spLockFPSToTPS := sp.NewList("Low Performance", []string{"No", "Yes"})
	spTargetTPS := sp.NewDigit("Target TPS", 0, 30, 99999)
	spTargetFPS := sp.NewDigit("Target FPS", 0, 60, 99999)
	spPlayerSpeed := sp.NewDigit("Player Speed", 0, 5, 99999)
	spPeaSpawnDelay := sp.NewDigit("Spawn Delay", 0, 5, 99999)
	spPeaSpawnLimit := sp.NewDigit("Spawn Limit", 0, 3, 99999)
	spPeaStartCount := sp.NewDigit("Spawn Count", 0, 1, 99999)

	mp := mm.Menu.NewMenu("MultiPlayer")
	mp.NewAction("Connect", func() { mode = "multiplayer" })
	mpIP := mp.NewIPv4("IP", "84.25.253.77")
	mpPort := mp.NewDigit("Port", 17530, 0, 65535)

	sr := mm.Menu.NewMenu("Server")
	sr.NewAction("Start", func() { mode = "server" })

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
	gm.State.Players = update.Players
	gm.State.PeaCrds = update.PeaCrds
	gm.State.StartTime = update.StartTime
	gm.State.PlusOneActive = update.PlusOneActive
	gm.State.TpsTracker = update.TpsTracker
	gm.Screen.MaxX = update.CurX
	gm.Screen.MaxY = update.CurY

	return nil
}

func Run() {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	gm, err := game.NewClient(oldState)
	if err != nil {
		panic(err)
	}
	defer gm.Close()

	mode, ipStr, err := mainMenu(gm)
	if err != nil {
		panic(err)
	}

	switch mode {
	case "singleplayer":
		stop := make(chan bool)
		go func() {
			defer term.Restore(int(os.Stdin.Fd()), oldState)

			for {
				select {
				case <-stop:
					return
				default:
					in := make([]byte, 3)
					_, err := os.Stdin.Read(in)
					if err != nil {
						panic(err)
					}
					gm.HandleInput(in)
				}
			}
		}()
		if err := gm.Start(); err != nil {
			panic(err)
		}
	case "multiplayer":
		term.Restore(int(os.Stdin.Fd()), oldState)

		if err := connect(gm, ipStr); err != nil {
			panic(err)
		}

		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			panic(err)
		}

		stop := make(chan bool)
		go func() {
			defer term.Restore(int(os.Stdin.Fd()), oldState)

			for {
				select {
				case <-stop:
					return
				default:
					in := make([]byte, 3)
					_, err := os.Stdin.Read(in)
					if err != nil {
						panic(err)
					}
					gm.HandleInput(in)
				}
			}
		}()
		if err := gm.Start(); err != nil {
			panic(err)
		}
	case "server":
		term.Restore(int(os.Stdin.Fd()), oldState)
		if err := server.NewServer("127.0.0.1", 17530).Run(); err != nil {
			panic(err)
		}
		fmt.Println()
	}
}

func main() {
	if args.Server {
		if err := server.NewServer("127.0.0.1", 17530).Run(); err != nil {
			panic(err)
		}
		fmt.Println()
	} else {
		Run()
	}
}

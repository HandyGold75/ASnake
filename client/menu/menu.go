package menu

import (
	"ASnake/client/game"
	"ASnake/client/screen"
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

type (
	keyBinds struct {
		RETURN, ESC, CTRL_C, CTRL_D, Q                            []byte
		W, D, S, A, K, L, J, H, UP, RIGHT, DOWN, LEFT             []byte
		Zero, One, Two, Three, For, Five, Six, Seven, Eight, Nine []byte
	}
	menuObjects struct{ Default, Empty, Background, Text, Selected, Warning uint8 }

	Menu struct {
		KeyBinds keyBinds
		Objects  menuObjects
		Screen   *screen.Screen
	}
)

var (
	stop = make(chan string)
	trm  = &term.Terminal{}
	gm   = &game.Game{}

	ip1, ip2, ip3, ip4, port = 84, 25, 253, 77, 17530

	inputCallback          = func([]byte) bool { return false }
	backSelections         = []string{}
	optionSelectedCallback = func(string, bool) {}

	lastNumberInput   = time.Now()
	numberInputBuffer = []string{}

	settingSetter = func(menu *Menu, setting *int, maxValue int, backSels []string) {
		currentSelection = strconv.Itoa(*setting)

		s := make([]string, maxValue)
		for i := range s {
			s[i] = strconv.Itoa(i)
		}
		availalbeSelections = s

		inputCallback = menu.SettingInput
		backSelections = backSels
		optionSelectedCallback = func(value string, confirm bool) {
			out, err := strconv.Atoi(value)
			if err != nil {
				menu.Screen.H.RenderString(err.Error(), 8, 2, menu.Objects.Warning)
				time.Sleep(time.Second * time.Duration(3))
			}
			*setting = out
		}

		menu.updateMenu("")
	}

	mainSelections   = []string{"Start", "Multiplayer", "Options", "Exit"}
	optionSelections = []string{"Player Speed", "Spawn Delay", "Spawn Limit", "Spawn Count", "Low Performance", "Target TPS", "Target FPS"}
	mpSelections     = []string{"IP 1/4", "IP 2/4", "IP 3/4", "IP 4/4", "Port", "Connect"}

	currentSelection    = ""
	availalbeSelections = mainSelections
	selectionActions    = map[string]func(*Menu){
		"Multiplayer": func(menu *Menu) {
			currentSelection = ""
			availalbeSelections = mpSelections
			menu.updateMenu("")
		},
		"IP 1/4": func(menu *Menu) { settingSetter(menu, &ip1, 256, mpSelections) },
		"IP 2/4": func(menu *Menu) { settingSetter(menu, &ip2, 256, mpSelections) },
		"IP 3/4": func(menu *Menu) { settingSetter(menu, &ip3, 256, mpSelections) },
		"IP 4/4": func(menu *Menu) { settingSetter(menu, &ip4, 256, mpSelections) },
		"Port":   func(menu *Menu) { settingSetter(menu, &port, 65536, mpSelections) },
		"Connect": func(menu *Menu) {
			availalbeSelections = []string{"Confirm"}

			inputCallback = menu.SettingInput
			backSelections = mpSelections
			optionSelectedCallback = func(value string, confirm bool) {
				if !confirm || value != "Confirm" {
					return
				}

				menu.updateMenu("")
				menu.Screen.H.RenderString("Connecting", 2, 8, menu.Objects.Warning)

				con, err := NewConnection()
				if err != nil {
					menu.updateMenu("")
					menu.Screen.H.RenderString("Failed", 2, 8, menu.Objects.Warning)
					time.Sleep(time.Second * time.Duration(3))
					return
				}
				gm.Config.Connection = con

				menu.updateMenu("")
				menu.Screen.H.RenderString("Joined", 2, 8, menu.Objects.Warning)

				reader := bufio.NewReader(gm.Config.Connection)
				msg, err := reader.ReadString('\n')
				if err != nil {
					menu.updateMenu("")
					menu.Screen.H.RenderString("Failed", 2, 8, menu.Objects.Warning)
					time.Sleep(time.Second * time.Duration(3))
					return
				}
				msg = strings.ReplaceAll(msg, "\n", "")

				for msg == "waiting" {
					menu.updateMenu("")
					menu.Screen.H.RenderString("Waiting for players", 2, 8, menu.Objects.Warning)

					msg, err = reader.ReadString('\n')
					if err != nil {
						menu.updateMenu("")
						menu.Screen.H.RenderString("Failed", 2, 8, menu.Objects.Warning)
						time.Sleep(time.Second * time.Duration(3))
						return
					}
					msg = strings.ReplaceAll(msg, "\n", "")
				}

				menu.updateMenu("")
				menu.Screen.H.RenderString("Starting", 2, 8, menu.Objects.Warning)

				update := game.FirstUpdatePacket{}
				err = json.Unmarshal([]byte(msg), &update)
				if err != nil {
					menu.updateMenu("")
					menu.Screen.H.RenderString("Failed", 2, 8, menu.Objects.Warning)
					time.Sleep(time.Second * time.Duration(3))
					return
				}

				gm.Config.ClientId = update.ClientId
				gm.State.Players = update.Players
				gm.State.PeaCrds = update.PeaCrds
				gm.State.StartTime = update.StartTime
				gm.State.PlusOneActive = update.PlusOneActive
				gm.State.TpsTracker = update.TpsTracker
				gm.Screen.MaxX = update.CurX
				gm.Screen.MaxY = update.CurY

				stop <- "Start"
			}

			menu.updateMenu("")
			menu.Screen.H.RenderString(fmt.Sprintf("%d.%d.%d.%d:%d", ip1, ip2, ip3, ip4, port), 2, 8, menu.Objects.Warning)
		},

		"Options": func(menu *Menu) {
			currentSelection = ""
			availalbeSelections = optionSelections
			menu.updateMenu("")
		},
		"Player Speed": func(menu *Menu) { settingSetter(menu, &gm.Config.PlayerSpeed, gm.Config.TargetTPS+1, optionSelections) },
		"Spawn Delay":  func(menu *Menu) { settingSetter(menu, &gm.Config.PeaSpawnDelay, 100000, optionSelections) },
		"Spawn Limit":  func(menu *Menu) { settingSetter(menu, &gm.Config.PeaSpawnLimit, 100000, optionSelections) },
		"Spawn Count":  func(menu *Menu) { settingSetter(menu, &gm.Config.PeaStartCount, 100000, optionSelections) },
		"Low Performance": func(menu *Menu) {
			if gm.Config.LockFPSToTPS {
				currentSelection = "Yes"
			} else {
				currentSelection = "No"
			}
			availalbeSelections = []string{"Yes", "No"}

			inputCallback = menu.SettingInput
			backSelections = optionSelections
			optionSelectedCallback = func(value string, confirm bool) { gm.Config.LockFPSToTPS = value == "Yes" }

			menu.updateMenu("")
		},
		"Target TPS": func(menu *Menu) { settingSetter(menu, &gm.Config.TargetTPS, 100000, optionSelections) },
		"Target FPS": func(menu *Menu) { settingSetter(menu, &gm.Config.TargetFPS, 100000, optionSelections) },
	}
)

func NewConnection() (*net.TCPConn, error) {
	tcp, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%d.%d.%d.%d:%d", ip1, ip2, ip3, ip4, port))
	if err != nil {
		return &net.TCPConn{}, err
	}

	conn, err := net.DialTCP("tcp", nil, tcp)
	if err != nil {
		return &net.TCPConn{}, err
	}

	_, err = conn.Write([]byte("Join\n"))
	if err != nil {
		conn.Close()
		return &net.TCPConn{}, err
	}

	reader := bufio.NewReader(conn)

	msg, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return &net.TCPConn{}, err
	}
	msg = strings.ReplaceAll(msg, "\n", "")

	if msg != "Accept" {
		conn.Close()
		return &net.TCPConn{}, err
	}

	return conn, nil
}

func NewMenu(gamePnt *game.Game) (*Menu, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return &Menu{}, errors.New("stdin/ stdout should be a terminal")
	}
	gm = gamePnt

	trm = term.NewTerminal(struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}, "")

	menu := &Menu{
		KeyBinds: keyBinds{
			RETURN: []byte{13, 0, 0},
			ESC:    []byte{27, 0, 0}, CTRL_C: []byte{3, 0, 0}, CTRL_D: []byte{4, 0, 0}, Q: []byte{113, 0, 0},
			W: []byte{119, 0, 0}, D: []byte{100, 0, 0}, S: []byte{115, 0, 0}, A: []byte{97, 0, 0},
			K: []byte{108, 0, 0}, L: []byte{107, 0, 0}, J: []byte{106, 0, 0}, H: []byte{104, 0, 0},
			UP: []byte{27, 91, 65}, RIGHT: []byte{27, 91, 67}, DOWN: []byte{27, 91, 66}, LEFT: []byte{27, 91, 68},
			Zero: []byte{48, 0, 0}, One: []byte{49, 0, 0}, Two: []byte{50, 0, 0}, Three: []byte{51, 0, 0}, For: []byte{52, 0, 0}, Five: []byte{53, 0, 0}, Six: []byte{54, 0, 0}, Seven: []byte{55, 0, 0}, Eight: []byte{56, 0, 0}, Nine: []byte{57, 0, 0},
		},
		Objects: menuObjects{
			Default:    0,
			Empty:      1,
			Background: 2,
			Text:       3,
			Selected:   4,
			Warning:    5,
		},
	}

	ResetBytes := append([]byte("██"), trm.Escape.Reset...)
	scr, err := screen.NewScreen(2560, 1440, false, map[uint8][]byte{
		menu.Objects.Default:    append(trm.Escape.Magenta, ResetBytes...),
		menu.Objects.Empty:      []byte("  "),
		menu.Objects.Background: append(trm.Escape.Black, ResetBytes...),
		menu.Objects.Text:       append(trm.Escape.White, ResetBytes...),
		menu.Objects.Selected:   append(trm.Escape.Yellow, ResetBytes...),
		menu.Objects.Warning:    append(trm.Escape.Red, ResetBytes...),
	}, trm)
	if err != nil {
		return &Menu{}, err
	}

	scr.OnResizeCallback = func(scr *screen.Screen) {
		menu.updateMenu("")
	}

	menu.Screen = scr

	return menu, nil
}

func (menu *Menu) statsBar() {
	msg := fmt.Sprintf("Selected: %v   Size: %vx %vy", currentSelection, menu.Screen.CurX, menu.Screen.CurY)

	if len([]rune(msg)) > menu.Screen.CurX*2 {
		fmt.Printf("\033[2K\r%."+strconv.Itoa(menu.Screen.CurX*2)+"s...", msg)
	} else {
		fmt.Printf("\033[2K\r%."+strconv.Itoa(menu.Screen.CurX*2)+"s", msg)
	}
}

func (menu *Menu) SettingInput(in []byte) bool {
	if slices.Equal(in, menu.KeyBinds.CTRL_C) || slices.Equal(in, menu.KeyBinds.CTRL_D) || slices.Equal(in, menu.KeyBinds.ESC) || slices.Equal(in, menu.KeyBinds.Q) {
		stop <- "Exit"
		return true

	} else if slices.Equal(in, menu.KeyBinds.W) || slices.Equal(in, menu.KeyBinds.L) || slices.Equal(in, menu.KeyBinds.UP) {
		return false
	} else if slices.Equal(in, menu.KeyBinds.D) || slices.Equal(in, menu.KeyBinds.K) || slices.Equal(in, menu.KeyBinds.RIGHT) || slices.Equal(in, menu.KeyBinds.RETURN) {
		optionSelectedCallback(currentSelection, true)

		inputCallback = func([]byte) bool { return false }
		optionSelectedCallback = func(string, bool) {}

		currentSelection = ""
		availalbeSelections = backSelections
		menu.updateMenu("")

		return true

	} else if slices.Equal(in, menu.KeyBinds.S) || slices.Equal(in, menu.KeyBinds.J) || slices.Equal(in, menu.KeyBinds.DOWN) {
		return false
	} else if slices.Equal(in, menu.KeyBinds.A) || slices.Equal(in, menu.KeyBinds.H) || slices.Equal(in, menu.KeyBinds.LEFT) {
		optionSelectedCallback(currentSelection, false)

		inputCallback = func([]byte) bool { return false }
		optionSelectedCallback = func(string, bool) {}

		currentSelection = ""
		availalbeSelections = backSelections
		menu.updateMenu("")

		return true

	}

	for i, bind := range [][]byte{menu.KeyBinds.Zero, menu.KeyBinds.One, menu.KeyBinds.Two, menu.KeyBinds.Three, menu.KeyBinds.For, menu.KeyBinds.Five, menu.KeyBinds.Six, menu.KeyBinds.Seven, menu.KeyBinds.Eight, menu.KeyBinds.Nine} {
		if slices.Equal(in, bind) {
			if time.Now().After(lastNumberInput.Add(time.Second)) {
				if len(numberInputBuffer) > 1 {
					numberInputBuffer = []string{}
				}
			}

			lastNumberInput = time.Now()
			numberInputBuffer = append(numberInputBuffer, strconv.Itoa(i))

			if slices.Index(availalbeSelections, strings.Join(numberInputBuffer, "")) == -1 {
				lastNumberInput = time.Now().Add(-time.Hour)
				numberInputBuffer = []string{strconv.Itoa(i)}
			}

			currentSelection = strings.Join(numberInputBuffer, "")
			menu.updateMenu("")

			return true
		}
	}
	return true
}

func (menu *Menu) HandleInput(in []byte) {
	if inputCallback(in) {
		return
	}

	if slices.Equal(in, menu.KeyBinds.CTRL_C) || slices.Equal(in, menu.KeyBinds.CTRL_D) || slices.Equal(in, menu.KeyBinds.ESC) || slices.Equal(in, menu.KeyBinds.Q) {
		stop <- "Exit"
	} else if slices.Equal(in, menu.KeyBinds.W) || slices.Equal(in, menu.KeyBinds.L) || slices.Equal(in, menu.KeyBinds.UP) {
		menu.updateMenu("up")
	} else if slices.Equal(in, menu.KeyBinds.D) || slices.Equal(in, menu.KeyBinds.K) || slices.Equal(in, menu.KeyBinds.RIGHT) || slices.Equal(in, menu.KeyBinds.RETURN) {
		f, ok := selectionActions[currentSelection]
		if ok {
			f(menu)
		} else {
			stop <- currentSelection
		}
	} else if slices.Equal(in, menu.KeyBinds.S) || slices.Equal(in, menu.KeyBinds.J) || slices.Equal(in, menu.KeyBinds.DOWN) {
		menu.updateMenu("down")
	} else if slices.Equal(in, menu.KeyBinds.A) || slices.Equal(in, menu.KeyBinds.H) || slices.Equal(in, menu.KeyBinds.LEFT) {
		currentSelection = ""
		availalbeSelections = mainSelections
		menu.updateMenu("")
	}
}

func (menu *Menu) updateMenu(dir string) {
	index := max(0, slices.Index(availalbeSelections, currentSelection))
	if dir == "up" {
		index--
	} else if dir == "down" {
		index++
	}

	if index < 0 {
		index = len(availalbeSelections) - 1
	} else if index > len(availalbeSelections)-1 {
		index = 0
	}
	currentSelection = availalbeSelections[index]

	for i := 0; i <= menu.Screen.CurX; i++ {
		menu.Screen.SetCol(i, menu.Objects.Background)
	}
	for i := 0; i <= menu.Screen.CurY; i++ {
		menu.Screen.SetRow(i, menu.Objects.Background)
	}

	offsetX, offsetY := 1, 1
	for _, item := range availalbeSelections {
		if item == currentSelection {
			menu.Screen.H.RenderString(item, offsetX, offsetY-(index*6), menu.Objects.Selected)
		} else {
			menu.Screen.H.RenderString(item, offsetX, offsetY-(index*6), menu.Objects.Text)
		}
		offsetY += 6
	}

	menu.Screen.Draw()
	menu.statsBar()
}

func (menu *Menu) TestText() {
	menu.Screen.H.RenderString("ABCD EFGH IJKL M", 0, 0, menu.Objects.Text)
	menu.Screen.H.RenderString("NOPQ RSTU VWXY Z", 0, 6, menu.Objects.Text)
	for i := 0; i < 10; i++ {
		menu.Screen.H.RenderString(strconv.Itoa(i), i*6, 12, menu.Objects.Text)
	}
	menu.Screen.H.RenderString("+/.", 0, 18, menu.Objects.Text)

	menu.Screen.Draw()
}

func (menu *Menu) loop() string {
	for {
		select {
		case <-time.After(time.Millisecond * time.Duration(100)):
			menu.Screen.Draw()
		case currentSelection = <-stop:
			return currentSelection
		}
	}
}

func (menu *Menu) Start() string {
	menu.updateMenu("")
	out := menu.loop()

	fmt.Print("\033c")
	return out
}

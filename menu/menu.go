package menu

import (
	"ASnake/game"
	"ASnake/screen"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"time"

	"golang.org/x/term"
)

type (
	keyBinds    struct{ RETURN, ESC, CTRL_C, CTRL_D, Q, W, D, S, A, K, L, J, H, UP, RIGHT, DOWN, LEFT []byte }
	menuObjects struct{ Default, Empty, Background, Text, Selected, Warning int8 }

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

	inputCallback          = func([]byte) bool { return false }
	optionSelectedCallback = func(string) {}

	settingSetter = func(menu *Menu, setting *int, maxValue int) {
		currentSelection = strconv.Itoa(*setting)

		s := make([]string, maxValue)
		for i := range s {
			s[i] = strconv.Itoa(i)
		}
		availalbeSelections = s

		inputCallback = menu.SettingInput
		optionSelectedCallback = func(value string) {
			out, err := strconv.Atoi(value)
			if err != nil {
				menu.Screen.H.RenderString(err.Error(), 8, 2, menu.Objects.Warning)
				time.Sleep(time.Second * time.Duration(3))
			}
			*setting = out
		}

		menu.updateMenu("")
	}

	mainSelections   = []string{"Start", "Options", "Exit"}
	optionSelections = []string{"Player Speed", "Spawn Delay", "Spawn Limit", "Spawn Count"}

	currentSelection    = ""
	availalbeSelections = mainSelections
	selectionActions    = map[string]func(*Menu){
		"Options": func(menu *Menu) {
			currentSelection = ""
			availalbeSelections = optionSelections
			menu.updateMenu("")
		},
		"Player Speed": func(menu *Menu) { settingSetter(menu, &gm.Config.PlayerSpeed, 11) },
		"Spawn Delay":  func(menu *Menu) { settingSetter(menu, &gm.Config.PeaSpawnDelay, 999) },
		"Spawn Limit":  func(menu *Menu) { settingSetter(menu, &gm.Config.PeaSpawnLimit, 999) },
		"Spawn Count":  func(menu *Menu) { settingSetter(menu, &gm.Config.PeaStartCount, 999) },
	}
)

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
			ESC:    []byte{27, 0, 0},
			CTRL_C: []byte{3, 0, 0},
			CTRL_D: []byte{4, 0, 0},
			W:      []byte{119, 0, 0},
			Q:      []byte{113, 0, 0},
			D:      []byte{100, 0, 0},
			S:      []byte{115, 0, 0},
			A:      []byte{97, 0, 0},
			K:      []byte{108, 0, 0},
			L:      []byte{107, 0, 0},
			J:      []byte{106, 0, 0},
			H:      []byte{104, 0, 0},
			UP:     []byte{27, 91, 65},
			RIGHT:  []byte{27, 91, 67},
			DOWN:   []byte{27, 91, 66},
			LEFT:   []byte{27, 91, 68},
		},
		Objects: menuObjects{
			Default:    -1,
			Empty:      0,
			Background: 1,
			Text:       2,
			Selected:   3,
			Warning:    4,
		},
	}

	ResetBytes := append([]byte("██"), trm.Escape.Reset...)
	scr, err := screen.NewScreen(1000, 1000, map[int8][]byte{
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
	msg := fmt.Sprintf("\rSelected: %v   Size: %vx %vy", currentSelection, menu.Screen.CurX, menu.Screen.CurY)

	if len([]rune(msg)) > menu.Screen.CurX*2 {
		fmt.Printf("\n\033[2K\r%."+strconv.Itoa(menu.Screen.CurX*2)+"s...", msg)
	} else {
		fmt.Printf("\n\033[2K\r%."+strconv.Itoa(menu.Screen.CurX*2)+"s", msg)
	}
}

func (menu *Menu) SettingInput(in []byte) bool {
	if slices.Equal(in, menu.KeyBinds.CTRL_C) || slices.Equal(in, menu.KeyBinds.CTRL_D) || slices.Equal(in, menu.KeyBinds.ESC) || slices.Equal(in, menu.KeyBinds.Q) {
		stop <- "Exit"
	} else if slices.Equal(in, menu.KeyBinds.W) || slices.Equal(in, menu.KeyBinds.L) || slices.Equal(in, menu.KeyBinds.UP) {
		menu.updateMenu("up")
	} else if slices.Equal(in, menu.KeyBinds.D) || slices.Equal(in, menu.KeyBinds.K) || slices.Equal(in, menu.KeyBinds.RIGHT) || slices.Equal(in, menu.KeyBinds.RETURN) {
		optionSelectedCallback(currentSelection)

		inputCallback = func([]byte) bool { return false }
		optionSelectedCallback = func(string) {}

		currentSelection = ""
		availalbeSelections = optionSelections
		menu.updateMenu("")

	} else if slices.Equal(in, menu.KeyBinds.S) || slices.Equal(in, menu.KeyBinds.J) || slices.Equal(in, menu.KeyBinds.DOWN) {
		menu.updateMenu("down")
	} else if slices.Equal(in, menu.KeyBinds.A) || slices.Equal(in, menu.KeyBinds.H) || slices.Equal(in, menu.KeyBinds.LEFT) {
		optionSelectedCallback(currentSelection)

		inputCallback = func([]byte) bool { return false }
		optionSelectedCallback = func(string) {}

		currentSelection = ""
		availalbeSelections = optionSelections
		menu.updateMenu("")
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

	offsetX, offsetY := 2, 2
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
	menu.Screen.H.RenderString("+", 0, 18, menu.Objects.Text)

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

	fmt.Print("\r\n\033[2K\r")
	return out
}

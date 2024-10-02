package game

import (
	"ASnake/screen"
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

type (
	Cord screen.Cord

	keyBinds struct {
		ESC, P,
		CTRL_C, CTRL_D, Q,
		W, D, S, A, K, L, J, H, UP, RIGHT, DOWN, LEFT []byte
	}
	Player struct {
		Crd         Cord
		Dir, CurDir string
		TailCrds    []Cord
		IsGameOver  bool
	}
	gameObjects struct{ Default, Empty, Wall, PlusOne, Warning, Pea, Player uint8 }
	gameConfig  struct {
		LockFPSToTPS                                                           bool
		Connection                                                             net.Conn
		TargetTPS, TargetFPS                                                   int
		PlayerSpeed, PeaSpawnDelay, PeaSpawnLimit, PeaStartCount, PlusOneDelay int
	}
	gameState struct {
		Players       []Player
		PeaCrds       []Cord
		StartTime     time.Time
		PlusOneActive bool
		TpsTracker    int
		FpsTracker    int
	}

	Game struct {
		KeyBinds keyBinds
		Objects  gameObjects
		Config   gameConfig
		State    gameState
		Screen   *screen.Screen
	}
)

var (
	stopping = false
	paused   = false

	originalTrm = &term.State{}
	trm         = term.NewTerminal(struct {
		io.Reader
		io.Writer
	}{}, "")
)

func NewGame(orgTrm *term.State) (*Game, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return &Game{}, errors.New("stdin/ stdout should be a terminal")
	}

	originalTrm = orgTrm
	trm = term.NewTerminal(struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}, "")

	gm, err := NewGameNoTUI()
	if err != nil {
		return &Game{}, err
	}

	gm.State.Players = []Player{{
		Crd: Cord{X: int(gm.Screen.CurX / 2), Y: int(gm.Screen.CurY / 2)},
		Dir: "right", CurDir: "right",
		TailCrds: []Cord{},
	}}

	for i := 0; i <= gm.Screen.CurX; i++ {
		gm.Screen.SetCol(i, gm.Objects.Empty)
	}
	for i := 0; i <= gm.Screen.CurY; i++ {
		gm.Screen.SetRow(i, gm.Objects.Empty)
	}
	gm.Screen.SetColRow(gm.State.Players[0].Crd.X, gm.State.Players[0].Crd.Y, gm.Objects.Player)

	gm.Screen.OnResizeCallback = func(scr *screen.Screen) {
		for i := 0; i <= gm.Screen.CurX; i++ {
			gm.Screen.SetCol(i, gm.Objects.Empty)
		}
		for i := 0; i <= gm.Screen.CurY; i++ {
			gm.Screen.SetRow(i, gm.Objects.Empty)
		}

		scr.SetColRow(gm.State.Players[0].Crd.X, gm.State.Players[0].Crd.Y, gm.Objects.Player)

		for _, cord := range gm.State.Players[0].TailCrds {
			scr.SetColRow(cord.X, cord.Y, gm.Objects.Player)
		}

		for _, cord := range gm.State.PeaCrds {
			scr.SetColRow(cord.X, cord.Y, gm.Objects.Pea)
		}
	}

	return gm, nil

}

func NewGameNoTUI() (*Game, error) {
	game := &Game{
		KeyBinds: keyBinds{
			ESC: []byte{27, 0, 0}, P: []byte{112, 0, 0},
			CTRL_C: []byte{3, 0, 0}, CTRL_D: []byte{4, 0, 0}, Q: []byte{113, 0, 0},
			W: []byte{119, 0, 0}, D: []byte{100, 0, 0}, S: []byte{115, 0, 0}, A: []byte{97, 0, 0},
			K: []byte{108, 0, 0}, L: []byte{107, 0, 0}, J: []byte{106, 0, 0}, H: []byte{104, 0, 0},
			UP: []byte{27, 91, 65}, RIGHT: []byte{27, 91, 67}, DOWN: []byte{27, 91, 66}, LEFT: []byte{27, 91, 68},
		},
		Objects: gameObjects{
			Default: 0,
			Empty:   1,
			Wall:    2,
			PlusOne: 3,
			Warning: 4,
			Pea:     5,
			Player:  6,
		},
		Config: gameConfig{
			LockFPSToTPS:  false,
			TargetTPS:     30,
			TargetFPS:     60,
			PlayerSpeed:   24, // Player moves 1 tile every `TargetTPS-PlayerSpeed` updates.
			PeaSpawnDelay: 5,
			PeaSpawnLimit: 3,
			PeaStartCount: 1,
			PlusOneDelay:  1,
		},
		State: gameState{
			Players:       []Player{},
			PeaCrds:       []Cord{},
			StartTime:     time.Now(),
			PlusOneActive: false,
			TpsTracker:    0,
			FpsTracker:    0,
		},
	}

	ResetBytes := append([]byte("██"), trm.Escape.Reset...)
	scr, err := screen.NewScreen(1000, 1000, map[uint8][]byte{
		game.Objects.Default: append(trm.Escape.Magenta, ResetBytes...),
		game.Objects.Empty:   []byte("  "),
		game.Objects.Wall:    append(trm.Escape.Black, ResetBytes...),
		game.Objects.PlusOne: append(trm.Escape.Green, ResetBytes...),
		game.Objects.Warning: append(trm.Escape.Red, ResetBytes...),
		game.Objects.Pea:     append(trm.Escape.Yellow, ResetBytes...),
		game.Objects.Player:  append(trm.Escape.White, ResetBytes...),
	}, trm)
	if err != nil {
		return &Game{}, err
	}

	scr.SetCol(0, game.Objects.Wall)
	scr.SetRow(0, game.Objects.Wall)
	scr.SetCol(scr.CurX, game.Objects.Wall)
	scr.SetRow(scr.CurY, game.Objects.Wall)

	game.Screen = scr

	return game, nil
}

func (game *Game) statsBar() {
	timeDiff := time.Now().Sub(game.State.StartTime)
	timeStr := fmt.Sprintf("%02d:%02d:%02d:%03d", int(timeDiff.Hours()), int(timeDiff.Minutes())%60, int(timeDiff.Seconds())%60, int(timeDiff.Milliseconds())%1000)

	tpsColor := ""
	if game.State.TpsTracker < game.Config.TargetTPS {
		tpsColor = string(trm.Escape.Red)
	}
	fpsColor := ""
	if !game.Config.LockFPSToTPS && game.State.FpsTracker < game.Config.TargetFPS-(game.Config.TargetFPS/5) {
		fpsColor = string(trm.Escape.Red)
	}

	msg := fmt.Sprintf("Time: %v   Peas: %v   Size: %vx %vy   FPS: %v   TPS: %v ", timeStr, len(game.State.Players[0].TailCrds), game.Screen.CurX, game.Screen.CurY, fpsColor+strconv.Itoa(game.State.FpsTracker)+string(trm.Escape.Reset), tpsColor+strconv.Itoa(game.State.TpsTracker)+string(trm.Escape.Reset))

	if len([]rune(msg)) > game.Screen.CurX*2 {
		fmt.Printf("\033[2K\r%."+strconv.Itoa(game.Screen.CurX*2)+"s...", msg)
	} else {
		fmt.Printf("\033[2K\r%."+strconv.Itoa(game.Screen.CurX*2)+"s", msg)
	}
}

func (game *Game) HandleInput(in []byte) {
	if slices.Equal(in, game.KeyBinds.CTRL_C) || slices.Equal(in, game.KeyBinds.CTRL_D) || slices.Equal(in, game.KeyBinds.Q) {
		stopping = true
		return

	} else if game.State.Players[0].IsGameOver {
		return

	} else if slices.Equal(in, game.KeyBinds.ESC) || slices.Equal(in, game.KeyBinds.P) {
		paused = !paused
		if paused {
			game.Screen.H.RenderStringIf("Paused", 2, 2, game.Objects.Warning, func(val uint8) bool { return val < game.Objects.Player })
		} else {
			game.Screen.H.RenderStringIf("Paused", 2, 2, game.Objects.Empty, func(val uint8) bool { return val < game.Objects.Player })
		}
		game.Screen.Draw()
		return
	}

	dir := game.State.Players[0].Dir
	if !paused && game.State.Players[0].CurDir != "down" && (slices.Equal(in, game.KeyBinds.W) || slices.Equal(in, game.KeyBinds.L) || slices.Equal(in, game.KeyBinds.UP)) {
		dir = "up"
	} else if !paused && game.State.Players[0].CurDir != "left" && (slices.Equal(in, game.KeyBinds.D) || slices.Equal(in, game.KeyBinds.K) || slices.Equal(in, game.KeyBinds.RIGHT)) {
		dir = "right"
	} else if !paused && game.State.Players[0].CurDir != "up" && (slices.Equal(in, game.KeyBinds.S) || slices.Equal(in, game.KeyBinds.J) || slices.Equal(in, game.KeyBinds.DOWN)) {
		dir = "down"
	} else if !paused && game.State.Players[0].CurDir != "right" && (slices.Equal(in, game.KeyBinds.A) || slices.Equal(in, game.KeyBinds.H) || slices.Equal(in, game.KeyBinds.LEFT)) {
		dir = "left"
	}

	if game.Config.Connection != nil {
		game.Config.Connection.Write([]byte(dir + "\n"))
		return
	}
	game.State.Players[0].Dir = dir
}

func (game *Game) UpdatePlayer(id int) {
	oldCords := game.State.Players[id].Crd
	if game.State.Players[id].Dir == "up" {
		game.State.Players[id].Crd.Y -= 1
	} else if game.State.Players[id].Dir == "right" {
		game.State.Players[id].Crd.X += 1
	} else if game.State.Players[id].Dir == "down" {
		game.State.Players[id].Crd.Y += 1
	} else if game.State.Players[id].Dir == "left" {
		game.State.Players[id].Crd.X -= 1
	}
	game.State.Players[id].CurDir = game.State.Players[id].Dir

	if game.State.Players[id].Crd.X <= 0 {
		game.State.Players[id].Crd.X = game.Screen.CurX - 1
	} else if game.State.Players[id].Crd.X >= game.Screen.CurX {
		game.State.Players[id].Crd.X = 1
	} else if game.State.Players[id].Crd.Y <= 0 {
		game.State.Players[id].Crd.Y = game.Screen.CurY - 1
	} else if game.State.Players[id].Crd.Y >= game.Screen.CurY {
		game.State.Players[id].Crd.Y = 1
	}

	val, err := game.Screen.GetColRow(game.State.Players[id].Crd.X, game.State.Players[id].Crd.Y)
	if err != nil { // Player most likely downscaled extremly fast, skip update as the current frame is unreliable.
		return
	}

	if val == game.Objects.Player {
		game.State.Players[id].IsGameOver = true
		game.Screen.H.RenderString("Game", 2, 2, game.Objects.Warning)
		game.Screen.H.RenderString("Over", 8, 8, game.Objects.Warning)
		return
	}

	if val == game.Objects.Pea {
		game.State.PeaCrds = slices.DeleteFunc(game.State.PeaCrds, func(cord Cord) bool {
			return cord == Cord{X: game.State.Players[id].Crd.X, Y: game.State.Players[id].Crd.Y}
		})

		game.State.Players[id].TailCrds = append(game.State.Players[id].TailCrds, Cord{X: oldCords.X, Y: oldCords.Y})

		game.State.PlusOneActive = true
		game.Screen.H.RenderCordsIf(screen.Chars.Plus, 2, 2, game.Objects.PlusOne, func(val uint8) bool { return val == game.Objects.Empty || val == game.Objects.PlusOne })
		game.Screen.H.RenderCordsIf(screen.Chars.One, 8, 2, game.Objects.PlusOne, func(val uint8) bool { return val == game.Objects.Empty || val == game.Objects.PlusOne })

	} else {
		if len(game.State.Players[id].TailCrds) > 0 {
			game.State.Players[id].TailCrds = append(game.State.Players[id].TailCrds, Cord{X: oldCords.X, Y: oldCords.Y})
			oldCords = game.State.Players[id].TailCrds[id]
			game.State.Players[id].TailCrds = slices.Delete(game.State.Players[id].TailCrds, 0, 1)
		}
		game.Screen.SetColRow(oldCords.X, oldCords.Y, game.Objects.Empty)
	}

	game.Screen.SetColRow(game.State.Players[id].Crd.X, game.State.Players[id].Crd.Y, game.Objects.Player)
}

func (game *Game) SpawnPea() {
	for i := 1; i < 10; i++ {
		cord := Cord{X: rand.IntN(game.Screen.CurX-1) + 1, Y: rand.IntN(game.Screen.CurY-1) + 1}
		val, _ := game.Screen.GetColRow(cord.X, cord.Y)
		if val == game.Objects.Empty {
			game.State.PeaCrds = append(game.State.PeaCrds, cord)
			game.Screen.SetColRow(cord.X, cord.Y, game.Objects.Pea)
			break
		}
	}
}

func (game *Game) loop() {
	updateFramePlayer := max(1, game.Config.TargetTPS-game.Config.PlayerSpeed)
	updateFramePea := max(1, game.Config.PeaSpawnDelay*game.Config.TargetTPS)
	updateFramePlusOne := max(1, game.Config.PlusOneDelay*game.Config.TargetTPS)

	if !game.Config.LockFPSToTPS {
		go func() {
			defer func() { stopping = true; term.Restore(int(os.Stdin.Fd()), originalTrm) }()

			for !stopping {
				t := time.Now()

				game.Screen.Draw()
				time.Sleep((time.Second / time.Duration(game.Config.TargetFPS)) - time.Now().Sub(t))

				game.State.FpsTracker = int(time.Second/time.Now().Sub(t)) + 1
				if !game.State.Players[0].IsGameOver {
					game.statsBar()
				}

			}
		}()
	}

	if game.Config.Connection != nil {
		reader := bufio.NewReader(game.Config.Connection)
		for !stopping {
			msg, err := reader.ReadString('\n')
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					// Disconnected
					panic("Disconnected")
					break
				}
				panic(err)
				break
			}
			msg = strings.ReplaceAll(msg, "\n", "")
			err = json.Unmarshal([]byte(msg), &game.State)
			if err != nil {
				panic(err)
			}
		}

		return
	}

	for i := 1; !stopping; i++ {
		t := time.Now()

		if paused || game.State.Players[0].IsGameOver {
			time.Sleep((time.Second / time.Duration(game.Config.TargetTPS)) - time.Now().Sub(t))
			game.State.TpsTracker = int(time.Second/time.Now().Sub(t)) + 1
			game.State.StartTime = game.State.StartTime.Add(time.Now().Sub(t))

			if paused && game.Config.LockFPSToTPS {
				game.State.FpsTracker = game.State.TpsTracker
				game.statsBar()

			}
			continue
		}

		if game.State.PlusOneActive && i%updateFramePlusOne == 0 {
			game.State.PlusOneActive = false
			game.Screen.H.RenderCordsIf(screen.Chars.Plus, 2, 2, game.Objects.Empty, func(val uint8) bool { return val == game.Objects.Empty || val == game.Objects.PlusOne })
			game.Screen.H.RenderCordsIf(screen.Chars.One, 8, 2, game.Objects.Empty, func(val uint8) bool { return val == game.Objects.Empty || val == game.Objects.PlusOne })

		}

		if i%updateFramePlayer == 0 {
			game.Screen.SetCol(0, game.Objects.Wall)
			game.Screen.SetRow(0, game.Objects.Wall)
			game.Screen.SetCol(game.Screen.CurX, game.Objects.Wall)
			game.Screen.SetRow(game.Screen.CurY, game.Objects.Wall)

			game.UpdatePlayer(0)
		}

		if i%updateFramePea == 0 {
			game.State.PeaCrds = slices.DeleteFunc(game.State.PeaCrds, func(cord Cord) bool {
				val, err := game.Screen.GetColRow(cord.X, cord.Y)
				return err != nil || val != game.Objects.Pea
			})

			if len(game.State.PeaCrds) < game.Config.PeaSpawnLimit {
				game.SpawnPea()
			}
		}

		if game.Config.LockFPSToTPS {
			game.Screen.Draw()

			time.Sleep((time.Second / time.Duration(game.Config.TargetTPS)) - time.Now().Sub(t))
			game.State.TpsTracker = int(time.Second/time.Now().Sub(t)) + 1

			game.State.FpsTracker = game.State.TpsTracker
			game.statsBar()
			continue
		}

		time.Sleep((time.Second / time.Duration(game.Config.TargetTPS)) - time.Now().Sub(t))
		game.State.TpsTracker = int(time.Second/time.Now().Sub(t)) + 1
	}
}

func (game *Game) Start() error {
	for i := 0; i < game.Config.PeaStartCount; i++ {
		game.SpawnPea()
	}

	game.State.StartTime = time.Now()
	game.Screen.Draw()
	game.loop()

	fmt.Print("\033[0;0H\r\n" + string(game.Screen.CharMap[game.Objects.Wall]))

	return nil
}

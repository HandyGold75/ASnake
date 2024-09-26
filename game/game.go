package game

import (
	"ASnake/screen"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"slices"
	"strconv"
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
	gameObjects struct{ Default, Empty, Wall, PlusOne, Warning, Pea, Player int8 }
	gameConfig  struct {
		LockFPSToTPS                                                           bool
		TargetTPS, TargetFPS                                                   int
		PlayerSpeed, PeaSpawnDelay, PeaSpawnLimit, PeaStartCount, PlusOneDelay int
	}
	gameState struct {
		PlayerCrd               Cord
		PlayerDir, PlayerCurDir string
		TailCrds, PeaCrds       []Cord
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
	trm         = &term.Terminal{}

	startTime     = time.Now()
	plusOneActive = false

	tpsTracker = 0
	fpsTracker = 0
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

	game := &Game{
		KeyBinds: keyBinds{
			ESC: []byte{27, 0, 0}, P: []byte{112, 0, 0},
			CTRL_C: []byte{3, 0, 0}, CTRL_D: []byte{4, 0, 0}, Q: []byte{113, 0, 0},
			W: []byte{119, 0, 0}, D: []byte{100, 0, 0}, S: []byte{115, 0, 0}, A: []byte{97, 0, 0},
			K: []byte{108, 0, 0}, L: []byte{107, 0, 0}, J: []byte{106, 0, 0}, H: []byte{104, 0, 0},
			UP: []byte{27, 91, 65}, RIGHT: []byte{27, 91, 67}, DOWN: []byte{27, 91, 66}, LEFT: []byte{27, 91, 68},
		},
		Objects: gameObjects{
			Default: -1,
			Empty:   0,
			Wall:    1,
			PlusOne: 2,
			Warning: 3,
			Pea:     4,
			Player:  5,
		},
		Config: gameConfig{
			TargetTPS:     30,
			TargetFPS:     60,
			LockFPSToTPS:  false,
			PlayerSpeed:   24, // Player moves 1 tile every `TargetTPS-PlayerSpeed` updates.
			PeaSpawnDelay: 5,
			PeaSpawnLimit: 3,
			PeaStartCount: 1,
			PlusOneDelay:  1,
		},
	}

	ResetBytes := append([]byte("██"), trm.Escape.Reset...)
	scr, err := screen.NewScreen(1000, 1000, map[int8][]byte{
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

	game.State = gameState{
		PlayerCrd: Cord{X: int(scr.CurX / 2), Y: int(scr.CurY / 2)},
		PlayerDir: "right",
		TailCrds:  []Cord{},
		PeaCrds:   []Cord{},
	}

	scr.SetCol(0, game.Objects.Wall)
	scr.SetRow(0, game.Objects.Wall)
	scr.SetCol(scr.CurX, game.Objects.Wall)
	scr.SetRow(scr.CurY, game.Objects.Wall)
	scr.SetColRow(game.State.PlayerCrd.X, game.State.PlayerCrd.Y, game.Objects.Player)

	scr.OnResizeCallback = func(scr *screen.Screen) {
		scr.SetColRow(game.State.PlayerCrd.X, game.State.PlayerCrd.Y, game.Objects.Player)

		for _, cord := range game.State.TailCrds {
			scr.SetColRow(cord.X, cord.Y, game.Objects.Player)
		}

		for _, cord := range game.State.PeaCrds {
			scr.SetColRow(cord.X, cord.Y, game.Objects.Pea)
		}
	}

	game.Screen = scr

	return game, nil
}

func (game *Game) statsBar() {
	timeDiff := time.Now().Sub(startTime)
	timeStr := fmt.Sprintf("%02d:%02d:%02d:%03d", int(timeDiff.Hours()), int(timeDiff.Minutes())%60, int(timeDiff.Seconds())%60, int(timeDiff.Milliseconds())%1000)

	tpsColor := ""
	if tpsTracker < game.Config.TargetTPS {
		tpsColor = string(trm.Escape.Red)
	}
	fpsColor := ""
	if !game.Config.LockFPSToTPS && fpsTracker < game.Config.TargetFPS-(game.Config.TargetFPS/5) {
		fpsColor = string(trm.Escape.Red)
	}

	msg := fmt.Sprintf("Time: %v   Peas: %v   Size: %vx %vy   FPS: %v   TPS: %v ", timeStr, len(game.State.TailCrds), game.Screen.CurX, game.Screen.CurY, fpsColor+strconv.Itoa(fpsTracker)+string(trm.Escape.Reset), tpsColor+strconv.Itoa(tpsTracker)+string(trm.Escape.Reset))

	if len([]rune(msg)) > game.Screen.CurX*2 {
		fmt.Printf("\033[2K\r%."+strconv.Itoa(game.Screen.CurX*2)+"s...", msg)
	} else {
		fmt.Printf("\033[2K\r%."+strconv.Itoa(game.Screen.CurX*2)+"s", msg)
	}
}

func (game *Game) HandleInput(in []byte) {
	if slices.Equal(in, game.KeyBinds.CTRL_C) || slices.Equal(in, game.KeyBinds.CTRL_D) || slices.Equal(in, game.KeyBinds.Q) {
		stopping = true
	} else if slices.Equal(in, game.KeyBinds.ESC) || slices.Equal(in, game.KeyBinds.P) {
		paused = !paused
		if paused {
			game.Screen.H.RenderStringIf("Paused", 2, 2, game.Objects.Warning, func(val int8) bool { return val < game.Objects.Player })
		} else {
			game.Screen.H.RenderStringIf("Paused", 2, 2, game.Objects.Empty, func(val int8) bool { return val < game.Objects.Player })
		}
		game.Screen.Draw()
	} else if !paused && game.State.PlayerCurDir != "down" && (slices.Equal(in, game.KeyBinds.W) || slices.Equal(in, game.KeyBinds.L) || slices.Equal(in, game.KeyBinds.UP)) {
		game.State.PlayerDir = "up"
	} else if !paused && game.State.PlayerCurDir != "left" && (slices.Equal(in, game.KeyBinds.D) || slices.Equal(in, game.KeyBinds.K) || slices.Equal(in, game.KeyBinds.RIGHT)) {
		game.State.PlayerDir = "right"
	} else if !paused && game.State.PlayerCurDir != "up" && (slices.Equal(in, game.KeyBinds.S) || slices.Equal(in, game.KeyBinds.J) || slices.Equal(in, game.KeyBinds.DOWN)) {
		game.State.PlayerDir = "down"
	} else if !paused && game.State.PlayerCurDir != "right" && (slices.Equal(in, game.KeyBinds.A) || slices.Equal(in, game.KeyBinds.H) || slices.Equal(in, game.KeyBinds.LEFT)) {
		game.State.PlayerDir = "left"
	}
}

func (game *Game) updatePlayer() {
	oldCords := game.State.PlayerCrd
	if game.State.PlayerDir == "up" {
		game.State.PlayerCrd.Y -= 1
	} else if game.State.PlayerDir == "right" {
		game.State.PlayerCrd.X += 1
	} else if game.State.PlayerDir == "down" {
		game.State.PlayerCrd.Y += 1
	} else if game.State.PlayerDir == "left" {
		game.State.PlayerCrd.X -= 1
	}
	game.State.PlayerCurDir = game.State.PlayerDir

	if game.State.PlayerCrd.X <= 0 {
		game.State.PlayerCrd.X = game.Screen.CurX - 1
	} else if game.State.PlayerCrd.X >= game.Screen.CurX {
		game.State.PlayerCrd.X = 1
	} else if game.State.PlayerCrd.Y <= 0 {
		game.State.PlayerCrd.Y = game.Screen.CurY - 1
	} else if game.State.PlayerCrd.Y >= game.Screen.CurY {
		game.State.PlayerCrd.Y = 1
	}

	val, err := game.Screen.GetColRow(game.State.PlayerCrd.X, game.State.PlayerCrd.Y)
	if err != nil { // Player most likely downscaled extremly fast, skip update as the current frame is unreliable.
		return
	}

	if val == game.Objects.Player {
		stopping = true
		game.Screen.H.RenderString("Game", 2, 2, game.Objects.Warning)
		game.Screen.H.RenderString("Over", 8, 8, game.Objects.Warning)
		game.Screen.Draw()
		return
	}

	if val == game.Objects.Pea {
		game.State.PeaCrds = slices.DeleteFunc(game.State.PeaCrds, func(cord Cord) bool {
			return cord == Cord{game.State.PlayerCrd.X, game.State.PlayerCrd.Y}
		})

		game.State.TailCrds = append(game.State.TailCrds, Cord{X: oldCords.X, Y: oldCords.Y})

		plusOneActive = true
		game.Screen.H.RenderCordsIf(screen.Chars.Plus, 2, 2, game.Objects.PlusOne, func(val int8) bool { return val == game.Objects.Empty || val == game.Objects.PlusOne })
		game.Screen.H.RenderCordsIf(screen.Chars.One, 8, 2, game.Objects.PlusOne, func(val int8) bool { return val == game.Objects.Empty || val == game.Objects.PlusOne })

	} else {
		if len(game.State.TailCrds) > 0 {
			game.State.TailCrds = append(game.State.TailCrds, Cord{X: oldCords.X, Y: oldCords.Y})
			oldCords = game.State.TailCrds[0]
			game.State.TailCrds = slices.Delete(game.State.TailCrds, 0, 1)
		}
		game.Screen.SetColRow(oldCords.X, oldCords.Y, game.Objects.Empty)
	}

	game.Screen.SetColRow(game.State.PlayerCrd.X, game.State.PlayerCrd.Y, game.Objects.Player)
}

func (game *Game) spawnPea() {
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

				fpsTracker = int(time.Second/time.Now().Sub(t)) + 1
				game.statsBar()

			}
		}()
	}

	for i := 1; !stopping; i++ {
		t := time.Now()

		if paused {
			time.Sleep((time.Second / time.Duration(game.Config.TargetTPS)) - time.Now().Sub(t))
			tpsTracker = int(time.Second/time.Now().Sub(t)) + 1
			startTime = startTime.Add(time.Now().Sub(t))

			if game.Config.LockFPSToTPS {
				fpsTracker = tpsTracker
				game.statsBar()
			}
			continue
		}

		if plusOneActive && i%updateFramePlusOne == 0 {
			plusOneActive = false
			game.Screen.H.RenderCordsIf(screen.Chars.Plus, 2, 2, game.Objects.Empty, func(val int8) bool { return val == game.Objects.Empty || val == game.Objects.PlusOne })
			game.Screen.H.RenderCordsIf(screen.Chars.One, 8, 2, game.Objects.Empty, func(val int8) bool { return val == game.Objects.Empty || val == game.Objects.PlusOne })

		}

		if i%updateFramePlayer == 0 {
			game.Screen.SetCol(0, game.Objects.Wall)
			game.Screen.SetRow(0, game.Objects.Wall)
			game.Screen.SetCol(game.Screen.CurX, game.Objects.Wall)
			game.Screen.SetRow(game.Screen.CurY, game.Objects.Wall)

			game.updatePlayer()
		}

		if i%updateFramePea == 0 {
			game.State.PeaCrds = slices.DeleteFunc(game.State.PeaCrds, func(cord Cord) bool {
				val, err := game.Screen.GetColRow(cord.X, cord.Y)
				return err != nil || val != game.Objects.Pea
			})

			if len(game.State.PeaCrds) < game.Config.PeaSpawnLimit {
				game.spawnPea()
			}
		}

		if game.Config.LockFPSToTPS {
			game.Screen.Draw()

			time.Sleep((time.Second / time.Duration(game.Config.TargetTPS)) - time.Now().Sub(t))
			tpsTracker = int(time.Second/time.Now().Sub(t)) + 1

			fpsTracker = tpsTracker
			game.statsBar()
			continue
		}

		time.Sleep((time.Second / time.Duration(game.Config.TargetTPS)) - time.Now().Sub(t))
		tpsTracker = int(time.Second/time.Now().Sub(t)) + 1
	}
}

func (game *Game) Start() error {
	for i := 0; i < game.Config.PeaStartCount; i++ {
		game.spawnPea()
	}

	startTime = time.Now()
	game.Screen.Draw()
	game.loop()

	fmt.Print("\033[0;0H\r\n" + string(game.Screen.CharMap[game.Objects.Wall]))

	return nil
}

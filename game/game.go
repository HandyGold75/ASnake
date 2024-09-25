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

	keyBinds    struct{ ESC, CTRL_C, CTRL_D, Q, W, D, S, A, K, L, J, H, UP, RIGHT, DOWN, LEFT []byte }
	gameObjects struct{ Default, Empty, Wall, Player, Pea, GameOver, PlusOne int8 }
	gameConfig  struct {
		MinFrameTime                                                           int
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
	trm      = &term.Terminal{}

	startTime     = time.Now()
	plusOneActive = false
)

func NewGame() (*Game, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return &Game{}, errors.New("stdin/ stdout should be a terminal")
	}

	trm = term.NewTerminal(struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}, "")

	game := &Game{
		KeyBinds: keyBinds{
			ESC:    []byte{27, 0, 0},
			CTRL_C: []byte{3, 0, 0},
			CTRL_D: []byte{4, 0, 0},
			Q:      []byte{113, 0, 0},
			W:      []byte{119, 0, 0},
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
		Objects: gameObjects{
			Default:  -1,
			Empty:    0,
			Wall:     1,
			Player:   2,
			Pea:      3,
			GameOver: 4,
			PlusOne:  5,
		},
		Config: gameConfig{
			MinFrameTime:  100,
			PlayerSpeed:   8, // Player moves 1 tile every `10-PlayerSpeed` updates.
			PeaSpawnDelay: 5,
			PeaSpawnLimit: 3,
			PeaStartCount: 1,
			PlusOneDelay:  1,
		},
	}

	ResetBytes := append([]byte("██"), trm.Escape.Reset...)
	scr, err := screen.NewScreen(1000, 1000, map[int8][]byte{
		game.Objects.Default:  append(trm.Escape.Magenta, ResetBytes...),
		game.Objects.Empty:    []byte("  "),
		game.Objects.Wall:     append(trm.Escape.Black, ResetBytes...),
		game.Objects.Player:   append(trm.Escape.White, ResetBytes...),
		game.Objects.Pea:      append(trm.Escape.Yellow, ResetBytes...),
		game.Objects.GameOver: append(trm.Escape.Red, ResetBytes...),
		game.Objects.PlusOne:  append(trm.Escape.Green, ResetBytes...),
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

func (game *Game) statsBar(t time.Time) {
	timeDiff := time.Now().Sub(startTime)
	timeStr := fmt.Sprintf("%02d:%02d:%02d", int(timeDiff.Hours()), int(timeDiff.Minutes())%60, int(timeDiff.Seconds())%60)

	delay := time.Now().Sub(t).Truncate(time.Microsecond)
	delayColor := ""
	if delay > time.Duration(game.Config.MinFrameTime)*time.Millisecond {
		delayColor = string(trm.Escape.Red)
	}

	msg := fmt.Sprintf("\rTime: %v   Peas: %v   Size: %vx %vy   Delay: %v ", timeStr, len(game.State.TailCrds)+1, game.Screen.CurX, game.Screen.CurY, delayColor+delay.String()+string(trm.Escape.Reset))

	if len([]rune(msg)) > game.Screen.CurX*2 {
		fmt.Printf("\n\033[2K\r%."+strconv.Itoa(game.Screen.CurX*2)+"s...", msg)
	} else {
		fmt.Printf("\n\033[2K\r%."+strconv.Itoa(game.Screen.CurX*2)+"s", msg)
	}
}

func (game *Game) HandleInput(in []byte) {
	if slices.Equal(in, game.KeyBinds.CTRL_C) || slices.Equal(in, game.KeyBinds.CTRL_D) || slices.Equal(in, game.KeyBinds.ESC) || slices.Equal(in, game.KeyBinds.Q) {
		stopping = true
	} else if game.State.PlayerCurDir != "down" && (slices.Equal(in, game.KeyBinds.W) || slices.Equal(in, game.KeyBinds.L) || slices.Equal(in, game.KeyBinds.UP)) {
		game.State.PlayerDir = "up"
	} else if game.State.PlayerCurDir != "left" && (slices.Equal(in, game.KeyBinds.D) || slices.Equal(in, game.KeyBinds.K) || slices.Equal(in, game.KeyBinds.RIGHT)) {
		game.State.PlayerDir = "right"
	} else if game.State.PlayerCurDir != "up" && (slices.Equal(in, game.KeyBinds.S) || slices.Equal(in, game.KeyBinds.J) || slices.Equal(in, game.KeyBinds.DOWN)) {
		game.State.PlayerDir = "down"
	} else if game.State.PlayerCurDir != "right" && (slices.Equal(in, game.KeyBinds.A) || slices.Equal(in, game.KeyBinds.H) || slices.Equal(in, game.KeyBinds.LEFT)) {
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
		game.Screen.H.RenderString("Game", 2, 2, game.Objects.GameOver)
		game.Screen.H.RenderString("Over", 8, 8, game.Objects.GameOver)
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
	updateFramePlayer := max(1, 10-game.Config.PlayerSpeed)
	updateFramePea := max(1, game.Config.PeaSpawnDelay*(1000/game.Config.MinFrameTime))
	updateFramePlusOne := max(1, game.Config.PlusOneDelay*(1000/game.Config.MinFrameTime))

	for i := 1; !stopping; i++ {
		t := time.Now()

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

		game.Screen.Draw()
		game.statsBar(t)

		time.Sleep((time.Millisecond * time.Duration(game.Config.MinFrameTime)) - time.Now().Sub(t))
	}
}

func (game *Game) Start() error {
	for i := 0; i < game.Config.PeaStartCount; i++ {
		game.spawnPea()
	}

	game.Screen.Draw()
	game.loop()

	fmt.Print("\033[0;0H\r\n" + string(game.Screen.CharMap[game.Objects.Wall]))

	return nil
}

package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"slices"
	"strconv"
	"time"

	"golang.org/x/term"
)

type (
	Cord struct{ X, Y int }

	gameState struct {
		PlayerCrd     Cord
		PlayerDir     string
		PlayerCurDir  string
		PlayerSpeed   int
		PeaSpawnDelay int
		PeaSpawnLimit int
		PeaStartCount int
		PlusOneDelay  int
		TailCrds      []Cord
		PeaCrds       []Cord
	}

	keyBinds    struct{ ESC, CTRL_C, CTRL_D, W, D, S, A, UP, RIGHT, DOWN, LEFT []byte }
	gameObjects struct{ Default, Empty, Wall, Player, Pea, GameOver, PlusOne int8 }
)

var (
	Stopping        = false
	DynamicResizing = false

	MinFrameTime = 100

	startTime     = time.Now()
	plusOneActive = false

	originalTrm = func() *term.State {
		oldState, err := term.MakeRaw(0)
		if err != nil {
			panic(err)
		}
		return oldState
	}()

	KeyBinds = keyBinds{
		ESC:    []byte{27, 0, 0},
		CTRL_C: []byte{3, 0, 0},
		CTRL_D: []byte{4, 0, 0},
		W:      []byte{119, 0, 0},
		D:      []byte{100, 0, 0},
		S:      []byte{115, 0, 0},
		A:      []byte{97, 0, 0},
		UP:     []byte{27, 91, 65},
		RIGHT:  []byte{27, 91, 67},
		DOWN:   []byte{27, 91, 66},
		LEFT:   []byte{27, 91, 68},
	}

	GameObjects = gameObjects{
		Default:  -1,
		Empty:    0,
		Wall:     1,
		Player:   2,
		Pea:      3,
		GameOver: 4,
		PlusOne:  5,
	}
)

func statsBar(gs *gameState) {
	timeDiff := time.Now().Sub(startTime)
	timeStr := fmt.Sprintf("%02d:%02d:%02d", int(timeDiff.Hours()), int(timeDiff.Minutes())%60, int(timeDiff.Seconds())%60)

	fmt.Printf("\r\nTime Alive: %v	Peas Eaten: %v", timeStr, len(gs.TailCrds))
}

func listenKeys(gs *gameState) {
	defer term.Restore(0, originalTrm)

	for !Stopping {
		in := make([]byte, 3)
		_, err := os.Stdin.Read(in)
		if err != nil {
			panic(err)
		}

		if slices.Equal(in, KeyBinds.CTRL_C) || slices.Equal(in, KeyBinds.CTRL_D) || slices.Equal(in, KeyBinds.ESC) {
			Stopping = true
			return
		} else if gs.PlayerCurDir != "down" && (slices.Equal(in, KeyBinds.W) || slices.Equal(in, KeyBinds.UP)) {
			gs.PlayerDir = "up"
		} else if gs.PlayerCurDir != "left" && (slices.Equal(in, KeyBinds.D) || slices.Equal(in, KeyBinds.RIGHT)) {
			gs.PlayerDir = "right"
		} else if gs.PlayerCurDir != "up" && (slices.Equal(in, KeyBinds.S) || slices.Equal(in, KeyBinds.DOWN)) {
			gs.PlayerDir = "down"
		} else if gs.PlayerCurDir != "right" && (slices.Equal(in, KeyBinds.A) || slices.Equal(in, KeyBinds.LEFT)) {
			gs.PlayerDir = "left"
		}
	}
}

func updatePlayer(f *Frame, gs *gameState) {
	oldCords := gs.PlayerCrd

	switch gs.PlayerDir {
	case "up":
		gs.PlayerCrd.Y -= 1
	case "right":
		gs.PlayerCrd.X += 1
	case "down":
		gs.PlayerCrd.Y += 1
	case "left":
		gs.PlayerCrd.X -= 1
	}
	gs.PlayerCurDir = gs.PlayerDir

	if gs.PlayerCrd.X <= 0 {
		gs.PlayerCrd.X = f.CurX - 1
	} else if gs.PlayerCrd.X >= f.CurX {
		gs.PlayerCrd.X = 1
	} else if gs.PlayerCrd.Y <= 0 {
		gs.PlayerCrd.Y = f.CurY - 1
	} else if gs.PlayerCrd.Y >= f.CurY {
		gs.PlayerCrd.Y = 1
	}

	val, err := f.GetColRow(gs.PlayerCrd.X, gs.PlayerCrd.Y)
	if err != nil {
		panic(err)
	}
	if val == GameObjects.Player {
		Stopping = true
		failScreen(f)
		return
	}

	if val == GameObjects.Pea {
		gs.TailCrds = append(gs.TailCrds, Cord{X: oldCords.X, Y: oldCords.Y})
		plusOneActive = true
		plusOneScreen(f)

	} else {
		if len(gs.TailCrds) > 0 {
			gs.TailCrds = append(gs.TailCrds, Cord{X: oldCords.X, Y: oldCords.Y})
			oldCords = gs.TailCrds[0]
			gs.TailCrds = slices.Delete(gs.TailCrds, 0, 1)
		}
		f.SetColRow(oldCords.X, oldCords.Y, GameObjects.Empty)
	}

	f.SetColRow(gs.PlayerCrd.X, gs.PlayerCrd.Y, GameObjects.Player)
}

func spawnPea(f *Frame, gs *gameState) {
	cord := Cord{}
	for i := 1; i < 100; i++ {
		cord = Cord{X: rand.IntN(f.CurX-1) + 1, Y: rand.IntN(f.CurY-1) + 1}
		val, _ := f.GetColRow(cord.X, cord.Y)
		if val == GameObjects.Empty {
			gs.PeaCrds = append(gs.PeaCrds, cord)
			f.SetColRow(cord.X, cord.Y, GameObjects.Pea)
			break
		}
	}
}

func setup() (*Frame, *gameState, error) {
	f, err := NewFrame(1000, 1000, map[int8]string{
		GameObjects.Default:  Colors.Purple + "██" + Colors.Reset,
		GameObjects.Empty:    "  ",
		GameObjects.Wall:     Colors.Gray + "██" + Colors.Reset,
		GameObjects.Player:   "██",
		GameObjects.Pea:      Colors.Yellow + "██" + Colors.Reset,
		GameObjects.GameOver: Colors.Red + "██" + Colors.Reset,
		GameObjects.PlusOne:  Colors.Green + "██" + Colors.Reset,
	})
	if err != nil {
		return &Frame{}, &gameState{}, err
	}

	gs := &gameState{
		PlayerCrd:     Cord{X: int(f.CurX / 2), Y: int(f.CurY / 2)},
		PlayerDir:     "right",
		PlayerSpeed:   8, // Player moves 1 tile every `10-PlayerSpeed` updates.
		PeaSpawnDelay: 5,
		PeaSpawnLimit: 3,
		PeaStartCount: 1,
		PlusOneDelay:  1,
		TailCrds:      []Cord{},
		PeaCrds:       []Cord{},
	}

	f.SetCol(0, GameObjects.Wall)
	f.SetRow(0, GameObjects.Wall)
	f.SetCol(f.CurX, GameObjects.Wall)
	f.SetRow(f.CurY, GameObjects.Wall)

	f.SetColRow(gs.PlayerCrd.X, gs.PlayerCrd.Y, GameObjects.Player)

	go listenKeys(gs)

	return f, gs, nil
}

func loop(f *Frame, gs *gameState) {
	for i := 1; !Stopping; i++ {
		t := time.Now()

		if DynamicResizing {
			f.Reload()

			f.SetCol(0, GameObjects.Wall)
			f.SetRow(0, GameObjects.Wall)
			f.SetCol(f.CurX, GameObjects.Wall)
			f.SetRow(f.CurY, GameObjects.Wall)

			f.SetColRow(gs.PlayerCrd.X, gs.PlayerCrd.Y, GameObjects.Player)
			for _, cord := range gs.TailCrds {
				f.SetColRow(cord.X, cord.Y, GameObjects.Player)
			}

			for _, cord := range gs.PeaCrds {
				f.SetColRow(cord.X, cord.Y, GameObjects.Pea)
			}
		}

		if plusOneActive && i%max(1, gs.PlusOneDelay*(1000/MinFrameTime)) == 0 {
			plusOneActive = false
			plusOneScreen(f)
		}

		if i%max(1, 10-gs.PlayerSpeed) == 0 {
			updatePlayer(f, gs)
		}

		gs.PeaCrds = slices.DeleteFunc(gs.PeaCrds, func(cord Cord) bool {
			val, err := f.GetColRow(cord.X, cord.Y)
			return err != nil || val != GameObjects.Pea
		})

		if len(gs.PeaCrds) < gs.PeaSpawnLimit && i%max(1, gs.PeaSpawnDelay*(1000/MinFrameTime)) == 0 {
			spawnPea(f, gs)
		}

		f.Draw()

		statsBar(gs)
		time.Sleep((time.Millisecond * time.Duration(MinFrameTime)) - time.Now().Sub(t))
	}
}

func handleArgs(gs *gameState) bool {
	for _, help := range []string{"-h", "--help", "help"} {
		if slices.Contains(os.Args, help) {
			fmt.Printf("\r\nAnother game of Snake" +
				"\r\n\r\nMinimal recomended resolution: 66x16" +
				"\r\n\r\nArgs:" +
				"\r\n\t-ps --player-speed [0-10]" +
				"\r\n\t\tAjust the player speed (default: 8)\r\n" +
				"\r\n\t-sd --spawn-delay [0-X]" +
				"\r\n\t\tAjust the pea spawn delay in seconds (default: 5)\r\n" +
				"\r\n\t-sl --spawn-limit [0-X]" +
				"\r\n\t\tAjust the pea spawn limit (default: 3)\r\n" +
				"\r\n\t-sc --spawn-count [0-X]" +
				"\r\n\t\tAjust the starting pea count (default: 1)\r\n" +
				"\r\n\t-dr --dynamic-resizing" +
				"\r\n\t\tEnable dynamic resizing during gameplay, expirmental! (default: off)" +
				"\r\n\t\tRequires more resources to run smoothly." +
				"\r\n\t\tWhen downscaling peas will be despawned when out off bounds." +
				"\r\n\t\tWhen downscaling the player will be teleported to the new edge when out off bounds, this causes a game over if teleported inside itself.\r\n" +
				"\r\n",
			)
			return false
		}
	}

	for i, arg := range os.Args {
		if (arg == "-ps" || arg == "--player-speed") && len(os.Args) > i {
			v, err := strconv.Atoi(os.Args[i+1])
			if err != nil {
				panic(err)
			}
			gs.PlayerSpeed = v
			continue
		}

		if (arg == "-sd" || arg == "--spawn-delay") && len(os.Args) > i {
			v, err := strconv.Atoi(os.Args[i+1])
			if err != nil {
				panic(err)
			}
			gs.PeaSpawnDelay = v
			continue
		}

		if (arg == "-sl" || arg == "--spawn-limit") && len(os.Args) > i {
			v, err := strconv.Atoi(os.Args[i+1])
			if err != nil {
				panic(err)
			}
			gs.PeaSpawnLimit = v
			continue
		}

		if (arg == "-sc" || arg == "--spawn-count") && len(os.Args) > i {
			v, err := strconv.Atoi(os.Args[i+1])
			if err != nil {
				panic(err)
			}
			gs.PeaStartCount = v
			continue
		}

		if arg == "-dr" || arg == "--dynamic-resizing" {
			DynamicResizing = true
			continue
		}
	}

	return true
}

func main() {
	defer term.Restore(0, originalTrm)

	f, gs, err := setup()
	if err != nil {
		panic(err)
	}

	if !handleArgs(gs) {
		return
	}

	for i := 0; i < gs.PeaStartCount; i++ {
		spawnPea(f, gs)
	}

	f.Draw()
	loop(f, gs)

	fmt.Print("	")
}

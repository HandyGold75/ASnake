package game

import (
	"ASnake/screen"
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
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
	keyBinds struct {
		ESC, P,
		CTRL_C, CTRL_D, Q,
		W, D, S, A, K, L, J, H, UP, RIGHT, DOWN, LEFT []byte
	}

	Player struct {
		Crd         [2]int
		Dir, CurDir string
		TailCrds    [][2]int
		IsGameOver  bool
	}

	GameConfig struct {
		LockFPSToTPS                                                           bool
		Connection                                                             net.Conn
		ClientId                                                               string
		TargetTPS, TargetFPS                                                   int
		PlayerSpeed, PeaSpawnDelay, PeaSpawnLimit, PeaStartCount, PlusOneDelay int
	}
	GameState struct {
		Players       map[string]Player
		PeaCrds       [][2]int
		PlusOneActive bool
		TpsTracker    int
	}
	Game struct {
		KeyBinds   keyBinds
		Config     GameConfig
		State      GameState
		Screen     *screen.Screen
		StartTime  time.Time
		fpsTracker int
		stopping   bool
		paused     bool
	}

	FirstUpdatePacket struct {
		ClientId   string
		StartTime  time.Time
		MaxX, MaxY int
		State      GameState
	}
)

const (
	Reset string = "\033[0m"

	Black   string = "\033[30m"
	Red     string = "\033[31m"
	Green   string = "\033[32m"
	Yellow  string = "\033[33m"
	Blue    string = "\033[34m"
	Magenta string = "\033[35m"
	Cyan    string = "\033[36m"
	White   string = "\033[37m"
)

const (
	ObjEmpty uint8 = iota
	ObjWall
	ObjPlusOne
	ObjWarning
	ObjPea
	ObjPlayer
)

func NewGame(headless bool) (*Game, error) {
	maxX, maxY, forceMax := 2560, 1440, false
	if headless {
		maxX, maxY, forceMax = 50, 50, true
	}
	scr, err := screen.NewScreen(maxX, maxY, forceMax, map[uint8][]byte{
		ObjEmpty:   []byte("  "),
		ObjWall:    []byte(Black + "██" + Reset),
		ObjPlusOne: []byte(Green + "██" + Reset),
		ObjWarning: []byte(Red + "██" + Reset),
		ObjPea:     []byte(Yellow + "██" + Reset),
		ObjPlayer:  []byte(White + "██" + Reset),
	})
	if err != nil {
		return &Game{}, err
	}

	_ = scr.SetCol(0, ObjWall)
	_ = scr.SetRow(0, ObjWall)
	_ = scr.SetCol(scr.CurX, ObjWall)
	_ = scr.SetRow(scr.CurY, ObjWall)

	game := &Game{
		KeyBinds: keyBinds{
			ESC: []byte{27, 0, 0}, P: []byte{112, 0, 0},
			CTRL_C: []byte{3, 0, 0}, CTRL_D: []byte{4, 0, 0}, Q: []byte{113, 0, 0},
			W: []byte{119, 0, 0}, D: []byte{100, 0, 0}, S: []byte{115, 0, 0}, A: []byte{97, 0, 0},
			K: []byte{107, 0, 0}, L: []byte{108, 0, 0}, J: []byte{106, 0, 0}, H: []byte{104, 0, 0},
			UP: []byte{27, 91, 65}, RIGHT: []byte{27, 91, 67}, DOWN: []byte{27, 91, 66}, LEFT: []byte{27, 91, 68},
		},
		Config: GameConfig{
			LockFPSToTPS:  false,
			ClientId:      "0",
			TargetTPS:     30,
			TargetFPS:     60,
			PlayerSpeed:   5,
			PeaSpawnDelay: 5,
			PeaSpawnLimit: 4,
			PeaStartCount: 2,
			PlusOneDelay:  1,
		},
		State: GameState{
			Players:       map[string]Player{},
			PeaCrds:       [][2]int{},
			PlusOneActive: false,
			TpsTracker:    0,
		},
		Screen:     scr,
		StartTime:  time.Now(),
		fpsTracker: 0,
		stopping:   false,
		paused:     false,
	}

	if headless {
		return game, nil
	}

	game.State.Players = map[string]Player{"0": {
		Crd: [2]int{int(game.Screen.CurX / 2), int(game.Screen.CurY / 2)},
		Dir: "right", CurDir: "right",
		TailCrds: [][2]int{},
	}}
	_ = game.Screen.SetColRow(game.State.Players[game.Config.ClientId].Crd[0], game.State.Players[game.Config.ClientId].Crd[1], ObjPlayer)

	game.Screen.OnResizeCallback = func(scr *screen.Screen) {
		_ = scr.SetCol(0, ObjWall)
		_ = scr.SetRow(0, ObjWall)
		_ = scr.SetCol(scr.CurX, ObjWall)
		_ = scr.SetRow(scr.CurY, ObjWall)

		if game.State.PlusOneActive {
			game.Screen.RenderStringIf("+", 2, 2, ObjPlusOne, func(val uint8) bool { return val == ObjEmpty })
			game.Screen.RenderStringIf("1", 8, 2, ObjPlusOne, func(val uint8) bool { return val == ObjEmpty })
		}

		if game.paused {
			game.Screen.RenderStringIf("Paused", 2, 2, ObjWarning, func(val uint8) bool { return val < ObjPlayer })
		}

		_ = scr.SetColRow(game.State.Players[game.Config.ClientId].Crd[0], game.State.Players[game.Config.ClientId].Crd[1], ObjPlayer)

		for _, cord := range game.State.Players[game.Config.ClientId].TailCrds {
			_ = scr.SetColRow(cord[0], cord[1], ObjPlayer)
		}
		for _, cord := range game.State.PeaCrds {
			_ = scr.SetColRow(cord[0], cord[1], ObjPea)
		}
		_ = game.Screen.Draw()
	}

	return game, nil
}

func (game *Game) statsBar() {
	timeDiff := time.Since(game.StartTime)
	timeStr := fmt.Sprintf("%02d:%02d:%02d:%03d", int(timeDiff.Hours()), int(timeDiff.Minutes())%60, int(timeDiff.Seconds())%60, int(timeDiff.Milliseconds())%1000)

	sizeXColor := ""
	if game.Config.Connection != nil && game.Screen.CurX < game.Screen.MaxX {
		sizeXColor = Red
	}
	sizeYColor := ""
	if game.Config.Connection != nil && game.Screen.CurY < game.Screen.MaxY {
		sizeYColor = Red
	}

	fpsColor := ""
	if !game.Config.LockFPSToTPS && game.fpsTracker < game.Config.TargetFPS-(game.Config.TargetFPS/5) {
		fpsColor = Red
	}
	tpsColor := ""
	if game.State.TpsTracker < game.Config.TargetTPS {
		tpsColor = Red
	}

	msg := fmt.Sprintf("Time: %v   Peas: %v   Size: %vx %vy   FPS: %v   TPS: %v ",
		timeStr,
		len(game.State.Players[game.Config.ClientId].TailCrds),
		sizeXColor+strconv.Itoa(game.Screen.CurX)+Reset,
		sizeYColor+strconv.Itoa(game.Screen.CurY)+Reset,
		fpsColor+strconv.Itoa(game.fpsTracker)+Reset,
		tpsColor+strconv.Itoa(game.State.TpsTracker)+Reset,
	)

	if len([]rune(msg)) > game.Screen.CurX*2 {
		fmt.Printf("\033[2K\r%."+strconv.Itoa(game.Screen.CurX*2)+"s...", msg)
	} else {
		fmt.Printf("\033[2K\r%."+strconv.Itoa(game.Screen.CurX*2)+"s", msg)
	}
}

func (game *Game) HandleInput(in []byte) error {
	playerState := game.State.Players[game.Config.ClientId]
	if slices.Equal(in, game.KeyBinds.CTRL_C) || slices.Equal(in, game.KeyBinds.CTRL_D) || slices.Equal(in, game.KeyBinds.Q) {
		game.stopping = true
		return nil

	} else if playerState.IsGameOver {
		return nil
	} else if (slices.Equal(in, game.KeyBinds.ESC) || slices.Equal(in, game.KeyBinds.P)) && game.Config.Connection == nil {
		game.paused = !game.paused
		if game.paused {
			game.Screen.RenderStringIf("Paused", 2, 2, ObjWarning, func(val uint8) bool { return val < ObjPlayer })
		} else {
			game.Screen.RenderStringIf("Paused", 2, 2, ObjEmpty, func(val uint8) bool { return val < ObjPlayer })
		}
		err := game.Screen.Draw()
		if err != nil {
			return err
		}
		return nil
	}

	dir := playerState.Dir
	if !game.paused && playerState.CurDir != "down" && (slices.Equal(in, game.KeyBinds.W) || slices.Equal(in, game.KeyBinds.K) || slices.Equal(in, game.KeyBinds.UP)) {
		dir = "up"
	} else if !game.paused && playerState.CurDir != "left" && (slices.Equal(in, game.KeyBinds.D) || slices.Equal(in, game.KeyBinds.L) || slices.Equal(in, game.KeyBinds.RIGHT)) {
		dir = "right"
	} else if !game.paused && playerState.CurDir != "up" && (slices.Equal(in, game.KeyBinds.S) || slices.Equal(in, game.KeyBinds.J) || slices.Equal(in, game.KeyBinds.DOWN)) {
		dir = "down"
	} else if !game.paused && playerState.CurDir != "right" && (slices.Equal(in, game.KeyBinds.A) || slices.Equal(in, game.KeyBinds.H) || slices.Equal(in, game.KeyBinds.LEFT)) {
		dir = "left"
	}

	if game.Config.Connection != nil {
		_, err := game.Config.Connection.Write([]byte(dir + "\n"))
		return err
	}

	playerState.Dir = dir
	game.State.Players[game.Config.ClientId] = playerState
	return nil
}

func (game *Game) UpdatePlayer(id string) {
	playerState := game.State.Players[id]

	oldCords := playerState.Crd
	switch playerState.Dir {
	case "up":
		playerState.Crd[1] -= 1
	case "right":
		playerState.Crd[0] += 1
	case "down":
		playerState.Crd[1] += 1
	case "left":
		playerState.Crd[0] -= 1
	}
	playerState.CurDir = playerState.Dir

	if playerState.Crd[0] <= 0 {
		playerState.Crd[0] = game.Screen.CurX - 1
	} else if playerState.Crd[0] >= game.Screen.CurX {
		playerState.Crd[0] = 1
	} else if playerState.Crd[1] <= 0 {
		playerState.Crd[1] = game.Screen.CurY - 1
	} else if playerState.Crd[1] >= game.Screen.CurY {
		playerState.Crd[1] = 1
	}

	val, err := game.Screen.GetColRow(playerState.Crd[0], playerState.Crd[1])
	if err != nil {
		game.State.Players[id] = playerState
		return
	}

	if val == ObjPlayer {
		playerState.Crd = oldCords
		playerState.IsGameOver = true
		game.Screen.RenderString("Game", 2, 2, ObjWarning)
		game.Screen.RenderString("Over", 8, 8, ObjWarning)

		game.State.Players[id] = playerState
		return
	}

	if val == ObjPea {
		game.State.PeaCrds = slices.DeleteFunc(game.State.PeaCrds, func(cord [2]int) bool {
			return cord == playerState.Crd
		})

		playerState.TailCrds = append(playerState.TailCrds, oldCords)

		game.State.PlusOneActive = true
		game.Screen.RenderStringIf("+", 2, 2, ObjPlusOne, func(val uint8) bool { return val == ObjEmpty })
		game.Screen.RenderStringIf("1", 8, 2, ObjPlusOne, func(val uint8) bool { return val == ObjEmpty })

	} else {
		if len(playerState.TailCrds) > 0 {
			playerState.TailCrds = append(playerState.TailCrds, oldCords)
			oldCords = playerState.TailCrds[0]
			playerState.TailCrds = slices.Delete(playerState.TailCrds, 0, 1)
		}
		_ = game.Screen.SetColRow(oldCords[0], oldCords[1], ObjEmpty)
	}

	_ = game.Screen.SetColRow(playerState.Crd[0], playerState.Crd[1], ObjPlayer)
	game.State.Players[id] = playerState
}

func (game *Game) SpawnPea() {
	for i := 1; i < 100; i++ {
		cord := [2]int{rand.IntN(game.Screen.CurX-1) + 1, rand.IntN(game.Screen.CurY-1) + 1}
		val, _ := game.Screen.GetColRow(cord[0], cord[1])
		if val == ObjEmpty {
			game.State.PeaCrds = append(game.State.PeaCrds, cord)
			_ = game.Screen.SetColRow(cord[0], cord[1], ObjPea)
			break
		}
	}
}

func (game *Game) loopSingle(iteration int) {
	now := time.Now()
	updateFramePlayer := max(1, game.Config.TargetTPS/game.Config.PlayerSpeed)
	updateFramePea := max(1, game.Config.PeaSpawnDelay*game.Config.TargetTPS)
	updateFramePlusOne := max(1, game.Config.PlusOneDelay*game.Config.TargetTPS)

	if game.State.PlusOneActive && iteration%updateFramePlusOne == 0 {
		game.State.PlusOneActive = false
		game.Screen.RenderStringIf("+", 2, 2, ObjEmpty, func(val uint8) bool { return val == ObjPlusOne })
		game.Screen.RenderStringIf("1", 8, 2, ObjEmpty, func(val uint8) bool { return val == ObjPlusOne })
	}

	if game.paused || game.State.Players[game.Config.ClientId].IsGameOver {
		time.Sleep((time.Second / time.Duration(game.Config.TargetTPS)) - time.Since(now))
		game.State.TpsTracker = int(time.Second/time.Since(now)) + 1
		game.StartTime = game.StartTime.Add(time.Since(now))

		if game.paused && game.Config.LockFPSToTPS {
			game.fpsTracker = game.State.TpsTracker
			game.statsBar()

		}
		return
	}

	if iteration%updateFramePlayer == 0 {
		_ = game.Screen.SetCol(0, ObjWall)
		_ = game.Screen.SetRow(0, ObjWall)
		_ = game.Screen.SetCol(game.Screen.CurX, ObjWall)
		_ = game.Screen.SetRow(game.Screen.CurY, ObjWall)

		game.UpdatePlayer("0")
	}

	if iteration%updateFramePea == 0 {
		game.State.PeaCrds = slices.DeleteFunc(game.State.PeaCrds, func(cord [2]int) bool {
			val, err := game.Screen.GetColRow(cord[0], cord[1])
			return err != nil || val != ObjPea
		})

		if len(game.State.PeaCrds) < game.Config.PeaSpawnLimit {
			game.SpawnPea()
		}
	}

	if game.Config.LockFPSToTPS {
		_ = game.Screen.Draw()
		time.Sleep((time.Second / time.Duration(game.Config.TargetTPS)) - time.Since(now))
		game.State.TpsTracker = int(time.Second/time.Since(now)) + 1

		game.fpsTracker = game.State.TpsTracker
		game.statsBar()
		return
	}

	time.Sleep((time.Second / time.Duration(game.Config.TargetTPS)) - time.Since(now))
	game.State.TpsTracker = int(time.Second/time.Since(now)) + 1
}

func (game *Game) loopMulti() {
	reader := bufio.NewReader(game.Config.Connection)
	for !game.stopping {
		msg, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				continue
			}
			return
		}
		msg = strings.ReplaceAll(msg, "\n", "")

		err = json.Unmarshal([]byte(msg), &game.State)
		if err != nil {
			panic(err)
		}

		for i := 0; i <= game.Screen.CurX; i++ {
			_ = game.Screen.SetCol(i, ObjEmpty)
		}
		for i := 0; i <= game.Screen.CurY; i++ {
			_ = game.Screen.SetRow(i, ObjEmpty)
		}

		_ = game.Screen.SetCol(0, ObjWall)
		_ = game.Screen.SetRow(0, ObjWall)
		_ = game.Screen.SetCol(game.Screen.CurX, ObjWall)
		_ = game.Screen.SetRow(game.Screen.CurY, ObjWall)

		if game.State.PlusOneActive {
			game.Screen.RenderStringIf("+", 2, 2, ObjPlusOne, func(val uint8) bool { return val == ObjEmpty })
			game.Screen.RenderStringIf("1", 8, 2, ObjPlusOne, func(val uint8) bool { return val == ObjEmpty })
		} else {
			game.Screen.RenderStringIf("+", 2, 2, ObjEmpty, func(val uint8) bool { return val == ObjPlusOne })
			game.Screen.RenderStringIf("1", 8, 2, ObjEmpty, func(val uint8) bool { return val == ObjPlusOne })
		}

		for _, peaCrd := range game.State.PeaCrds {
			_ = game.Screen.SetColRow(peaCrd[0], peaCrd[1], ObjPea)
		}

		for _, player := range game.State.Players {
			_ = game.Screen.SetColRow(player.Crd[0], player.Crd[1], ObjPlayer)
			for _, tailCrd := range player.TailCrds {
				_ = game.Screen.SetColRow(tailCrd[0], tailCrd[1], ObjPlayer)
			}
		}

		if game.State.Players[game.Config.ClientId].IsGameOver {
			game.Screen.RenderString("Game", 2, 2, ObjWarning)
			game.Screen.RenderString("Over", 8, 8, ObjWarning)
		}

		if game.Config.LockFPSToTPS {
			_ = game.Screen.Draw()

			game.fpsTracker = game.State.TpsTracker
			game.statsBar()
		}
	}
}

func (game *Game) Start() error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return errors.New("stdin/ stdout should be a terminal")
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	go func() {
		defer term.Restore(int(os.Stdin.Fd()), oldState)
		for {
			in := make([]byte, 3)
			_, err := os.Stdin.Read(in)
			if err != nil {
				fmt.Println(err)
				return
			}
			err = game.HandleInput(in)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}()

	game.StartTime = time.Now()
	if !game.Config.LockFPSToTPS {
		// TODO: Stop render updates during game updates and vice virsa due to possible race condition.
		// Error: `fatal error: concurrent map read and map write`
		// Repeatable when using high TargetTPS and/ or TargetFPS.
		go func() {
			defer func() { game.stopping = true; _ = term.Restore(int(os.Stdin.Fd()), oldState) }()
			for !game.stopping {
				now := time.Now()

				_ = game.Screen.Draw()

				time.Sleep((time.Second / time.Duration(game.Config.TargetFPS)) - time.Since(now))
				game.fpsTracker = int(time.Second/time.Since(now)) + 1
				if !game.State.Players[game.Config.ClientId].IsGameOver {
					game.statsBar()
				}

			}
		}()
	}

	if game.Config.Connection != nil {
		game.loopMulti()
	} else {
		for i := 0; i < game.Config.PeaStartCount; i++ {
			game.SpawnPea()
		}
		for i := 1; !game.stopping; i++ {
			game.loopSingle(i)
		}
	}

	fmt.Print("\033[0;0H\r\n" + string(game.Screen.CharMap[ObjWall]))
	if game.Config.Connection != nil {
		_ = game.Config.Connection.Close()
	}
	return nil
}

package ASnakeClient

import (
	"ASnake/client/game"
	"ASnake/client/menu"
	"os"

	"golang.org/x/term"
)

var (
	originalTrm = &term.State{}

	onKeyEvent = func([]byte) {}
	stopping   = false
)

func listenKeys() {
	defer term.Restore(int(os.Stdin.Fd()), originalTrm)

	for !stopping {
		in := make([]byte, 3)
		_, err := os.Stdin.Read(in)
		if err != nil {
			panic(err)
		}
		onKeyEvent(in)
	}
}

func Run() {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	originalTrm = oldState
	defer func() { stopping = true; term.Restore(int(os.Stdin.Fd()), originalTrm) }()

	gm, err := game.NewClient(originalTrm)
	if err != nil {
		panic(err)
	}
	mn, err := menu.NewMenu(gm)
	if err != nil {
		panic(err)
	}

	go listenKeys()

	onKeyEvent = mn.HandleInput
	if out := mn.Start(); out == "Start" {
		onKeyEvent = gm.HandleInput
		gm.Start()
	}
}

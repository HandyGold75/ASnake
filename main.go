package main

import (
	ASnakeClient "ASnake/client"
	ASnakeServer "ASnake/server"

	"github.com/HandyGold75/GOLib/argp"
)

var args = argp.ParseArgs(struct {
	Help   bool `switch:"h,-help"   opts:"help" help:"Another game of Snake."`
	Server bool `switch:"s,-server"             help:"Start as a server instace."`
}{})

func main() {
	if args.Server {
		ASnakeServer.Run()
	} else {
		ASnakeClient.Run()
	}
}

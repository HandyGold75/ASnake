package main

import (
	ASnakeClient "ASnake/client"
	ASnakeServer "ASnake/server"
	"os"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		os.Stdout.WriteString("Another game of Snake.\r\nUse -s or --server to start a server instance.\r\n")
	} else if len(os.Args) > 1 && (os.Args[1] == "-s" || os.Args[1] == "--server") {
		ASnakeServer.Run()
	} else {
		ASnakeClient.Run()
	}
}

package ASnakeServer

import "ASnake/server/tcp"

func Run() {
	tcp.NewServer("127.0.0.1", 17530).Run()
}

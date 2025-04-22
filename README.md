# Another game of Snake

Just another game of snake.

Works with any terminal that supports 3-bit color.

Minimal recommended play area: 50x 25y.
Minimal required play area (Multiplayer): 50x 50y.

## Args

```text
Usage: ASnake [-h] [-s] [-i <string>] [-p <uint16>] [-m <int>]
        Another game of Snake.

Help
  -h --help         <bool>    (help)
Server
  -s --server       <bool>
        Start as a server instace.
IP
  -i --ip           <string>
        Listen on this ip when started as server.
Port
  -p --port         <uint16>
        Listen on this port when started as server.
MaxClients
  -m --max-clients  <int>
        Max amount of clients per pool.
```

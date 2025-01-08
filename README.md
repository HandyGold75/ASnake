# Another game of Snake

Just another game of snake.

Works with POSIX compliant terminals.

Minimal recommended play area: 50x 25y.
Minimal required play area (Multiplayer): 50x 50y.

## Config

- Player Speed (Speed of the player).
  - Min: `1`; Max: `Target TPS`
  - In multiplayer controlled by the server.
  - Higher TPS does not mean an higher player speed, but does mean an higher player speed limit.
  - If `Target TPS`/ `Player Speed` is not an rounded number then player movement might become choppy.
- Spawn Delay (Delay between peas spawning in seconds).
  - Min: `0`; Max: `99999`
  - In multiplayer controlled by the server.
- Spawn Limit (Stop spawing peas past this limit).
  - Min: `0`; Max: `99999`
  - In multiplayer controlled by the server.
- Spawn Count (Start with this amount of peas).
  - Min: `0`; Max: `99999`
  - In multiplayer controlled by the server.
  - Ignores the spawn limit.
- Low Performance (Only update frame when game has updated).
  - Values: `Yes`, `No`
  - Locks FPS to TPS by only updating the frame when the game is updated.
  - In singleplayer this equates to TPS.
  - In multiplayer this equates to every time the game state changes with a limit of TPS.
- Target TPS (Max game updates per second).
  - Min: `0`; Max: `99999`
  - In multiplayer controlled by the server.
  - If target TPS can not be achived then the game will run slower in actual time.
- Target FPS (Max frame updates per second).
  - Min: `0`; Max: `99999`
  - Ignored if Low Performance is used.
  - Some terminals don't like fast rendering and might flikker when set to high.

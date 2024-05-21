# Another game of Snake

Meant to run under Linux inside of bash.
In some circumstances, the game can flicker a lot.

Minimal recommended resolution: 66x16

Args:

```text
    -ps --player-speed [0-10]
        Adjust the player speed (default: 8)

    -sd --spawn-delay [0-X]
        Adjust the pea spawn delay in seconds (default: 5)

    -sl --spawn-limit [0-X]
        Adjust the pea spawn limit (default: 3)

    -sc --spawn-count [0-X]
        Adjust the starting pea count (default: 1)

    -dr --dynamic-resizing
        Enable dynamic resizing during gameplay, experimental! (default: off)
        Requires more resources to run smoothly.
        When downscaling peas will be despawned when out of bounds.
        When downscaling the player will be teleported to the new edge when out of bounds, this causes a game over if teleported inside itself.
```

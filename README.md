# Another game of Snake

Minimal recomended resolution: 66x16

Args:

```text
    -ps --player-speed [0-10]
        Ajust the player speed (default: 8)

    -sd --spawn-delay [0-X]
        Ajust the pea spawn delay in seconds (default: 5)

    -sl --spawn-limit [0-X]
        Ajust the pea spawn limit (default: 3)

    -sc --spawn-count [0-X]
        Ajust the starting pea count (default: 1)

    -dr --dynamic-resizing
        Enable dynamic resizing during gameplay, expirmental! (default: off)
        Requires more resources to run smoothly.
        When downscaling peas will be despawned when out off bounds.
        When downscaling the player will be teleported to the new edge when out off bounds, this causes a game over if teleported inside itself.
```

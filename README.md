# Another game of Snake

Works with POSIX compliant terminals.

Minimal recommended play area: 32x 14y
Delay won't impact gameplay while under 100ms.
If the delay goes above 100ms please decrease the play area you madlad.

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
```

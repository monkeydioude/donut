# Map `world`: World, 2048 x 1024

The starter map (ADR 0007). Every PNG is the map's size, 1 pixel per cell, column 0 on the left and row 0 at
the top. Each pixel is fully transparent or one of the exact colors below: no antialiasing, no half
transparency. Paint with a pencil, or in indexed mode with `palette.gpl` (GIMP, Aseprite).

- `make check-world` checks the folder.
- `make reset-world`, with the server stopped, replaces the world in the database with it.
- `make world` generates it again from its seed: it refuses to overwrite hand edits without `FORCE=1`.

`map.json` holds the name, size, wrapping and regions, each with its color in `regions.png`.

## `ground.png`

| Color | Means |
|-------|-------|
| transparent | void |
| `#74d3a5` | grass |
| `#6fb7e0` | water |
| `#2b2540` | hole |
| `#b98a5a` | bridge |
| `#4f9fd8` | ocean |

## `height.png`

| Color | Means |
|-------|-------|
| `#202020` | height -3 |
| `#404040` | height -2 |
| `#606060` | height -1 |
| `#808080` | height 0 |
| `#a0a0a0` | height 1 |
| `#c0c0c0` | height 2 |
| `#e0e0e0` | height 3 |

## `regions.png`

| Color | Means |
|-------|-------|
| transparent | no region |
| `#7fd99a` | Sprinkle Meadow |
| `#6cc9a0` | Glaze Hills |
| `#8adfae` | Cinnamon Vale |
| `#68cfb0` | Maple Hollow |
| `#93d99c` | Honey Downs |
| `#74c793` | Sugar Fields |
| `#82e0bd` | Cocoa Glen |
| `#60c2a1` | Frosting Flats |
| `#9ce3b0` | Crumb Valley |
| `#6fd6bf` | Jelly Ridge |
| `#86cf8f` | Cruller Heath |
| `#79bd9f` | Powder Plains |
| `#0d9977` | Caramel Shore |
| `#0e9977` | Nougat Cliffs |
| `#0f9977` | Praline Woods |
| `#109977` | Glaze Sea |
| `#119977` | Syrup Sea |
| `#129977` | Custard Sea |

## `objects.png`

| Color | Means |
|-------|-------|
| transparent | nothing |
| `#d2cec5` | rock |

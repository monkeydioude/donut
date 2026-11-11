# Map `doma-1`: Doma-1, 32 x 32

A map built into the server (ADR 0007, ADR 0012). Every PNG is the map's size, 1 pixel per cell, column 0 on
the left and row 0 at the top. Each pixel is fully transparent or one of the exact colors below: no antialiasing,
no half transparency. Paint with a pencil, or in indexed mode with `palette.gpl` (GIMP, Aseprite).

- `go run ./cmd/mapfile check mapfiles/<slug>`, from `back/`, checks a folder (`make check-world` the world's).
- `make reset-maps`, with the server stopped, replaces every map in the database with its folder.
- `make world` and `make city` generate the world and Doma-1 again: they refuse to overwrite hand edits
  without `FORCE=1`.

`map.json` holds the name, size, wrapping and regions, each with its color in `regions.png`, and the domes:
each one's back cell (its 2 x 2 footprint's lowest column and row) and the map it leads to.

## `ground.png`

| Color | Means |
|-------|-------|
| transparent | void |
| `#74d3a5` | grass |
| `#6fb7e0` | water |
| `#2b2540` | hole |
| `#b98a5a` | bridge |
| `#4f9fd8` | ocean |
| `#8a8694` | mountain |
| `#dff4fb` | glacier |
| `#d9d2c3` | floor |
| `#6f6a7a` | wall |

## `height.png`

| Color | Means |
|-------|-------|
| `#1e3a8a` | height -3 |
| `#2563eb` | height -2 |
| `#38bdf8` | height -1 |
| `#4ade80` | height 0 |
| `#facc15` | height 1 |
| `#f97316` | height 2 |
| `#dc2626` | height 3 |
| `#d8b4fe` | height 4 |
| `#a855f7` | height 5 |
| `#6b21a8` | height 6 |

## `regions.png`

| Color | Means |
|-------|-------|
| transparent | no region |
| `#7fd99a` | Doma-1 |

## `objects.png`

| Color | Means |
|-------|-------|
| transparent | nothing |
| `#d2cec5` | rock |

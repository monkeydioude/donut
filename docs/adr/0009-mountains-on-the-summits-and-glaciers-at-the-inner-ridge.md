# ADR 0009: Mountains on the summits and glaciers at the inner ridge

- Status: accepted, 2026-11-11; its mountains, glaciers and their drawing replaced by ADR 0011
- Replaces nothing. Adds to ADR 0007 and ADR 0008.

## Context

The world has hills, water, holes and rocks, but nothing that reads as a landscape from afar: a hilltop is grass like
the plain. We want two unwalkable ground kinds that mark places: bare rock in packs on the high hills, and ice at
the inner ridge of the donut, the line that faces the hole and gets the least sun, so the coldest.

This is the last change to the generator for now.

## Decisions

### What a cell holds

- Two new ground kinds:

  | Code | Kind       | Walkable | Notes                                                       |
  |------|------------|----------|-------------------------------------------------------------|
  | 6    | `mountain` | no       | bare rock on a summit, drawn; it keeps the cell's height    |
  | 7    | `glacier`  | no       | ice at the inner ridge, drawn; it keeps the cell's height   |

  The chunk format doesn't change. Rocks still stand only on grass, so never on them.
- Naming: the ground kind is `mountain`, not `rock`, because `rock` is already the boulder object: the map folder's
  README and palette would list "rock" twice, meaning two things.
- Clicking a mountain or a glacier works like clicking water: the hover shows, the menu opens, and Move is off.
- Walking, steps, areas and the spawn rule don't change.

### The generated world

Both come after the holes and before the rocks, on grass only: never on water, a hole, a bridge, nor a bridge's
kept ends.

- **Mountains**, in packs on the summits:
  - A **summit** is a connected group (8 neighbors, across the seams) of grass cells at level 3, of at least 4
    cells. Smaller ones get nothing.
  - Each summit gets a **pack**: blobs of 3 to 10 cells, grown like holes from random cells of the summit, on the
    summit's grass only, until about 60% of the summit's cells are `mountain`. A blob is cut so the pack never goes
    past that share, and the pack stops when fewer than 3 cells are left to take; a blob that can't reach 3 cells is
    dropped, so no mountain cell stands alone. After 4 tries per cell of the summit, the pack stays smaller.
  - A mountain cell keeps its height, 3: the pack is the hilltop itself.
- **Glaciers**, at the inner ridge:
  - The **ridge** is row height / 2, the row that faces the hole (ADR 0003). The **cold band** is the 80 rows on each
    side of it.
  - **Count**: 4 glaciers per chunk of the band, on average: 384 chunks on the starter world, so about 1,540.
  - Each glacier is a blob of 6 to 24 cells, grown like holes from a seed at a random column, and at a row drawn as
    the ridge plus the difference of two draws in 0…80: a triangle, densest on the ridge, zero at the band's edges.
    A seed on the sea, water, a hole, a mountain or ice is skipped, so the sea stays open and the count is a ceiling.
  - A glacier stays within the band and at least 48 cells from the spawn: the spawn's continent is centered on the
    ridge (ADR 0008), so without that the spawn, its river and its bridge would be iced in.
  - A glacier cell keeps its height.
- **Rocks**: as before, on grass only.
- **Regions**: unchanged.

### The starter map file

- `ground.png` gets mountain `#8a8694` and glacier `#dff4fb`. `README.md` and `palette.gpl` list them.
- The folder is regenerated: `make world FORCE=1`.
- **Migration `0005`** deletes every map, so the next boot loads the new file. Rooms keep the map they loaded, so
  the server is restarted.

### Drawing

- Both are baked into the chunk's ground texture, like holes and the ocean, with cliff faces toward lower front
  neighbors like any cell, in their own material: **stone** (gray) under a mountain, **ice** (pale blue) under a
  glacier.
- **Mountain**: a gray stone top with darker cracks, and a dark outline on each edge toward a cell that isn't
  mountain, so a pack reads as one massif. A **peak** rises from the cell's middle 6 px into the frame's headroom:
  a triangle, lit on its left, dark on its right, with a snow cap on its top 2 px. Peaks only ever overlap the cells
  behind them, which are drawn first.
- **Glacier**: an ice block. Its top is the ground diamond raised by 3 px, in pale ice blue with lighter crack lines;
  the 3 px under its front edges (se, sw) are the block's sides, a deeper blue, darker on the right. The top has a
  white rim on each edge toward a cell that isn't glacier.
- Neither gets a grass rim: rims mark grass tops only (ADR 0007).

### The map screen

- The image gets mountain `#8a8694` and glacier `#dff4fb`. The palette grows by two colors, to 190.

### Websocket

- `chunk` keeps its shape. `kinds` may now hold `mountain` and `glacier`. The client drops a chunk with an unknown
  kind, as before.

### Logs

- The generator's cells per ground kind (ADR 0007) now show `mountain` and `glacier`. Nothing more.

### How it is built

- Back:
  - `worldmap/domain`:
    - `map.go`: `TileMountain`, `TileGlacier`;
    - `mountains.go`: summits and packs; `glaciers.go`: the cold band;
    - `generate.go`: `MountainOptions` and `GlacierOptions`, run after the holes;
    - `image.go`: their colors.
  - `worldmap/infra/maplayers.go`: their colors in `ground.png`.
  - `migrations/up/0005.up.sql`.
- Client:
  - `map/worldMap.ts`: the new kinds.
  - `art/mountain.ts`, `art/glacier.ts`: the tiles; `art/cliff.ts`: the stone and ice faces.
  - `art/tileArtSet.ts`, `render/tileGraphics.ts`, `render/chunkView.ts`: wiring them in, with their edges.

### Security review

- The chunk decoder accepts code 7 and still refuses anything above it. Rocks on a mountain or a glacier are refused
  when a map is checked. The client drops a chunk with a kind it doesn't know.
- The generator runs only from `mapfile generate`. Summit packs stop after 4 tries per summit cell, and glaciers are a
  fixed count of bounded blobs, so a seed can't make it loop.
- The terrain's sprites are still kept by their edges only: 16 mountains and 16 glaciers at most.

### Not now

- Sea ice: glaciers over the ocean at the ridge.
- Snow on the grass near the ridge, or a cold biome.
- Mountains that rise higher than level 3, and climbing them.
- A warm band at the outer equator: sand, a desert.

## Test cases

### Back: map rules and walking (`worldmap/domain`)

1. Codes 6 and 7 decode as `mountain` and `glacier`; code 8 is refused. A rock on either is refused.
2. Neither is walkable; a path never crosses them.

### Back: the generator (`worldmap/domain`)

3. Mountains:
   - some exist, every one at level 3, and none is alone: each touches another mountain cell (8 neighbors);
   - over the summits (the groups of grass and mountain cells at level 3) of 4 cells or more, mountains cover 40
     to 75% of the cells.
4. Glaciers:
   - some exist, all within 80 rows of row 512 (the ridge), none within 48 cells of the spawn;
   - more glacier cells lie within 20 rows of the ridge than in the 20 rows past 60.
5. Generated without mountain nor glacier options, the same seed gives the same ground and heights, but every
   mountain and glacier cell is grass: none are made, and they only ever replace grass.
6. Rocks are still on grass only, so never on a mountain or a glacier.
7. The generated world passes the map file's checks, and the spawn's bridge is still reached from the spawn (ADR 0008
   test 9).

### Back: the map file, boot and migration

8. A map with mountains and glaciers writes and reads back the same: 7 ground kinds. The README lists their colors.
9. The committed `mapfiles/world` passes every check.
10. Migration `0005` deletes every map.

### Back: the map image

11. Mountains and glaciers have their colors. The palette has 190 colors.

### Client

12. Protocol: a `chunk` with `mountain` and `glacier` decodes.
13. Both are baked into the ground, with edges toward cells of another kind only, and faces toward a lower front
    neighbor in their material: stone under a mountain, ice under a glacier.
14. Mountain tile: stone top with cracks, an outline on the edges given only, and a peak rising into the headroom,
    lit left and dark right, with snow on top.
15. Glacier tile: the top raised 3 px above the ground diamond, sides under its front edges, a white rim on the edges
    given only, and nothing drawn above the raised top.

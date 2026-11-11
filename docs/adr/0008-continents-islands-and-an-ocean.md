# ADR 0008: Continents, islands and an ocean, with lakes and descending rivers

- Status: accepted, 2026-11-10
- Replaces, from ADR 0007:
  - the generated world's size (1024 × 512) and "The river" (one loop around the ring);
  - "Regions: as before (ADR 0003), 12 points";
  - "Lakes" in "Not now".

## Context

The starter world is one landmass with a river looping around the ring. We want a world: continents and islands in
an ocean, with lakes and rivers of their own, so the dome city has a coast, a hinterland and other shores to reach
later. The map is also too small for that to feel big: a continent would be 250 cells across.

## Decisions

### What a cell holds

- A new ground kind:

  | Code | Kind    | Walkable | Notes                                              |
  |------|---------|----------|----------------------------------------------------|
  | 5    | `ocean` | no       | the sea, drawn still; `water` stays rivers and lakes |

  The chunk format doesn't change. Rocks still stand only on grass.
- Clicking the ocean works like clicking water: the hover shows, the menu opens, and Move is off.
- Walking, steps, areas and the spawn rule don't change: the spawn is still the walkable cell closest to the center.

### The generated world

The generator makes the starter world, now **2048 × 1024** and wrapping, from a seed. The same seed gives the same
world. Everything below is on the torus: distances are the shortest way round.

- **Landmasses**: 3 continents and 6 islands, each a seed cell and a radius, with a noisy coast.
  - Continents are evenly spaced around the ring, one every third of the width. The first is centered on the map's
    center, so the spawn is on it. The other two sit a quarter of the height above and below it, each seed moved by
    up to ±40 cells either way. Their radius is 270 cells: about 11% of the map each.
  - Islands have a radius of 80 cells, about 1% each. Their seeds are drawn at random, each at least 40 cells of
    open sea from every other landmass at their widest, trying up to 200 places for each; an island that finds no
    place is dropped.
  - A cell is land when, for some landmass, its distance to the seed is under the radius × (1 + 0.3 × n), n a
    smooth noise in −1…1 that wraps (octaves of 256, 128, 64 and 32 cells). So coasts are ragged, with bays, capes and
    islets, which all stay whatever their size. A land cell belongs to the landmass whose seed made it land, the
    closest one on a tie.
  - About 40% of the surface is land. The rest is `ocean`, at −1 everywhere.
- **Relief**, on land only: the noise of ADR 0007 (octaves of 64, 32 and 16 cells), plus a mild rise inland: a cell
  d steps from the ocean gets min(d, 120) / 120 × 0.6 added to its noise, which spans about −1…1. The shares per
  level (ADR 0007) are cut over the land cells only, so about half the land is at 0.
- **Banks**: every water (ocean, lake, river) keeps the cells around it near its own level: a cell k steps from
  water at level L is kept between L − (k − 1) and L + k, for k up to 4, for every water level around it. So banks
  are at L or L + 1, the ocean's at −1 or 0 as before, and the ground steps between waters of different levels.
  Then cliffs are smoothed as before: water never moves, nor do bridges and their ends.
- **Lakes**: flat blobs of 6 to 20 `water` cells, grown like holes, at least 30 cells from the ocean and 20 from any
  other water.
  - **Source lakes** feed rivers. Their level is the ground's at their center, at least 0.
  - **Standalone lakes** have no river out: 1 to 3 per continent, none on islands. Their level is one below the
    ground's at their center, −3 at the lowest, so a lake can sit below the sea.
  - A lake can receive a river, and a source lake can receive one too, chaining lakes.
- **Rivers**: 2 to 4 per continent, 0 or 1 per island, each from its own source lake.
  - Its **sink** is the closest of the ocean and the lakes at least one level below the source lake. On an island
    the ocean is the only sink.
  - Its **course** leaves the lake from its cell closest to the sink and heads straight for it, bending sideways by
    a smooth wobble of up to an eighth of its length, zero at both ends. It ends at the first water it meets:
    the sink, or a lake or river on the way, which it joins. A river that meets water higher than itself is tried
    again with another wobble, 5 times, then dropped.
  - Its **level** goes from the lake's down to the sink's, one level at a time, evenly along the course: to −1 at
    the sea, or to the lake's level, which may be below the sea. Two touching river cells are at most 1 apart: a
    small rapid, drawn like any water. A course too short for its drop (under 3 cells per level) is dropped.
  - Its **width** is 1 cell for the first third of the course, 2 for the second, 3 for the last. A short river may
    end at 2.
  - **The spawn's river**: the first river of the spawn's continent is fixed. Its lake is 120 to 160 cells west of
    the spawn, 18 to 26 rows above it, its sink is the ocean east, and its wobble is at most ±5 rows. So at the
    spawn's column, water is 15 to 30 rows above the spawn, as before.
- **Bridges**: 1 cell wide, as today.
  - About one every 60 cells along each river, the first 30 cells in, at a spot where the river runs within 30° of a
    column or a row and both banks can stand a level above the water.
    The deck runs across the river, along the other axis, over its 1 to 3 cells, at the river's level + 1 there;
    the bank cell at each end is set to the deck's level, and that cell and the one past it are kept free of rocks
    and holes.
  - Each river gets at least one bridge when it has such a spot; the spawn's river has one within 30 columns of
    the spawn.
- **Holes**: as before, at least 3 cells from any water, ocean included, and from bridges.
- **Rocks**: as before.
- **Regions**: cut by the closest point, on land within each landmass, and over the ocean:
  - 3 points per continent and 1 per island: a region never spans two landmasses;
  - 3 ocean regions, their points between the continents: every ocean cell takes the closest. They are much
    bigger than land regions.
  - That makes 18 regions. Names are placeholders stored with the world, 6 land names and 3 sea names added to
    ADR 0003's twelve.

### The starter map file

- `ground.png` gets ocean `#4f9fd8`. `README.md` and `palette.gpl` list it.
- The folder is regenerated: `make world FORCE=1`, at the new size.
- **Migration `0004`** deletes every map, so the next boot loads the new file. Rooms keep the map they loaded, so
  the server is restarted.

### Drawing

- **Ocean** is baked into the chunk's ground texture, like grass: no sprite per cell, no animation for now. An
  all-ocean chunk is one texture. It is a deeper blue than the river, flat, with a lighter foam line on each edge
  next to a cell that isn't ocean, so coasts and river mouths show.
- Banks show their earth face toward the ocean in front, as toward a river, and their rim above it.
- River water keeps its own layer and flow (ADR 0007).

### The map screen

- The image gets ocean `#4f9fd8`. Ocean regions aren't shaded: the sea is one color.
- The palette grows by one color.

### Websocket

- `chunk` keeps its shape. `kinds` may now hold `ocean`. The client drops a chunk with an unknown kind, as before.

### Logs

The generator logs, on top of ADR 0007's: the land share, the landmasses with their cells, the lakes, the rivers
with their length and sink, and the bridges.

### How it is built

- Back:
  - `worldmap/domain`:
    - `map.go`: `TileOcean`; `terrain.go`: not walkable;
    - `land.go` (from `generate.go`): the land's arrays, now with each cell's landmass, its steps to the ocean and
      to water, and the bank settling for every water level;
    - `landmasses.go`: seeds, coast noise, the ocean, and the inland rise;
    - `lakes.go`, `rivers.go`: lakes, courses, levels and widths; `bridges.go` (from `river.go`): bridges on a
      course;
    - `regions.go`: regions per landmass and per ocean;
    - `image.go`: the ocean's color.
  - `worldmap/infra/mapfile.go`: the ocean's color.
  - `migrations/up/0004.up.sql`.
- Client:
  - `map/worldMap.ts`: `ocean`, drawn and baked.
  - `art/ocean.ts`: the ocean tile, with foam.
  - `render/chunkView.ts`: ocean cells baked with foam toward non-ocean neighbors.

### Security review

- The chunk decoder accepts code 5 and still refuses anything above it. Rocks on ocean are refused when a map is
  checked. The client drops a chunk with a kind it doesn't know.
- The generator runs only from `mapfile generate`, on fixed sizes. Island placement, river tries and lake growth
  are all bounded, so a seed can't make it loop.
- A 2048 × 1024 world is 2,048 chunks, about 6 MB in the database and in memory, and a 2 MB paletted image while
  rendering: within ADR 0005's bound of 4096 × 4096.

### Not now

- Travel between landmasses: ships, dome transports.
- Beaches: a sand ground kind.
- The ocean's animation, and deep water.
- Waterfalls drawn where a river steps down.

## Test cases

### Back: map rules and walking (`worldmap/domain`)

1. Code 5 decodes as `ocean`; code 6 is refused. A rock on ocean is refused.
2. Ocean isn't walkable; a path never crosses it; clicking it is like clicking water.

### Back: the generator (`worldmap/domain`)

3. Size: the starter world is 2048 × 1024, wrapping, and the same seed gives the same map.
4. Landmasses:
   - land (every non-ocean cell) is 30 to 50% of the cells;
   - exactly 3 land bodies have more than 100,000 cells, and 4 to 6 have 8,000 to 60,000;
   - the spawn is on the biggest-three body that holds the map's center;
   - every ocean cell is at −1.
5. Relief:
   - every height is within −3…3, and any two neighbors are at most 1 apart, across the seams included;
   - 40 to 60% of the land cells are at 0;
   - land cells more than 100 steps from the ocean are higher on average than land cells within 10 steps.
6. Banks: every land cell next to water is between the lowest water level next to it and the highest plus 1: next
   to one water at L, at L or L + 1.
7. Lakes:
   - at least 3 standalone lakes: flat water bodies of 6 to 20 cells (a bridge counts as part of its river);
   - river water is at −1 or above, unless the river ends in a lake below the sea: then the river's lowest cells
     form a flat group of 6 or more, that lake.
8. Rivers:
   - at least 10 water bodies of more than 20 cells (rivers), and one touches the ocean;
   - every river's highest cells form a flat group of 6 or more: its source lake;
   - along any water body, each cell has a neighbor in it at the same or a lower level, until the body's lowest
     cells (water flows down);
   - at the spawn's column, water is 15 to 30 rows above the spawn.
9. Bridges:
   - every bridge is a straight run of 1 to 3 `bridge` cells, 1 cell wide, at the water's level + 1, with a bank
     cell at each end at the deck's level, walkable and rock-free;
   - at least 10 bridges exist;
   - one is within 30 columns of the spawn, and a path from the spawn reaches the cell past its far end.
10. Holes: at least 3 cells from any water, ocean or bridge.
11. Regions:
    - 18 regions, every cell has one, and the ocean is cut among 3;
    - no region holds cells of two landmasses of 8,000 cells or more (islets belong to their landmass's regions), nor
      of land and ocean.
12. The generated world passes the map file's checks.

### Back: the map file, boot and migration

13. A map with ocean writes and reads back the same. An unknown color is still refused.
14. The committed `mapfiles/world` passes every check, at 2048 × 1024.
15. On an empty database, boot saves 2,048 chunks and 18 regions.
16. Migration `0004` deletes every map.

### Back: the map image

17. Ocean has its color, the same for every ocean region. The image stays a paletted PNG.

### Client

18. Protocol: a `chunk` with `ocean` decodes; an unknown kind drops it.
19. Ocean is baked into the ground texture: a chunk that is all ocean has a ground texture and no water sprites.
20. Ocean foam: on the edges next to non-ocean cells only, not next to ocean nor next to a chunk not held.
21. A grass cell at 0 with ocean in front of it shows a face 1 level high toward it; with ocean behind it, none.

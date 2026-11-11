# ADR 0011: Massifs and the glacier ring

- Status: accepted, 2026-11-11
- Replaces, from ADR 0009:
  - "Mountains, in packs on the summits" and "Glaciers, at the inner ridge" in "The generated world";
  - "Mountain" and "Glacier" in "Drawing", and the stone and ice faces' bands;
  - "Mountains that rise higher than level 3" in "Not now".
- Adds to ADR 0007 (heights) and ADR 0008 (the spawn, its river and its bridge).

## Context

ADR 0009's glaciers are scattered blobs, and its mountains flat packs on the level-3 hilltops, each cell with its own
little peak or raised slab: neither reads as one mass. The inner ridge, facing the hole, is the coldest place of the
donut: it should be a pole, a ring of glacier all the way round. Mountains should be big chunks of rock rising above
the hills, their lower blocks leaning up to the rock above them, their tops pointy.

## Decisions

### Heights

- A cell's height goes from −3 to **+6**. Above +3, only a mountain or a glacier: a map with another kind above +3 is
  refused ("(col, row): height 4 is only for mountains and glaciers").
- The walkable terrain doesn't change: the relief, the water, the banks, the bridges and the 1-level step rule stay
  within −3…+3 (`MaxGroundHeight`). Mountains and glaciers stay unwalkable.
- **`height.png`** gets a color ramp instead of grays: blue to red from −3 to +3, then purples above.

  | level | −3        | −2        | −1        | 0         | +1        | +2        | +3        | +4        | +5        | +6        |
  |-------|-----------|-----------|-----------|-----------|-----------|-----------|-----------|-----------|-----------|-----------|
  | color | `#1e3a8a` | `#2563eb` | `#38bdf8` | `#4ade80` | `#facc15` | `#f97316` | `#dc2626` | `#d8b4fe` | `#a855f7` | `#6b21a8` |

  `README.md` and `palette.gpl` list them. Any other color, the old grays included, isn't a height.

### The generated world

Mountains come first, then the glacier ring, then the rocks (on grass only, as before). Regions don't change: the
ring is part of the regions it crosses.

- **Massifs**:
  - **Summits** as in ADR 0009: connected groups (8 neighbors, across the seams) of grass at level 3, of 4 cells or
    more. Each one seeds a massif.
  - A massif grows from its summit, 8 neighbors, over grass at level 1 or above, up to **12 cells** away, with a
    ragged edge: a cell k cells from the summit cell it was reached from, as the crow flies, joins when
    k ≤ 12 × (0.5 + 0.5 × noise), the noise smooth and wrapping. Measured as the crow flies, massifs are round, not
    square. Kept cells (a bridge's ends) never join. Massifs may merge.
  - None within **24 cells** of the spawn.
  - **Height**: the cell's ground height plus its distance to the massif's edge, 8 neighbors: the cells beside a
    non-mountain cell are +1, the next ring +2… Capped at +6.
- **The glacier ring**:
  - The **ridge** is row height / 2 (512). The **core** is the rows within **40** of it; the **edge** reaches **64**:
    a cell r rows from the ridge, 40 < r ≤ 64, freezes when (r − 40) / 24 < noise, so the edge comes in lobes.
  - The ring goes all the way round, over land and sea: grass, ocean, lakes and rivers become glacier. Holes stay:
    they read as crevasses. Bridges become glacier, their kept ends too. Mountains stay: rock standing in the ice.
  - **The pass**: no glacier in the columns within **48** of the spawn's column (across the seam), on the whole
    ring's height. You walk from the spawn to the land on either side through it. The spawn's lake, 120 to 160 cells
    west of it, freezes; its river flows again in the pass, and its bridge, within 30 columns of the spawn, stays.
  - **Height**: the cell's height before freezing (−1 on the sea) plus 1 level every 4 cells in from the ring's
    edge: d being the distance (8 neighbors) to the nearest cell that isn't glacier, a hole or a mountain, the
    height is + 1 + ⌊(d − 1) / 4⌋, capped at +6.
- The spawn rule doesn't change: the walkable cell closest to the center, which is in the pass.

### The starter map file

- Regenerated with `make world FORCE=1`, then **migration `0006`** deletes every map, so the next boot loads the new
  file. The folder is regenerated before the migration is added: ADR 0009's migration ran on a server built with the
  old folder, which kept the old map.

### Drawing

Settled on preview images of a sample massif and of the ring's edge, rendered with the client's own layout and art.

- **Slopes, no vertical walls**: a mountain's or glacier's top is 4 triangles, from its middle to each edge. Its
  back corner `n` and its side corners `w` and `e` rise to the highest cell of its kind touching them behind or
  beside (`n`: the cells at col − 1 and row − 1 and straight above on screen; `w`: col − 1 and the cell to the left;
  `e`: row − 1 and the cell to the right); its front corner stays at its height. So a lower block in front leans up
  to the higher one: the massif and the ice are one continuous, sloped surface. A cell not held, or of another
  kind, raises nothing. Under the top's front edges, down to the cell's height, the block's sides.
- **Peaks**: a mountain with no higher mountain among its 8 neighbors is a pyramid, its point 1.5 levels (12 px) over
  its middle. On a massif's +6 crest, that's a row of points.
- **Light**: each triangle is shaded by how it faces the light, from the left of the screen and above, like the
  faces: in 4 shades per material, the flat top on the second lightest.
- **Rock**: gray-purple, in large facets with cracks between them, laid on the map, not on the cell: the pattern
  repeats every **8 × 8 cells**, so neighbors join and the repeat doesn't show. Half the facets one shade darker. A
  dark outline only on the edges toward a cell that isn't mountain, or toward a lower mountain behind (`nw`, `ne`):
  a lower one in front leans up to it, which joins them. The tops at **+6** are drawn in the glacier's ice:
  cosmetic, still mountains.
- **Ice**: one merged mass, cyan-teal, in plates over the same 8 × 8 repeat. Flat ice is one shade, so the slopes,
  every 4 cells, stand out lighter or darker. Soft cracks only between plates of different shades, a glint on the
  others. A thin bright line on the edges outlined like the rock's.
- **Faces**: stone and ice faces have no band per level, so a tall face reads as one wall: stone with uneven darker
  vertical cracks, ice with lighter vertical streaks. Earth and wood keep theirs.
- **Neighbors across chunks**: a block's corners look at the cells straight above and beside it, which can be in the
  diagonal chunks. A chunk that arrives redraws the 8 held chunks around it, diagonals included.
- **On the heights**: a +6 cell is drawn 48 px above level 0. Picking scans from +6 down to −3. Clicking the upper part
  of a slope may pick the cell behind it: both are unwalkable, so Move is off either way.

### The map screen

- Mountains and glaciers are shaded by level like grass (about 7% per level, ADR 0007), lighter going up, from −3 to
  +6. Glaciers start from the client's flat ice (`#9fe4f1`): the ground layer's `#dff4fb` is too pale to show the
  steps. The image stays a paletted PNG, of 208 colors.

### Websocket

- `chunk` keeps its shape; its heights go from −3 to +6. The client drops a chunk with a height outside.

### Logs

- The generator's log already gives the cells per ground kind and per level, +4 to +6 included. Nothing more.

### How it is built

- Back:
  - `worldmap/domain`:
    - `map.go`: `MaxHeight = 6`, `MaxGroundHeight = 3`, the check of kinds above 3;
    - `relief.go`, `land.go`, `bridges.go`: `MaxGroundHeight` where they meant the terrain's range;
    - `mountains.go`: summits, massifs and their heights; `glaciers.go`: the ring, the pass and its heights;
    - `image.go`: mountains and glaciers shaded per level.
  - `worldmap/infra/maplayers.go`: the height colors.
  - `migrations/up/0006.up.sql`.
- Client:
  - `map/worldMap.ts`: `MAX_HEIGHT = 6`.
  - `render/chunkView.ts`: each block's shape, outline and icy top in the layout; the 8 chunks around redrawn.
  - `art/block.ts`: a block's top as 4 lit triangles, and its sides; `art/facets.ts`: the 8 × 8 pattern;
    `art/mountain.ts`, `art/glacier.ts`: their colors; `art/cliff.ts`: stone and ice faces without bands;
    `art/tileArtSet.ts`, `render/tileGraphics.ts`: their frames and cache keys.
  - `map/grid.ts`: `chunksAround`, the 8 chunks around one.

### Security review

- The map check refuses a height above +6 or below −3, and anything but a mountain or a glacier above +3. The client
  drops a chunk with a height outside −3…+6.
- The generator runs only from `mapfile generate`. Massifs and the ring are flood fills bounded by the map's cells,
  so a seed can't make it loop.
- The terrain's sprites stay bounded: a mountain or glacier sprite is keyed by its place in the 8 × 8 repeat (64),
  its outline edges (16), its icy top (2) and its shape (each corner 0 to 9 levels, a peak or not). Only the
  combinations a map's cells use are painted, each once, at most 32 × 88 pixels.
- Picking scans 9 levels instead of 6, per pointer move and per frame: still cheap.

### Not now

- A named polar region and its banner.
- Ice ground you can walk on, sea ice as its own kind.
- Players and rocks hidden behind tall terrain, and their silhouettes (see `docs/FOR_LATER.md`).
- Climbing mountains.

## Test cases

### Back: map rules (`worldmap/domain`, `worldmap/infra`)

1. Heights: a mountain or a glacier at +4, +5 and +6 passes the checks; grass, water, a hole or a bridge at +4 is
   refused with its message; +7 and −4 are refused for every kind.
2. The height layer: each level from −3 to +6 writes and reads back as its color; a gray like `#808080` is refused
   ("isn't a height"); a map with +6 mountains and glaciers writes and reads back the same. The README lists the 10
   colors.

### Back: the generator (`worldmap/domain`)

3. Massifs, on a small hand-built land (a level-3 summit on a slope of levels 1 and 2):
   - every mountain cell is at most 12 steps from the summit, on what was grass at level 1 or above, and none is
     within 24 cells of the spawn;
   - each height is the ground height plus the distance to the massif's edge (+1 beside a non-mountain cell),
     capped at +6;
   - a kept cell never becomes mountain.
4. The ring, on a small hand-built wrapping land (grass, a strip of ocean, a lake, a river with a bridge, a hole and
   a mountain inside the core):
   - every cell within 40 rows of the ridge, outside the pass, is glacier, a hole or a mountain; none past 64 rows;
   - the ocean, the lake, the river and the bridge inside the core are glacier; the hole and the mountain stay;
   - no glacier in the columns within 48 of the spawn's, across the seam too;
   - each glacier's height is its height before freezing + 1 + ⌊(d − 1) / 4⌋, capped at +6: +1 for the 4 cells in
     from the edge, +2 for the next 4, −1 + 1 = 0 on the sea's edge.
5. The starter world:
   - every cell above +3 is a mountain or a glacier; every other cell stays within −3…+3;
   - rows within 40 of the ridge, outside the pass, hold no grass, water or ocean;
   - between 40 and 64 rows from the ridge, some cells are glacier and some aren't (the edge is ragged);
   - from the spawn, a path reaches a walkable cell more than 64 rows from the ridge, on each side;
   - the spawn's bridge is still reached from the spawn (ADR 0008 test 7).
6. Generated without mountain nor glacier options, the same seed gives the same ground and heights as with them,
   outside the mountain and glacier cells: they only ever replace other cells.
7. The map image: a mountain at +6 is lighter than one at +2, a glacier at +4 lighter than one at 0; the palette
   holds at most 256 colors.

### Client

8. Protocol: a `chunk` with a height of 6 decodes; 7 or −4 drops it.
9. Drawn position: a cell at +6 is drawn 48 px higher than at 0; picking finds the top of a +6 cell.
10. Blocks (`layoutChunk`, pure), for a mountain L at height 1:
    - a mountain straight above at 3: its back corner up 2;
    - one at `nw` at 2: its back and left corners up 1; at `ne`: back and right; to its left at 3: its left corner
      up 2; several: the highest for each corner;
    - a glacier behind, a lower mountain behind, a higher one in front, or a cell not held: nothing rises;
    - a mountain with no higher mountain around is a peak, a glacier never; glaciers rise toward glaciers the same;
    - outlined toward another kind and toward a lower mountain behind only; an icy top at +6, not at +5.
11. Chunks to draw: a chunk that arrives redraws the 8 held chunks around it, each once, across the seams.
12. Rock and ice art:
    - the pattern is the map's: two cells sharing an edge get the same facet along it, it repeats every 8 cells, and
      has no seam where it does;
    - a flat block is its ground diamond only; a raised corner makes its top lean up to it, a raised side corner a
      side under the edge going to it; a peak is a pyramid, lit on its left, its point 12 px over its middle;
    - the outline (rock) and the bright line (ice) are only on the edges given;
    - an icy mountain top is painted in the ice's colors, its sides in stone; ice sides are streaked.
13. The terrain's sprites are cached by pattern place, edges, shape and icy top: the same look twice paints once.
14. By hand: the preview image of a massif and of the ring's edge is approved before the generator changes.

### Back: migrations

15. Migration `0006` deletes every map, like `0003` to `0005`.

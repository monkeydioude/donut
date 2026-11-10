# ADR 0003: A donut-shaped world in chunks, loaded around the player as they walk

- Status: accepted, 2026-11-10
- Replaces, from ADR 0001:
  - the starter map `meadow`, the `ground` column of `maps` and its 256-cell limit;
  - the layers in `GET /api/maps/{slug}`;
  - everyone on a map seeing everyone.
- Replaces, from ADR 0002:
  - the camera centered on the map;
  - the Pixelify Sans font;
  - "Area of interest" in "Not now".

## Context

Maps fit on one screen and come whole over HTTP. The game needs one big world that you walk across with no
loading screen.

The game is set on a donut-shaped planet: an online RPG where you start as a battered robot in a dome city. Fog of
war will come later, and with factions and events it will be strict: the client must only get what its player may
see.

## Decisions

### The world

- **Shape:** it's the surface of the donut. Laid flat, that surface is a rectangle that **wraps both ways**: walk
  off the east edge and you come back in from the west, and the same for north and south. There is no edge.
- **Size:** the starter world is `world`, **1024 × 512 cells**: 1024 around the ring, 512 around the tube. Rows
  near 0 are the outer equator, and rows near 256 face the hole. Every cell has the same size.
- **Wrapping** is a property of the map (`wraps`). The world wraps. Interiors, floors and the dome city's insides
  come later as their own maps with real edges, like maps today.
- **Sizes:** multiples of 32 (the chunk size), from 32 to 4096 cells per side. A wrapping map is at least
  160 cells (5 chunks) per side, so a chunk never shows up twice in the zone a player holds.
- **What each cell holds:**
  - a ground kind:
    - `grass`;
    - `void`: not drawn and not walkable, for a place with no ground, such as a chunk that disappeared;
  - a height: a signed byte, 0 everywhere for now. Drawing and walking on heights get their own ADR.
  - a region id: 1 to 255, or 0 for no region.
- **Objects** stay as they are: rocks, at most one per cell. A cell is walkable when its ground isn't `void` and
  nothing stands on it.
- **Regions:** each has a name and a biome. There is only one biome for now, `meadow`.
- **The starter world**, generated from `STARTER_MAP_SEED`:
  - grass everywhere, height 0, and rocks on about 12% of the cells;
  - 12 regions: 12 points scattered at random, and each cell belongs to the closest one. Distances wrap, so regions
    run across the seams. Their names are placeholders, stored in the database to rename:
    - Sprinkle Meadow, Glaze Hills, Cinnamon Vale, Maple Hollow, Honey Downs, Sugar Fields;
    - Cocoa Glen, Frosting Flats, Crumb Valley, Jelly Ridge, Cruller Heath, Powder Plains.
- **Spawn:** the walkable cell closest to the world's center, as before. A returning guest comes back where they
  left. The dome city comes later.

### Wrapping rules

- Cells are always told within the map: 0 ≤ col < width, 0 ≤ row < height. Cells outside get an empty path, as
  before.
- On a wrapping map, these all go across the seams:
  - the 8 neighbors of a cell;
  - the distance between two cells, and so the A* estimate;
  - paths;
  - the areas the terrain numbers;
  - which chunks are near a player.

  From (1023, 5), the cell east is (0, 5), and a walk to (2, 5) takes 3 steps.

### Chunks

- A chunk is 32 × 32 cells. Chunk (cx, cy) holds the cells from (32·cx, 32·cy) to (32·cx + 31, 32·cy + 31).
- **What a player holds:**
  - **Receiving:** you get the chunks within 1 chunk of yours (3 × 3), as they come into range.
  - **Forgetting:** you forget a chunk only once it's more than 2 chunks away (outside the 5 × 5 around yours).
    That gap stops chunks from bouncing in and out when you walk back and forth across a border.
  - The server keeps the list of what each connection holds, and never sends a chunk twice. Fog of war will plug in
    there later.
  - Holding at least 1 chunk around yours leaves at least 32 cells past your chunk in every direction. A 1080p
    screen shows about 21 cells from its center, so a new chunk always arrives before it can show.
- **Who you see:** the players standing on chunks you hold.
  - `welcome` lists only those.
  - `joined` tells that a player came into view: they arrived on the map, or onto a chunk you hold, or a chunk they
    stand on became yours.
  - `left` tells the reverse.
  - `moved` comes only for players you see, and you always see yourself.

### Storage

- `maps` loses `ground` and gets `wraps`. Its size checks become multiples of 32, from 32 to 4096.
- `map_chunks (map_id, cx, cy, version, data, updated_at)`:
  - The key is `(map_id, cx, cy)`, and a chunk is deleted with its map.
  - `data` is binary (bytea), 3,073 bytes in format 1:
    - 1 byte: the format, 1;
    - 1,024 bytes: the ground codes, row by row (0 `void`, 1 `grass`);
    - 1,024 signed bytes: the heights;
    - 1,024 bytes: the region ids.
  - `version` starts at 1 and goes up each time the chunk changes. Later, it will tell holders the chunk changed.
  - A chunk with no row is all `void`: a chunk that disappears is a deleted row.
- `map_regions (map_id, id, name, biome)`, keyed by `(map_id, id)`, with ids from 1 to 255.
- Rocks stay one row each in `map_objects`. They will change one at a time.
- **Migration `0002`:**
  - deletes every map: the dev `meadow`, its rocks with it;
  - drops `ground`, adds `wraps` and the new size checks;
  - creates `map_chunks` and `map_regions`.
- At boot with no map, the server generates the world and saves it in one transaction: its chunks, its regions
  and its rocks.

### Server

- **The world in memory:** when a room opens, it loads its whole map: chunks, regions and rocks, about 1.5 MB at
  1024 × 512. It builds the terrain once from that.

  Presence reads the world through an interface (walkable, path, spawn, the grid, a chunk's contents), so loading
  chunks on demand can replace it later.
- **Pathfinding** wraps. It stays under the room's lock, with the 8,192-cell search limit. One room now holds every
  player in the world, so moving searches out of the lock waits until there are many players.
- **A `plan` or `walk` to a cell in a chunk you don't hold gets an empty path**, so a `walk` also stops the current
  walk. You can only click what you see, and fog of war stays watertight later. The path itself may run outside
  your chunks.
- **On joining:** you get `welcome`, then your chunks, yours first. The players who hold your chunk get `joined`.
- **On a step into another chunk**, the walker gets, in this order:
  1. `moved`;
  2. a `chunk` for each chunk that came into range (3 walking straight, 5 diagonally);
  3. `joined` for the players on them;
  4. `left` for the players on the chunks it forgets;
  5. a `forget` for each of those chunks.
- **On any step**, the other players get `moved`, `joined`, `left` or nothing. It depends on whether they hold the
  chunk of the cell the walker left, the chunk of the cell it stepped on, or both.
- **The same guest connecting again** gets `welcome` and all its chunks again: what a connection holds starts
  empty.
- **A connection's queue** goes from 64 to 256 messages. Walking into a crowd brings a burst of `joined`, and a
  normal player mustn't be dropped as too slow for it.
- **Compression:** the websocket negotiates per-message compression (permessage-deflate, without context
  takeover), for messages over 512 bytes. A chunk is about 16 KB of JSON before compression, most of it the rocks'
  ids, and about 3.2 KB on the wire.
- **Logs:** debug logs for every new action:
  - the world loaded and the terrain built, with timings;
  - the starter world generated;
  - chunks sent and forgotten for each player;
  - players coming into and out of each other's view.

  Chunk messages are logged by their key and size, never in full.

### Websocket messages

- Server to client, new:
  - `chunk`:

    ```json
    {"type": "chunk", "cx": 0, "cy": 0, "version": 1,
     "kinds": ["grass", "void"], "ground": [0, 0, 1, "…1,024 indices into kinds, row by row"],
     "height": ["…1,024"], "region": ["…1,024"],
     "objects": [{"id": "…", "kind": "rock", "col": 3, "row": 7}]}
    ```

    The objects have their map cells, inside the chunk.
  - `forget {cx, cy}`.
- `welcome`, `joined`, `left` and `moved` keep their shape, with the meaning above.
- Client to server: unchanged (`plan`, `walk`).

### HTTP

- `GET /api/maps`: `[{id, slug, name, width, height, wraps}]`.
- `GET /api/maps/{slug}`: the map's description, without layers:

  ```json
  {"id": "…", "slug": "world", "name": "World", "width": 1024, "height": 512, "wraps": true,
   "chunkSize": 32, "regions": [{"id": 1, "name": "Sprinkle Meadow", "biome": "meadow"}]}
  ```

  An unknown slug gets a 404.

### Client

- It loads the description of `world`, then opens the websocket. The background shows until the first chunks
  arrive, a fraction of a second. "No loading" applies to walking, not to joining.
- **Drawing across the seams:** the client draws in coordinates that don't wrap, around your player.
  - Your player's drawn cell keeps counting past the seam: 1023, 1024, 1025…
  - Everything else is drawn at its copy nearest to it: chunks, rocks, players, borders and the path line.
  - Another player's step across the seam glides one cell, from its drawn copy.
  - Each point of the path line is the copy nearest to the point before it.
  - The game logic (selection, messages) keeps the server's cells, always within the map.
  - On a map that doesn't wrap, every copy is the cell itself.
- **Chunks:**
  - Each chunk's ground is drawn once into a texture, at art resolution, so there are 9 to 25 textures instead of
    up to 25,000 sprites. `void` cells are left transparent.
  - Rocks and players stay in one shared depth-sorted container, so a rock on a chunk border still sorts right
    against a player.
  - `forget` destroys the chunk's texture and its rocks.
- **Camera:** every frame, the world is placed so your player's drawn position is at the center of the screen, on
  whole pixels. There's no clamping and no panning.
- **Picking:** the pointer gives a drawn cell, turned back into its cell within the map. A cell in a chunk you
  don't hold shows no hover, and a click on it cancels, like a click outside the map.
- **Region banner**, Dark Souls style but cute:

  ```
                ·  ✦  ·
           Sprinkle Meadow
        ───────── ♥ ─────────
  ```

  - **Where:** centered on the screen, a bit above the middle. It's DOM over the canvas, with no window behind it.
  - **Look:** the region's name in big cream letters with a soft dark outline, and a thin line with a small heart
    under it.
  - **Timing:** fades in over 0.6 s, stays 2.5 s, and fades out over 1 s.
  - **When:**
    - at spawn, once the region of your cell is known;
    - whenever your player's cell enters another region.

    It isn't shown again for the same region within 30 s, so walking along a border doesn't flood the screen. A
    new region replaces a banner still showing. Region 0 shows nothing.
- **Font:** every text in the game is **Zen Maru Gothic** (OFL), Kabane's font, bundled through
  `@fontsource/zen-maru-gothic`. The banner uses weight 700 and the menu 500. It replaces Pixelify Sans.
- **Logs:** debug logs for:
  - the description loaded;
  - chunks received, drawn (with timings) and forgotten;
  - the drawn cell crossing a seam;
  - the banner shown or skipped.

### How it is built

- Back:
  - `core/domain/grid.go`: the grid's geometry: its size and whether it wraps, keeping a cell within the map,
    neighbors, shortest offsets, the chunk of a cell, and the distance between chunks.
  - `worldmap/domain`:
    - `world.go`: the map with its chunks and regions;
    - `chunk.go`: a chunk's cells;
    - `generate.go`: the donut world;
    - `terrain.go` and `path.go`: wrapping, `void`, and flat arrays instead of maps.
  - `worldmap/infra`:
    - `chunkcodec.go`: the binary format;
    - the repo reads and writes chunks and regions.
  - `presence`:
    - `app/room.go`: what each member holds, the zone, and who sees whom;
    - `domain`: `HeldChunks` (what a connection holds, receiving and forgetting), and the `ChunkShown` and
      `ChunkForgotten` events;
    - `transport/ws`: the new messages, compression, and the 256-message queue.
  - `migrations/up/0002.up.sql`.
- Client:
  - `map/grid.ts`: wrapping, nearest copies, the chunk of a cell.
  - `map/chunks.ts`: the chunks held, and the ground and region of a cell. It replaces `worldMap.ts`'s whole-map
    ground.
  - `net/mapApi.ts`: the description.
  - `net/protocol.ts`: `chunk` and `forget`.
  - `render/chunkView.ts`: a chunk's ground texture and its rocks. It replaces `mapView.ts`'s whole-map ground.
  - `render/camera.ts`: following your player.
  - `game/banner.ts`: when to show the banner, as pure logic.
  - `ui/banner.ts`: the banner itself.

### Security review

- Checked:
  - The 1 KB limit on what a client sends counts the bytes after decompression. A compressed message that inflates
    past it closes the connection; a test proves it.
  - Compression leaks nothing. No message carries a secret (guest ids never leave the server), and each message is
    compressed on its own, without context takeover.
  - Only the server decides which chunks a client gets. A client can't ask for one.
  - A `plan` or `walk` to a cell outside the chunks you hold, or outside the map, gets an empty path.
  - The client checks every chunk message (lengths, kinds, indices, ranges, objects inside the chunk) and drops a
    bad one.
  - Region names are shown as text, never as HTML.
  - Chunk messages are logged by their key and size, and guest ids never.
- Known risks:
  - Each join sends 9 chunks: about 29 KB on the wire and 150 KB of JSON to encode. Reconnecting in a loop costs
    the server more than the client. A limit per IP at the reverse proxy (ADR 0001) covers it until authentication.
  - A path may run outside the chunks you hold, so `planned` and `walking` tell cells you can't see. Fog of war
    will have to cut paths to what the player may see.
  - Each step goes through every player in the room. Players should be indexed by chunk once rooms get crowded.

### Not now

- Drawing and walking on heights.
- Fog of war.
- Live chunk changes: a chunk changing or vanishing while players hold it.
- The dome city, and interiors.
- Hand-made regions: importing Tiled files into chunk rows.
- More biomes and ground kinds.
- Moving pathfinding out of the room's lock, and indexing players by chunk.
- Loading the world's chunks on demand on the server.
- Caching chunks on the client across reloads.

## Test cases

### Back: grid (`core/domain`)

1. Keeping a cell within the map:
   - on a wrapping 64 × 64 grid, (−1, 0) gives (63, 0), and (64, 65) gives (0, 1);
   - on a bounded grid, a cell outside isn't inside, and isn't moved.
2. Across the seam:
   - from (63, 0), the shortest offset to (0, 0) is one column east, and they are neighbors;
   - on a 1024-wide world, chunks 0 and 31 of a row are 1 apart;
   - none of this wraps on a bounded grid.
3. The chunk of a cell: (31, 31) is in chunk (0, 0), and (32, 0) in chunk (1, 0).

### Back: the world (`worldmap/domain`, `worldmap/infra`)

4. A map is refused when:
   - its size isn't a multiple of 32 within 32 to 4096;
   - it wraps and is under 160 cells on a side;
   - a chunk is outside it;
   - a ground code is unknown;
   - a cell's region isn't one of its regions;
   - an object is outside it, on `void`, or shares a cell with another.
5. The chunk format:
   - a chunk encodes and decodes back to the same cells;
   - a wrong length, an unknown format or an unknown ground code gives an error.
6. The generator:
   - the same seed gives the same world: 1024 × 512, wrapping, all grass, height 0;
   - every cell is in one of the 12 regions, which is the closest point, distances wrapping;
   - every region has cells;
   - a rock chance of 0 gives no rocks, and 1 gives a rock on every cell.
7. The terrain on a wrapping map:
   - the cells across the seam are walkable neighbors;
   - `void` isn't walkable;
   - (63, 5) to (1, 5) on a 64-wide map steps through (0, 5): 2 steps, not 62;
   - no corner cutting across the seam;
   - a wall down the middle doesn't cut the map in two, since both halves meet across the seam.
8. On random maps, wrapping or not: every step is a walkable neighbor, no corner is cut, and the path's cost equals
   a brute-force shortest cost.
9. Spawn: the cell closest to the center, or the closest free one.

### Back: storage (`worldmap/app`, Postgres)

10. Migration `0002` on the old schema: `meadow` and its rocks are gone, and so is `ground`.
11. Without maps, the world is created: 512 chunks, 12 regions, its rocks. A second boot creates nothing.
12. The world reads back as saved: chunks, regions, rocks. A chunk with no row reads as all `void`.

### Back: HTTP

13. `GET /api/maps/world` gives the description: size, `wraps`, `chunkSize` and the regions, without layers. An
    unknown slug gives 404. The list shows `wraps`.

### Back: players on the world (`presence/app`, fake world and clock)

14. Join:
    - the newcomer gets `welcome` with only the players on its 3 × 3 chunks;
    - then its 9 chunks, its own first;
    - the players who hold its chunk get `joined`, and the others nothing.
15. Stepping into the next chunk:
    - straight, the walker gets the 3 new chunks; diagonally, 5;
    - it never gets a chunk it holds;
    - it forgets only the chunks more than 2 away;
    - back and forth across a border, nothing more is sent.
16. Another player's step:
    - an observer holding both chunks gets `moved`;
    - holding only the new one, `joined`;
    - holding only the old one, `left`;
    - holding neither, nothing.
17. When the walker's chunks change, it gets `joined` for the players on the new ones, and `left` for those on the
    ones it forgets.
18. On a wrapping world, a player in chunk column 0 holds chunk column 31, and sees the players there.
19. A `plan` or `walk` to a cell in a chunk you don't hold gets an empty path. The `walk` stops the current walk.
20. Leaving: only the players who held its chunk get `left`.
21. The same guest connecting again gets `welcome` and all its chunks again.

### Back: websocket

22. A client gets `welcome`, then 9 `chunk` messages in the documented shape.
23. A far player (more than 2 chunks away) doesn't see another one walk. A close one does.

### Client

24. Grid:
    - keeping a cell within the map;
    - the copy of a cell nearest to an anchor, wrapping or not;
    - the chunk of a cell.
25. Protocol:
    - `chunk` and `forget` decode;
    - these give nothing: wrong lengths, an index outside `kinds`, an unknown kind, a region over 255, an object
      outside its chunk.
26. The chunks held:
    - a chunk adds its cells, and a forget removes them;
    - a cell's ground and region come from its chunk;
    - a cell in a chunk not held is unknown.
27. The description: size, `wraps`, `chunkSize` and regions parse. An invalid one throws.
28. Drawn positions:
    - chunks, rocks and players are drawn at their copy nearest to your drawn cell;
    - your drawn cell goes 1023 to 1024 across the seam, and the glide is one cell;
    - the path line's points never jump across the world.
29. Picking: over a chunk you hold, the cell within the map; over a chunk you don't, nothing.
30. Camera: your drawn position lands at the center of the screen, on whole pixels.
31. The ground of a chunk is laid out from the chunk's corner, back to front, and `void` cells are skipped.
32. When the banner shows:
    - at spawn, once your cell's region is known;
    - on entering another region;
    - not for the same region within 30 s, and again after;
    - region 0 shows nothing.

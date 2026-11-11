# ADR 0012: Doma-1, the first dome city, and travelling between maps

- Status: accepted, 2026-11-11 (Kabane donut #6; sub-tasks #9, #10, #11)
- Replaces, from ADR 0001: the websocket at `GET /api/maps/{slug}/ws`. It's `GET /api/ws` now.
- Replaces, from ADR 0003:
  - "Sizes: multiples of 32", for maps that don't wrap;
  - "Spawn: the walkable cell closest to the world's center": new guests wake up in Doma-1;
  - where a returning guest comes back, kept per room;
  - the client loading `world`'s description before opening the websocket;
  - "a click on a chunk not held cancels, like a click outside the map": in a city, a click outside opens Exit;
  - "The dome city, and interiors" in "Not now".
- Replaces, from ADR 0005: the map "shown at ×2, ×1 or ×½", and the donut "only on a map that wraps".
- Replaces, from ADR 0007: "At boot with no map in the database, it loads the embedded `world`".
- Replaces, from ADR 0010: "a map that isn't the room's: unknown map", and "Teleporting to another map" in "Not now".

## Context

The game starts, in its story, inside a dome city: a consciousness uploaded into a battered robot wakes up under a
dome that shelters it from the radiation outside. There is no city yet, and the game knows a single map, `world`.

Kabane #6 asks for the first one. It also sets a hard rule for the scale of every map to come: each cell of the world
map holds a 16 × 16 map of its own. Fights later (T-RPG, FFTA style) will happen on the 16 × 16 map of the cell they
start on, made from that cell's terrain.

The card asked for a 48 × 48 city on 3 × 3 world cells. We made it 32 × 32 on 2 × 2: exactly one chunk.

## Decisions

### The scale of maps

- **The world map is an overworld**: one of its cells stands for 16 × 16 cells of a local map. A city on n × n world
  cells is 16·n × 16·n cells.
- Robots are drawn the same size on both: one step on the world crosses what would be 16 cells of a local map.
- Each world cell's own 16 × 16 map, made from its terrain, comes with the fights (`docs/FOR_LATER.md`). Only cities
  exist as local maps for now.

### Maps that don't wrap

- Their sides are any **multiple of 16**, from 16 to 4096. Wrapping maps keep multiples of 32, and their 160-cell
  minimum: their seam must fall on a chunk border.
- The chunks on the far edges are partly outside the map. They are stored and sent whole, 32 × 32, and the cells
  outside the map are `void`. A map with a cell outside it that isn't `void` is refused.

### Doma-1 on the world map

- **A dome** is a new object kind, `dome`, with a **2 × 2 footprint**. It stands on its back cell (the lowest column
  and row) and covers that cell and the 3 cells toward the front: (col, row) to (col + 1, row + 1).
  - Its 4 cells are not walkable.
  - Its footprint stays within one chunk: a dome is sent with that chunk, like a rock.
  - It is sorted against players by its front cell, (col + 1, row + 1): a player behind it is hidden by it.
  - Its **middle**, for the directions below, is the corner its 4 cells share: (col + 1, row + 1) as a point.
  - It **leads to** a map: Doma-1's leads to `doma-1`.
- **Declared in the world's `map.json`**, not painted in `objects.png`:

  ```json
  "domes": [{"col": 1023, "row": 511, "map": "doma-1"}]
  ```

  Moving a dome means editing its two numbers. Loading refuses a dome:
  - whose footprint isn't 4 walkable grass cells at one height, with no rock;
  - whose footprint crosses a chunk border;
  - that leads to a map with no folder in `mapfiles/`, or to a map another dome already leads to.
- **The generator** puts Doma-1 on the spawn, the walkable cell closest to the center, in the glacier pass (ADR
  0011). The spawn cell is its back cell, moved back by one column or row when the footprint would cross a chunk
  border.
  - The footprint and the ring of 12 cells around it become grass at the spawn's height, with no rocks or holes. They
    are kept cells, like a bridge's ends: the smoothing of cliffs moves the ground around them, never them.
  - The committed `mapfiles/world` is exactly the generator's output for seed 42 (checked), so it is generated again
    with `make world FORCE=1`, and no hand edits are lost.
- **The map image** (ADR 0005) paints a dome's 4 cells gold.

### Doma-1, the city

- **The map** `doma-1`, named "Doma-1": 32 × 32 cells, not wrapping. Its only region is "Doma-1", of a new biome,
  `dome`.
- **Two new ground kinds**:

  | Code | Kind    | Walkable | Notes                                   |
  |------|---------|----------|-----------------------------------------|
  | 8    | `floor` | yes      | the city's paving                       |
  | 9    | `wall`  | no       | the dome's rim, at the city's edge      |

  Rocks still stand only on grass.
- **The layout**, measured from the map's middle, the point (16, 16), to each cell's middle:
  - up to 14.5: `floor`, except a **park** of grass on the 6 × 6 cells in the middle (13 to 18 both ways);
  - from 14.5 to 15.5: `wall`, a ring 1 cell thick;
  - past it: `void`: the corners. The city floats on the background, like the edge of what you hold on the world.
- **Heights**: the floor and the park at 0. The wall is a **cutaway**:
  - the back half of the ring, toward the top of the screen (col + row < 30), at +3;
  - the front half (col + row > 32) at +1, a low rim that hides nothing;
  - +2 where they meet, at the sides.
- **The spawn** is the walkable cell closest to the middle, by the rule of every map: the park's middle.
- **Made by** `make city`, which runs `go run ./cmd/mapfile city -out mapfiles/doma-1`. Like `make world`, it
  refuses to overwrite a folder with files unless `FORCE=1`. Then the folder is edited by hand. It has the world's
  format: `map.json` and the four PNGs, with the new colors in `README.md` and `palette.gpl`.

### Travelling between maps

- **One websocket for the game**: `GET /api/ws?player=…&name=…&robot=…`, with no map in its address. The server
  knows which map each guest is on, and moves them between rooms on the same socket.
- **Where you are**, in memory: the hub keeps the map each guest is on, or left from; that map's room keeps the cell
  where they left it, as rooms did. A server restart forgets both.
- **On connecting:**
  - a new guest wakes up at **Doma-1's spawn**;
  - a returning guest comes back where they left, on that map;
  - a remembered cell that isn't walkable anymore, or a map that's gone, sends you to Doma-1's spawn too.

  Nobody joins the world directly anymore: its spawn rule only places the dome in the generator.
- **A move between maps** leaves the old room, then joins the new one, never holding both rooms' locks. For a moment
  you are in neither, never in both.
  - Where you go, map and cell, is worked out when you ask, the target's room opened first: maps don't change while
    the server runs, so a walk through a door ends with "go to this cell of this map", like a teleport.
  - A room hands the traveler to the hub once its own lock is released, and the hub puts them in the other room.
  - A connection that ends on the way leaves you remembered there.
  - The old room tells the players who saw you `left`, as when someone disconnects. You get nothing from it.
  - The new room places you on your arrival cell and tells you, in this order: `welcome`, which now carries the map's
    description, then your chunks. The players who hold your chunk get `joined`.
- **Your request limit** (10 per second, ADR 0002) is your connection's now, not a room's: it follows you across maps,
  so bouncing between rooms doesn't reset it, and it's counted before any map is looked up.
- The same guest connecting again, while already in a room, joins that room and replaces the old connection, as
  before.

### Enter and Exit

- **Enter**, on the world map: a click on any of a dome's 4 cells opens the menu with **Enter** where Move would be.
  - You walk to the **closest walkable cell touching the footprint** (8 neighbors, 12 cells): the closest by straight
    distance that a path reaches, so a request runs one path search, rarely more. On arrival, you go in.
  - Standing on one of those cells already, you go in at once.
  - With none reachable, Enter is off, like Move on a rock.
  - **Inside, you appear** on the city's **edge** (its walkable cells touching the wall, 8 neighbors), on the side you
    came from. It's the edge cell whose direction from the city's middle is closest to your cell's direction from
    the dome's middle. Coming from the dome's east, you appear on the city's east edge.
- **Exit**, in a city: a click on `void`, a `wall`, or past the map's edge opens the menu with **Exit** where Move
  would be.
  - You walk to the **edge cell closest to the clicked cell**, measured straight. A cell clicked past the map is
    first brought back into the map's box. On arrival, you go out.
  - Standing on that edge cell already, you go out at once.
  - **On the world, you appear** beside the dome, on the side you left by. It's the walkable cell touching its
    footprint whose direction from the dome's middle is closest to your edge cell's direction from the city's middle.
    When that cell isn't walkable, it's the next closest by direction that is.
- **A walk toward a door** is like any walk: another Move, Enter, Exit or a teleport replaces it, and you don't go
  through.
- **Refused**, with a `refused` message and nothing else:
  - Enter on a cell that isn't a dome's: "There's no dome at (5, 7)";
  - Exit on a map that isn't a city: "You're not in a city";
  - Exit when no world dome leads to the city, as when the world in the database is older than this ADR: "Doma-1
    stands nowhere on the world";
  - Exit when no cell beside the dome is walkable: "There's no way out of Doma-1";
  - no reachable cell: an empty path, as for any walk.
- On the world map, a click outside the map can't happen: it wraps. In a city, the right click still cancels.

### `/teleport` across maps

- `/teleport doma-1 16 16` works from the world, and `/teleport world 1023 511` from the city: a move between maps,
  landing on that cell, walkable or not.
- Refused, as before, with "unknown map: mars" for a map that doesn't exist, and "(5000, 2) is outside the map" for
  a cell outside the map named. Both are checked before you leave your room.
- Within your map, nothing changes. Across maps, you get the new room's `welcome` and then `teleported`, so the log
  says where you landed.

### The client switching maps

- **What lives for the whole game**: the canvas, the socket, your guest, the robots, the controls and the camera's
  lock, the chatbox with its history and the log, the region banner, the coordinates, and each map's exploration.
- **What each map builds**: its grid, its chunks and their textures, the players, the borders and path line, the
  selection, and the map screen with its image.
- **A `welcome` for another map than the current one**:
  1. the screen fades to black over 0.3 s, and the messages that come meanwhile wait, in order;
  2. the old map's views are destroyed, the new map's are built from the description in `welcome`;
  3. `welcome` and the waiting messages are applied, and the camera goes back onto your player: a free camera stays
     free and at its zoom, and follows you until it's moved, as when the game starts;
  4. the screen fades back in over 0.3 s, then the region banner shows.

  The first `welcome`, on load, fades in only.
- **The menu's action** comes from the clicked cell, the same rule as the server's: a dome's cell is Enter; in a
  city, `void`, a `wall` or past the map is Exit; anything else is Move. It's on once the server's `planned` gives a
  path for that same action, and the path line shows it, as for Move.
- **Picking the dome**: the pointer over the dome's drawn pixels (its opaque ones) picks the dome, before the ground.
  Its upper part rises over the cells behind it, which would otherwise be picked.
- **Exploration** is kept per map for the session: going back to the world finds its explored chunks.

### Drawing

Like ADR 0011, a preview image of each, rendered with the client's own art, is approved before it's wired in.

- **The dome**, 2 cells wide (64 art px): an old honeycomb dome of hexagonal glass cells on a gray metal ring, with
  the colored dark outline and the ground shadow of objects.
  - The glass is faded teal, light from the top left. The frame is rust-brown and gray.
  - Some cells are **missing**: they show the dark inside, with a faint hint of roofs. Some are **cracked**.
  - Rust streaks run down the frame. Moss and grime gather at its base, on the ring.
- **The floor**: worn pale paving stones. Some cells show cracks, stains or a missing slab (a sunken, darker square).
  They are picked from the cell's place, so the look is the same for everyone. Drawing only: they're all `floor`.
- **The wall**: its top is the metal ring. Its faces toward the inside, under the +3 back half, are old hexagonal
  glass panels, some missing (dark) and some cracked. The front rim's faces go down to −3 toward `void`, like the
  edge of what you hold (ADR 0007), in the same panels.
- **The park**: the world's grass.

### The map screen

- **In a city**, it shows the city's own image. The scale is the biggest of ×8, ×4, ×2, ×1 and ×½ that fits: Doma-1
  shows at ×8, 256 px.
- **The donut always shows**, top right. In a city, your mark on it is where the city stands on the world: its
  dome's cell. A city's description gives that place, `standsOn`, with the world's size, which the mark needs.
- The region line under the map's name is left out when it's the map's name: "Doma-1" shows once.
- Exploration and its share are the map's: Doma-1 is one chunk, so it's 100% explored at once.

### The game log

- Entering a city: "You enter Doma-1". Leaving it: "You leave Doma-1".
- The others see "Robo entered the area" and "Robo left the area", as today: from their room, you came or went.

### Storage and commands

- **Migration `0007`** adds `map_objects.leads_to`, a nullable text column: the slug of the map a dome leads to. Only
  domes have one (a check). It deletes nothing. **Migration `0008`** indexes it, for the domes only: a city's
  description looks for the dome leading to it among the world's rocks.
- **Boot** saves each embedded map that the database doesn't have yet: `doma-1` appears on its own at the next
  boot.
- **`make reset-maps`** (`go run ./cmd/donut reset-maps`), with the server stopped, replaces every embedded map in
  the database with its file, each in its own transaction. `make reset-world` stays.
- Until the world is reset, it has no dome: new guests still wake up in Doma-1, and Exit is refused (see above).
- `make city` makes Doma-1's folder (see above).

### Websocket messages

- Client to server, new:
  - `{"type": "enter", "col": 1023, "row": 511}`: a cell of a dome;
  - `{"type": "exit", "col": -3, "row": 40}`: the clicked cell, which may lie past the map.

  `plan`, `walk` and `teleport` keep their shapes.
- Server to client:
  - `welcome` gains `map`, the map's description, as `GET /api/maps/{slug}` gives it;
  - a map's description gains `standsOn`: `{"map": "world", "col": 1023, "row": 511, "width": 2048,
    "height": 1024}` for a city, `null` otherwise;
  - `planned` gains `action`: `"move"`, `"enter"` or `"exit"`. A `plan` on a dome's cell answers with the path to
    its side, and in a city a `plan` on an exit cell with the path to the edge. For a door you'd go through at once,
    the path is the cell you stand on, so the menu's action is on;
  - a `chunk`'s objects may be `{"id": "…", "kind": "dome", "col": 1023, "row": 511, "leadsTo": "doma-1"}`. The
    client knows a dome's footprint from its kind;
  - `refused` with `"request": "enter"` or `"exit"`, and the reasons above.
- The client drops a `welcome` without a valid map, a `planned` with an unknown action, and a dome without
  `leadsTo` or whose footprint leaves its chunk.

### Logs

Debug logs for:

- the server: a guest placed on connecting (new, returning, or sent back to the spawn, and why), a move between maps
  (from, to, cell, why: enter, exit or teleport), Enter and Exit asked, refused with why, or walked;
- loading a map file: its domes; `reset-maps`: each map replaced;
- the client: the menu's action for a cell, a map switch with how long the rebuild took, and the messages that
  waited during the fade.

### How it is built

- Back:
  - `core/domain`: `Dome` (a footprint, the cells around it), `MapInfo`, `Place`.
  - `worldmap/domain`:
    - `map.go`: `floor`, `wall`, the `dome` biome, sides of 16 for maps that don't wrap, cells past the edge `void`;
    - `objects.go`: the object kinds, a dome's footprint, `LeadsTo`, and their checks; `terrain.go`: footprints aren't
      walkable;
    - `world.go`: what a room reads: `Info`, `DomeAt`, `DomeLeadingTo`, a city's `Edge`, `IsOutside`;
    - `city.go`: Doma-1's layout; `dome.go`: the world generator placing the dome;
    - `image.go`: gold domes, `floor` and `wall`.
  - `worldmap/infra`: `mapfile.go` reads and writes `domes`, `CheckDomes` checks where they lead; `maplayers.go`,
    `mapreadme.go`: the new colors; the repo reads and writes `leads_to`, and the description's `standsOn`.
  - `worldmap/app`: `EnsureMaps` saves each embedded map missing; `ResetMaps`.
  - `mapfiles/doma-1/`; `cmd/mapfile city`; `cmd/donut reset-maps`; the Makefile's `city` and `reset-maps`.
  - `presence/domain/doors.go`: the travel rules, pure: arriving by direction, the closest edge, the refusals.
  - `presence/app`:
    - `hub.go`: the map each guest is on; `Connect` places them: `Return` to their map's room, else `Join` the home
      map (`doma-1`, set in `cmd/donut/routing.go`) at its spawn; and `arrive`: a traveler into their room;
    - `session.go`: a connection's requests, its limit, and the room it's in, set while holding its lock so a traveler
      waits for it; Enter, Exit and cross-map teleports work out where they lead;
    - `room.go`, `room_doors.go`: `Join` at the spawn, `Return` where they left (false when that cell isn't walkable
      anymore), `Admit`, `Enter`, `Exit`, `TeleportAway`, `Refuse`, plans with their action, and travelers handed out
      after the lock.
  - `presence/transport/ws`: `/api/ws`, the `enter` and `exit` requests, `welcome` with the map, `planned` with its
    action.
  - `migrations/up/0007.up.sql`, `0008.up.sql`.
- Client:
  - `map/worldMap.ts`: `floor`, `wall`, `dome`, footprints, `standsOn`; `map/chunks.ts`: `domeAt`; `net/mapApi.ts`:
    the size rule and `standsOn`.
  - `net/protocol.ts`, `net/gameSocket.ts`: the new requests and fields; `socketUrl` without a map.
  - `game/actions.ts`: a cell's action (Move, Enter, Exit), pure.
  - `game/travel.ts`: the map switch: when a `welcome` is for another map, and the messages that wait, pure.
  - `game/session.ts`: a map's session, given the socket and the chatbox; `game/command.ts`: the action and its path;
    `game/chatLine.ts`: typed lines, for the whole game.
  - `game/exploration.ts`: explorations per map; `game/gameLog.ts`: entering and leaving lines.
  - `main.ts`: what lives for the game and what each map builds and destroys; `ui/fade.ts`: the fade; `ui/banner.ts`:
    the banner held while the screen is dark.
  - `art/dome.ts`, `art/city.ts` (floor, wall top, glass faces): the art; `render/chunkView.ts`: the dome as an
    object, sorted by its front cell and picked by its pixels, floors and walls, clearing a map's chunks.
  - `game/mapMarks.ts`, `ui/mapScreen.ts`: the scales up to ×8, the donut in a city.

### Security review

Done by hand on the diff (the `/security-review` command needs an `origin/HEAD` this repo doesn't have).

- Checked:
  - `/api/ws` checks the origin, the guest id, the name and the robot before the upgrade, as before. The map comes
    from the server, never from the client: a connection can't pick its map, nor a cell on it.
  - `enter` and `exit` are decoded strictly (whole numbers), under the 1 KB limit. Every request is counted by the
    connection's limit before anything else, a map's lookup included: 10 per second, on every map.
  - An Enter needs a dome on a chunk the player holds. An Exit's clicked cell is only measured against the edge, after
    being brought back into the map's box: a huge number can't overflow or reach outside.
  - Where a door leads is worked out from maps that never change while the server runs, read without their room's
    lock. No request holds two rooms' locks: a room hands a traveler over once its lock is released, and the locks
    are always taken session first, then room. The test of two rooms trading players both ways at once passes under
    `-race`.
  - A dome leads only to a map slug (lowercase words and dashes, 64 characters at most), checked when its map is
    loaded: `CheckDomes` looks for that folder among the map folders, never outside them.
  - The SQL is parameterized; only domes may lead somewhere (a check), and a rock leads nowhere (`NULL`).
  - The client drops a `welcome` without a valid map (sizes, `standsOn` in its map), a `planned` with an unknown
    action, and a dome without `leadsTo` or spilling out of its chunk. Map and region names are shown as text.
  - Leaving a map destroys its chunk textures, its frame callbacks, its pointer listener and its map screen: going
    back and forth doesn't grow the page.
  - Logs name players by their public id, as before.
- Known risks:
  - Anyone may `/teleport` to any map, as to any cell before (ADR 0010): a developer's tool until roles come.
  - A teleport to a map that doesn't exist asks the database each time (10 per second per connection at most). A limit
    per IP at the reverse proxy (ADR 0001) covers it, as for reconnecting in a loop.
  - The hub remembers the map of every guest seen since the server started, as rooms remember their cell: it grows
    with guests until accounts come.
  - An Exit's path in a bigger city may run outside the chunks the player holds, like any path (ADR 0003).

### Delivery

Three sub-tasks under #6, each playable on its own:

1. **The dome on the world map**: the object, its art, the generator, `domes` in `map.json`, migration `0007`,
   `reset-maps`. Clicking it shows Move off, like a rock.
2. **Several maps**: maps that don't wrap, Doma-1's map and art, one socket, the hub, Enter, Exit, `/teleport`
   across maps, the client switching maps, the map screen.
3. **Waking up in Doma-1**: new and returning guests, the spawn.

### Not now

- Each world cell's own 16 × 16 map, and the fights on it (`docs/FOR_LATER.md`).
- Buildings and NPCs inside the city.
- Doors as their own objects, and maps inside the city (floors, interiors).
- Where you are kept across server restarts, which accounts will need.
- The websocket's cost with many players switching maps: measured later.
- Players hidden behind the dome showing as a silhouette (`docs/FOR_LATER.md`).

## Test cases

### Back: maps and their rules (`core/domain`, `worldmap/domain`)

1. Sizes:
   - a map that doesn't wrap at 16 × 48, 32 × 32 or 4096 × 16 is accepted; at 24 or 4112 it's refused;
   - a wrapping map at 48 is refused, at 160 and 192 accepted.
2. Partial chunks: on a 48 × 48 map, chunk (1, 1) holds the cells (32, 32) to (47, 47) and `void` past them; a cell
   past the map's edge that isn't `void` gets the map refused.
3. The new kinds: `floor` is walkable and `wall` isn't; both are accepted at heights −3 to +3; a rock on `floor` is
   refused.
4. A dome:
   - its 4 cells aren't walkable, and the 12 around it are its sides;
   - one on a rock, on water, on two heights, or across a chunk border is refused;
   - one leading to a map with no folder, or two leading to the same map, get the map refused.

### Back: the generator and the city (`worldmap/domain`, `cmd/mapfile`)

5. The world: Doma-1's dome stands at the spawn (or one cell back), on grass at the spawn's height, its ring too, with
   no rock or hole; a path from the ring reaches a cell more than 64 rows from the ridge, on each side (ADR 0011
   test 5 still holds).
6. Doma-1's layout: 32 × 32, not wrapping; the park is the 6 × 6 middle in grass; floor up to 14.5 from the middle,
   a closed wall ring, `void` past it; the wall at +3 at the back, +1 at the front; the spawn is in the park; every
   floor and park cell is reachable from the spawn.
7. `city` refuses a folder with files without `-force`; the committed `mapfiles/doma-1` passes every check.

### Back: map files and storage (`worldmap/infra`, `worldmap/app`, Postgres)

8. `domes` in `map.json` writes and reads back the same; a dome whose map has no folder is refused, naming it.
9. The new colors: `floor`, `wall` and the gold dome have their colors in the layers, `README.md` and `palette.gpl`.
10. Migration `0007` adds `leads_to` and keeps every map.
11. Boot saves `doma-1` when only `world` is in the database, and nothing when both are; `reset-maps` replaces both.
12. A dome reads back with `leads_to`; `doma-1`'s description has `standsOn` at the dome's cell with the world's
    size; the world's has none.
13. The map image: a dome's 4 cells are gold; `floor` and `wall` have their colors; the palette holds at most 256.

### Back: travelling (`presence/app`, fake worlds and clock)

14. Connecting: a new guest is placed at Doma-1's spawn; a guest who left the world at a cell comes back there; one
    whose cell isn't walkable anymore goes to Doma-1's spawn.
15. Enter:
    - from afar: the walk goes to the closest side cell by path, then the guest is in Doma-1, on the edge on the side
      they came from (from the dome's east, the east edge);
    - from a side cell: in at once;
    - on a cell that isn't a dome's: `refused` with its reason, nobody moves;
    - with every side cell blocked: an empty path.
16. Exit:
    - the walk goes to the edge cell closest to the click, a click past the map included, then the guest is on the
      world beside the dome, on that side (the west edge gives the dome's west);
    - when that side's cell isn't walkable, the next closest by direction;
    - on the world: `refused`, "You're not in a city"; with no dome leading to the city: "Doma-1 stands nowhere on
      the world".
17. A move between maps:
    - the players who saw the traveler get `left`; those holding the arrival chunk get `joined`; the traveler gets
      `welcome` with the map, then its chunks;
    - the traveler's limiter carries over: 10 requests across a switch, then the 11th within the second is ignored;
    - a Move, a teleport, or another Enter during the walk to a door cancels going through.
18. `/teleport` across maps lands on the cell, walkable or not: `welcome`, chunks, then `teleported`; an unknown map
    or a cell outside the map named is refused before anything moves.
19. Two rooms switching guests at once in both directions never deadlock (run with `-race`).

### Back: websocket

20. `/api/ws` connects and gets `welcome` with Doma-1's description, then its chunk; the old
    `/api/maps/{slug}/ws` is gone (404).
21. `enter` and `exit` in the documented shapes reach the room; one with a missing or fractional cell is ignored.
22. `planned` carries `action`; a dome object carries `leadsTo`.

### Client

23. Protocol: `welcome` with a map decodes, without a valid one gives nothing; `planned` with each action decodes,
    with another gives nothing; a dome object decodes, without `leadsTo` or past its chunk gives nothing;
    `encodeEnter`, `encodeExit` give the documented JSON; `socketUrl` has no map.
24. The description: a 32 × 32 map that doesn't wrap parses, 24 × 24 throws; `standsOn` parses, a bad one throws.
25. A cell's action (`game/actions.ts`): a dome's 4 cells are Enter on the world; in a city, `void`, a `wall` and a
    cell past the map are Exit, `floor` and grass are Move; on the world, `void` is Move (off).
26. The menu: its action is the cell's, on once `planned` gives a path for that action, and choosing it sends
    `walk`, `enter` or `exit`.
27. The switch (`game/travel.ts`):
    - a `welcome` for the current map applies as before; one for another map starts the fade;
    - messages during the fade wait and are applied after `welcome`, in order;
    - the first `welcome` fades in only; the banner shows after the fade in.
28. Drawing: a dome is sorted by its front cell; the pointer over its opaque pixels picks it, over a transparent one
    the ground behind.
29. The map screen: Doma-1 shows at ×8 on a 1080p screen, the world as before; the donut shows in a city, with your
    mark at its dome's cell; the region line is left out when it's the map's name.
30. The log: "You enter Doma-1" and "You leave Doma-1" on a switch; nothing for a `welcome` on the same map.
31. Exploration is kept per map: back on the world, its explored chunks are still explored.
32. By hand: the preview images of the dome and of Doma-1's inside are approved before they're wired in.

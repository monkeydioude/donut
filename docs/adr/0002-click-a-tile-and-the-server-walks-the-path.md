# ADR 0002: Click a tile, choose Move, and the server walks the player along its path

- Status: accepted, 2026-11-09
- Replaces, from ADR 0001: the arrow keys, the client's `move` message and the server's `rejected` message.

## Context

Players move with the arrow keys, one cell at a time, each step checked by the server. They should instead:

1. point at a tile, which shows a border,
2. click it to select it, which shows the path,
3. choose Move in a menu, in the classic JRPG style.

The server walks the path, so a client can't cheat on it, and later on server events (traps, NPCs) can stop or
change a walk.

This is how tile-based click-to-move online games usually do it (RuneScape, Habbo Hotel):
- The server is authoritative and runs on a fixed tick.
- The client sends intents ("go there"), and the server moves players one tile per step.
- Clients glide the sprites between tiles.
- There's no client-side prediction: it suits direct-control action games, not this one.

## Decisions

### Pathfinding, on the server only

- A* on the map's grid, through the 8 neighbors.
  - A straight step costs 1 and a diagonal step costs √2, so paths go straight where they can.
  - The estimate is the octile distance.
  - Ties go to the lowest estimate, then to the cell found first, so the same request always gives the same path.
- No corner cutting: a diagonal step needs both cells beside it to be walkable.
- A path lists the cells to step on, the start excluded and the target included. It is empty when the target is the
  player's own cell, isn't walkable, or can't be reached.
- `Terrain.Path(from, to)` in `worldmap/domain`, next to `IsWalkable`.
- The client has no pathfinding: it asks the server, so the line it shows is the path the server walks.

### Websocket messages

- Client to server:
  - `plan {col, row}`: the path from where I stand to that cell.
  - `walk {col, row}`: walk there.
- Server to client:
  - `planned {col, row, path}`: the answer to `plan`, to the asker only. `col` and `row` repeat the asked cell, so
    the client can drop an answer to an older click.
  - `walking {path}`: to the walker only, the path the server will walk. Empty when there's nowhere to go; any
    current walk then stops.
  - `moved {id, col, row}`: to everyone, at each step, as before.
  - `stopped {col, row}`: to the walker only, when a step stops being walkable mid-walk (a future server event). The
    walk ends there.
  - `welcome` also gives `stepMs`, the time of one step, so the client glides at the server's pace without
    hard-coding it.
  - `cells` are `{col, row}` objects.
- `move` and `rejected` are removed. An old `move` is ignored, like any unknown message.
- A player may send at most 10 `plan` and `walk` per second; extra ones are ignored.

### Walking, on the server

- Each room runs a loop that ticks every 50 ms while the server runs. At each tick, every walker whose next step is
  due steps one cell, and everyone gets `moved`.
- A step every 200 ms, the same speed for everyone.
- The first step of a walk comes at the next tick if the player was standing still. If they were already walking,
  it comes 200 ms after their last step: walking again doesn't make anyone faster.
- A new `walk` replaces the current path, starting from the current cell.
- Leaving stops the walk. The player is remembered where the walk got to.
- Pathfinding runs under the room's lock. That's fine for maps up to 256 × 256; bigger maps would need it outside.

### Client

- The arrow keys are removed.
- **Hover**: the tile under the pointer gets a white diamond border. There's none outside the map, over the menu, or
  on the selected tile, whose yellow border shows instead.
- **Picking**: the pointer position is turned back into a tile (the inverse of the projection). Only the ground
  diamond counts, not the rocks or players drawn above it.
- **Select**: a click on a tile:
  - gives it a yellow border,
  - sends `plan`,
  - opens the menu next to the tile.

  When `planned` comes back for that tile, the path line shows and Move turns on. A click on another tile selects
  that one instead.
- **Menu**, mouse only:
  - Items: Move, and Cancel.
  - Move is greyed out while the path is coming, and when there's no path.
  - Move sends `walk` and closes the menu. Cancel, or a right click anywhere, closes the menu and clears the
    selection.
  - It opens to the right of the tile, kept inside the screen.
  - It's DOM over the canvas: a blue gradient window, a light border with rounded corners, white text with a dark
    shadow.
  - The font is Pixelify Sans (OFL), bundled with the app through `@fontsource`.
- **Path line**:
  - It starts where your player is drawn, gliding or not, then runs through the ground centers of the tiles to the
    target. It's redrawn every frame, so it stays attached to your sprite. It's white with a dark outline and ends in
    a small diamond.
  - It's drawn above the ground and under the rocks and players.
  - The overlay layer keeps a fixed order (borders, then the line) and its sprites have no `zIndex`. In Pixi 8, one
    child with a `zIndex` makes the whole layer sort.
  - It shows the `planned` path while selecting, then the `walking` path. It shortens as your player steps, and
    disappears on arrival or `stopped`.
- **Cursor**: the mouse pointer is a white pointing hand with a dark outline, pointing up-left; its fingertip is
  the hotspot.
  - It's 20 × 20 pixel art stored as data (a grid of characters and a palette), drawn into a 2× PNG and used as the
    CSS cursor on the page. 16 × 16 was too small for a diagonal finger.
  - It's an original sprite in the JRPG style.
- **Smooth steps**: sprites glide between cells instead of jumping, for every player, yours included.
  - On `moved`, a player's sprite glides from where it is shown to the new cell in `stepMs`, in a straight line.
  - A `moved` that comes mid-glide starts the next glide from where the sprite is, so network jitter only changes
    the speed a little.
  - A player who joins, or comes back on `welcome`, shows directly on their cell. `stopped` puts your sprite on its
    cell at once.
  - The game logic stays per cell: the server's cell decides picking, menus and paths, never the drawn position.
  - The sprite's draw order follows the cell it is gliding to, so it doesn't flicker behind a rock it walks past.
  - The glide position is a pure function of the start, the end, the start time and the time now. Pixi's ticker
    redraws every frame.
- Your selection and path live in their own state, next to the players' state.

### How it is built

- Back:
  - `worldmap/domain/path.go`: A* on `Terrain`.
  - `presence/domain`: the walk rules and the new events.
  - `presence/app/room.go`: `Plan`, `Walk`, `Tick`. `Move` is removed.
  - `Room.Run(ctx)` ticks the room; the hub starts it when it opens a room.
  - The clock stays injected: tests call `Tick` themselves after moving a fake clock.
- Client:
  - `iso/projection.ts` gets `toTile`.
  - New: `game/command.ts` (selection, path and menu state, pure), `render/overlay.ts` (borders and path line),
    `render/glide.ts` (smooth steps), `ui/menu.ts`, `ui/cursor.ts`, `art/finger.ts`, `art/tileOutline.ts`.
  - `game/keys.ts` and `game/keyboardMover.ts` are removed.

### Security review

- Fixed:
  - Each `plan` and `walk` runs A* under the room's lock. On a 256 × 256 map an unreachable target took 89 ms, so
    10 requests a second from one player could freeze the room.
  - The terrain now numbers its connected areas once, so a target in another area is refused without searching.
  - A search explores at most 8,192 cells (about 10 ms); past that, there is no path.
- Checked:
  - Paths go to the asker only.
  - Coordinates out of range get an empty path.
  - Rooms only open for maps that exist.
  - The menu shows text, not HTML.
  - The font and the cursor are bundled, with no outside requests.

### Not now

- Other players' paths.
- A cursor drawn in the game for gamepads.
- Keyboard control of the menu.
- More actions.
- Showing that a step was blocked other than through `stopped`.
- Client-side prediction.
- Batching per tick: one message per tick with all its changes. Worth it when many players share a map.
- Area of interest: sending each player only what happens near them. Needed for big maps.

## Test cases

### Back: pathfinding (`worldmap/domain`)

1. On an open map, a straight line: (0,0) to (3,0) gives (1,0), (2,0), (3,0).
2. Diagonal when it's shorter: (0,0) to (2,2) gives (1,1), (2,2).
3. It goes around a rock and never steps on one.
4. No corner cutting: (0,0) to (1,1) with a rock on (1,0) gives (0,1), (1,1).
5. No path when the target:
   - is a rock,
   - is outside the map,
   - is walled off,
   - is the start cell.
6. On random maps:
   - every step is a walkable neighbor of the one before,
   - a diagonal step never cuts a corner,
   - the path's cost equals a brute-force shortest cost.

### Back: walking (`presence/app`, fake clock, manual ticks)

7. `plan`: the asker gets `planned` with the path; the others get nothing; nobody moves.
8. `walk`: the walker gets `walking`. Then one step per 200 ms, each `moved` sent to everyone, until the target.
9. A new `walk` mid-walk replaces the path, and its first step still comes 200 ms after the last one.
10. A `walk` to an unreachable cell: `walking` with an empty path, and the current walk stops.
11. Leaving stops the walk; coming back, the player stands where the walk got to.
12. Over 10 `plan`/`walk` in a second, the extra ones are ignored.
13. A step that's no longer walkable: `stopped` to the walker, and the walk ends.
14. A replaced connection's `plan` and `walk` are ignored.

### Back: websocket

15. Two clients: A plans (B gets nothing), then A walks two cells. A gets `walking`, then both get two `moved`.
16. An old `move` and other unknown messages are ignored.

### Client

17. Picking: tile centers, points near a diamond's edge, and the four corners between tiles give the right tile;
    outside the map gives none.
18. Protocol: `planned`, `walking` and `stopped` decode; `plan` and `walk` encode; malformed ones give nothing.
19. Command state:
    - hover;
    - a click selects and waits for the path;
    - `planned` for that tile shows the path and turns Move on; one for an older tile is ignored;
    - Move gives the walking path;
    - your `moved` steps shorten it, and arriving clears it;
    - Cancel and right click clear everything;
    - a click on another tile re-selects.
20. Menu placement: to the right of the tile, kept inside the viewport near the edges.
21. Finger cursor sprite: its size, a transparent background, a small palette, and an opaque hotspot pixel.
22. Tile border sprite: only the diamond's edge pixels are painted.
23. Path line points: from where your player is drawn, through every path cell, at their ground centers.
24. Smooth steps:
    - a glide goes from the shown position to the new cell in `stepMs`, and stays there after;
    - a `moved` mid-glide starts from where the sprite is;
    - joining, `welcome` and `stopped` place the sprite with no glide;
    - the draw order follows the target cell.

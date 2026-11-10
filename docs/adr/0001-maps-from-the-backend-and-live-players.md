# ADR 0001: A Go backend serves the maps' layers, and a websocket shows the players on a map

- Status: accepted, 2026-11-09
- Moves replaced by ADR 0002:
  - the arrow keys,
  - the `move` and `rejected` messages,
  - the 100 ms move limit.

  The server now walks paths, and `welcome` also gives `stepMs`.

## Context

The client generates its own map (grass and rock tiles). To plug in real maps and play together, maps must come from
a server, and players must see each other live. No authentication yet.

## Decisions

### Layers

- A map has two stored layers:
  - `ground`: one tile kind per cell, row by row. Only `grass` for now.
  - `objects`: things standing on cells, `{id, kind, col, row}`, at most one per cell. Only `rock` for now; trees
    and fences later.
- Rocks move from the ground to the objects: a rock tile becomes grass with a rock object on it.
- Players are not a layer. They are live: kept in memory per map and sent over the websocket. On screen they are
  sorted with the objects, so a player walks behind a rock.
- A cell is walkable when it is inside the map and has no object.

### Storage

- Postgres 16, schema `donut`, in Donut's own container (docker compose, port 5434). `make db` starts it.
- `maps`: `id`, `slug` (unique, used in URLs), `name`, `width` and `height` (1 to 256), `ground` (JSONB array of
  width × height kinds), `created_at`, `updated_at`.
- `map_objects`: `id`, `map_id` (deleted with its map), `kind`, `col`, `row`, `created_at`; unique per map and cell.
- Migrations as in Kabane: `back/migrations/init.sql`, then `back/migrations/up/NNNN.up.sql`, recorded in
  `schema_migrations`. They are embedded in the binary and applied at boot, under a Postgres advisory lock so two
  servers never apply them together. Each file runs in its own transaction. A failing migration stops the boot.

### Starter map

- At boot, after the migrations: with no map in the database, the server generates `meadow` and saves it in one
  transaction. It's 16 × 16 grass with rocks on about 12% of the cells, from the seed `STARTER_MAP_SEED`.
- Map generation moves to Go. The client no longer generates maps.

### HTTP API, no authentication

- `GET /api/maps`: `[{id, slug, name, width, height}]`.
- `GET /api/maps/{slug}`: `{id, slug, name, width, height, layers: {ground: [...], objects: [{id, kind, col, row}]}}`.
  An unknown slug gets a 404.

### Websocket: the players on a map

- `GET /api/maps/{slug}/ws?player={uuid}&name={name}`. These are refused before the upgrade:
  - A player id that is not a UUID, or a name that isn't 1 to 24 characters once trimmed: 400.
  - An unknown map: 404.
  - An `Origin` other than `FRONT_ORIGIN`: 403.
- JSON messages, each with a `type`:
  - Server to client:
    - `welcome {you, players}`: everyone on the map, you included.
    - `joined {player}`.
    - `moved {id, col, row}`.
    - `left {id}`.
    - `rejected {col, row}`: your move was refused, and this is where you are.
  - Client to server: `move {col, row}`, the target cell.
  - Bad JSON and unknown types are ignored; the connection stays open. A message over 1 KB closes the connection
    (status 1009, message too big).
- A move is accepted when:
  - the target is one of the 8 neighbor cells, and it is walkable,
  - and it comes at least 100 ms after the player's last accepted move.

  Then `moved` goes to everyone on the map, the mover included. Otherwise `rejected` goes to the mover only. Players
  don't block each other.
- Spawn: the player's last position if this guest id was on this map since the server started. Otherwise the free
  cell closest to the map center.
- The same guest id connecting again (second tab, refresh) replaces its old connection, which is closed. The others
  see no new `joined`.
- A client too slow to read (64 messages waiting) is disconnected.
- Players live in memory only: a restart forgets them.
- The guest id is a secret: whoever sends it plays that player. Messages only carry a player's public id, the first
  16 bytes of the SHA-256 of their guest id, in hex. `you` in `welcome` is your public id.

### Client

- Guest identity in localStorage: a random UUID and a `Guest-1234` name, created once. If storage fails, a new
  identity is used for the session.
- It loads `/api/maps/meadow` and draws the ground layer. Above it, one depth-sorted container holds the objects and
  the players.
- The arrow keys and WASD move one cell per press, and one every 150 ms while held. They follow the screen:
  - up: (col − 1, row − 1)
  - right: (col + 1, row − 1)
  - down: (col + 1, row + 1)
  - left: (col − 1, row + 1)

  Two keys together give the cell in between; opposite keys cancel. The client waits for the server's `moved`: no
  prediction for now.
- The rock art becomes an object sprite: the boulder without its grass, with a semi-transparent shadow.
- A placeholder player sprite is generated in the same style; your own player has another color.
- Vite runs on port 5180 and proxies `/api`, HTTP and websocket, to the back on 8400, so no CORS is needed.

### How it is built

- `back/`, Go module `donut`, laid out like Kabane:
  - `cmd/donut`: `main.go`, and `routing.go`, the composition root.
  - `internal/<module>/{compose.go, domain, app, infra, transport}`.
  - `pkg/`: `db`, `migrate`, `pg`, `errd`, `pgtest`.
- Packages, the same as Kabane: `calqs/gopkg` (`router`, `env`, `dt`), `pgx/v5`, `oklog/run`, `godotenv`, `testify`,
  `google/uuid`. New: `coder/websocket`.
- Modules:
  - `worldmap`: the maps, their layers, the starter map and the HTTP routes.
  - `presence`: the players on a map and the websocket. It checks moves through a `Walkable` interface that
    `worldmap` provides, from a copy of the map in memory per room.
- No global state: the pool, the config and the clock are passed in.
- Config from the environment:
  - `API_PORT`: `:8400`.
  - `DB_URL`: `postgres://donut:donut@localhost:5434/donut?search_path=donut`.
  - `FRONT_ORIGIN`: `http://localhost:5180`.
  - `STARTER_MAP_SEED`: `42`.
- `make dev` starts the database, the back (reloaded by air) and the front together.

### Security review

- Fixed:
  - Messages carried the guest ids, so anyone could connect with someone else's and take over their player. They now
    carry public ids.
  - Postgres listened on every network interface with the dev password. It now listens on `127.0.0.1` only.
- Known risks, for when authentication comes:
  - Anyone can open as many connections as they want, each one a new player told to everyone.
  - Every guest id ever seen keeps its last cell in memory until a restart.

  A limit per IP, for example at the reverse proxy, covers both until then.

### Not now

Authentication, saving players, editing maps, live map changes, ground in chunks for big maps, smooth movement and
client-side prediction.

## Test cases

### Back: map rules (`worldmap/domain`)

1. The same seed gives the same map. Every ground cell is grass. Rocks are inside the map, at most one per cell.
   A rock chance of 0 gives no objects; 1 gives a rock on every cell.
2. A map is refused when:
   - its size is outside 1 to 256,
   - its ground doesn't have width × height cells,
   - a tile or object kind is unknown,
   - an object is outside the map or shares a cell with another.
3. Walkable: an empty cell yes; a rock no; outside the map no.
4. Spawn: the center when free, otherwise the closest free cell; a full map gives an error.

### Back: migrations (`pkg/migrate`, Postgres)

5. On an empty schema, `init.sql` and every up file are applied and recorded. A second run applies nothing.
6. A broken migration returns an error and isn't recorded.

### Back: maps in the database (`worldmap/app`, Postgres)

7. Without maps, the starter map `meadow` is created with its ground and objects. A second boot creates nothing.
8. `meadow` reads back with both layers as saved. An unknown slug gives "not found".
9. The list shows the maps without their layers.

### Back: maps over HTTP

10. `GET /api/maps/{slug}`: 200 with the layers; an unknown slug gives 404.

### Back: players on a map (`presence/app`, fake connections and clock)

11. Join: the newcomer gets `welcome` with everyone, itself included; the others get `joined`.
12. Spawn at the center, or on the closest free cell. A returning guest gets its last position.
13. A valid move: `moved` to everyone, mover included.
14. These moves get `rejected` with the current position, to the mover only; the others get nothing:
    - onto a rock,
    - outside the map,
    - more than one cell away,
    - onto its own cell.
15. A move less than 100 ms after the last accepted one is rejected.
16. Leave: the others get `left`. The last position is kept.
17. The same guest joining again: the old connection is closed, there's still one player, and the others get no
    second `joined`.
18. A client with a full queue is disconnected; the others still get their messages.

### Back: websocket (test server, real websocket)

19. Two clients on one map: B sees A join, move and leave.
20. Refused before the upgrade:
    - a bad player id or name: 400,
    - an unknown map: 404,
    - a wrong `Origin`: 403.
21. Bad JSON or an unknown type is ignored, and the connection stays open. A message over 1 KB closes it.

### Client

22. Each server message decodes; an unknown or malformed one gives nothing.
23. The players' state follows `welcome`, `joined`, `moved` and `left`. `rejected` puts your player back.
24. Keys to steps: each key and each two-key diagonal gives the right cell; opposite keys cancel.
25. Guest identity: created once, then read back. A storage failure gives a new identity for the session.
26. A map from the API becomes ground and objects; an invalid one gives an error.
27. Rock sprite: only boulder and shadow pixels. Player sprite: a small palette, transparent around it.

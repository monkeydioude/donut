# ADR 0006: Players are robots from the Pixel Art Atlas, one given at random to each guest

- Status: accepted, 2026-11-10

## Context

Players are a placeholder drawn in code: a round body and head, your own in orange, the others in blue. The game
starts you as a battered robot. The [Pixel Art Atlas](https://huggingface.co/datasets/Angais/pixel-art-atlas-1.4m)
holds about 3,900 robot sprites (16 × 16, transparent, AI-made), and 14 of them fit: battered, full body, facing
the viewer.

## Decisions

### The robots

14 robots, each with a slug, taken from the atlas's `train` split:

| Slug                | Atlas `row_id` | Caption                                      |
|---------------------|----------------|----------------------------------------------|
| `dome-head`         | 797552         | dented robot with a dome head                |
| `rusty-toy`         | 222810         | worn toy robot with a sparkle                |
| `scrap-humanoid`    | 548926         | damaged humanoid robot, exposed arm panel    |
| `drill-worker`      | 519236         | dented worker robot with a drill arm         |
| `cracked-companion` | 45321          | silver robot companion with a cracked casing |
| `round-bot`         | 1264097        | round robot companion with a cracked casing  |
| `hard-hat`          | 1000213        | dented worker robot with a hard hat          |
| `plastic-toy`       | 257408         | worn plastic toy robot                       |
| `jumpsuit`          | 938094         | damaged humanoid robot with a jumpsuit       |
| `dial-eyes`         | 1227085        | damaged boxy robot with dial eyes            |
| `gray-companion`    | 391771         | damaged gray robot companion                 |
| `square-head`       | 713341         | dented boxy robot with a square head         |
| `toolbox-worker`    | 438868         | damaged worker robot with a toolbox          |
| `orange-wires`      | 49669          | old orange robot companion, exposed wires    |

The `row_id` is the dataset's own id of a row. Robots are fetched by it: a row's position in the split changes
between the dataset API's calls.

A 15th, `wired-screen` (a boxy robot with a screen face and wires), was dropped: the API's image cache now serves a
potion bottle for its row. That cache is keyed by those unstable positions, so it can serve another row's sprite,
which is also why the tool pins each sprite's pixels (below).

### Each guest gets one at random

- The client picks a robot when it creates the guest, each one equally likely, and saves it in localStorage with the
  guest's id and name.
- A guest saved before this ADR keeps their id and name and gets a robot on their next load, saved too.
- A saved robot that isn't in the sheet anymore is replaced by a new random one, saved too.
- Without storage, the robot lasts for the session, like the guest.
- No way to choose or change your robot yet (`docs/FOR_LATER.md`).

### On the wire

- The websocket URL gets the robot: `GET /api/maps/{slug}/ws?player={uuid}&name={name}&robot={slug}`.
- The server checks only its format: 1 to 32 lowercase letters, digits or dashes. Anything else is refused before
  the upgrade with a 400. A missing or empty `robot` is accepted, as `""`. The server doesn't know the list of
  robots: they're cosmetic, and robots can be added or removed without touching the back.
- Every player the server sends (`welcome`, `joined`, and the others listing players) gets a `robot` field.
- A guest connecting again while still connected keeps the robot of their first connection, as they keep their
  name (ADR 0001). The others see the new robot only once that player has left and come back.
- A client draws the sheet's first robot (`dome-head`) for a slug it doesn't know, or an empty one.

### On screen

- A player is drawn as three layers, bottom to top, standing on its cell's ground center:
  1. **Your ring**, on your own robot only: a thin ellipse ring, yellow `0xffd84a` like the selection border, a bit
     wider than the shadow.
  2. **The shadow**: our usual semi-transparent ground shadow, painted in code.
  3. **The robot**: its 16 × 16 cell from the sheet, the cell's bottom center on the ground center. Every robot's
     feet are on its cell's bottom row.
- The robot is drawn crisp (nearest filtering), at the world's zoom like the rest of the art.
- One frame only, facing the viewer: no walking animation or facing directions yet.
- The placeholder painter (`paintPlayer`, `PlayerLook`, the shirt colors) is deleted.

### The sheet

- `client/public/sprites/robots.png`: the 14 robots in one row of 16 × 16 cells, in the table's order (224 × 16).
- `client/public/sprites/robots.json`: `{ "cell": 16, "sprites": [{ "slug", "x", "y", "rowId" }] }`, `x` and `y`
  being the cell's top-left corner in the PNG.
- Both are committed. The client loads them at boot, alongside the map. If either fails or the JSON is malformed,
  the game stops with a notice: "Could not load the sprites."
- `client/public/sprites/CREDITS.md`: where they come from and their license (below).

### The tool that makes the sheet

- `go run ./cmd/spritesheet -picks cmd/spritesheet/robots.json -out ../client/public/sprites/robots`, from `back/`,
  or `make sprites`.
- The picks file holds the dataset, its config and split, and each robot's slug, `row_id` and `sha256`: the hash of
  the sprite's pixels as picked (alpha snapped, before the recolor).
- For each pick, one at a time:
  1. it asks the dataset's API for the row with that `row_id` (`datasets-server.huggingface.co/filter`,
     `where "row_id"=N`), and stops if the row that comes back has another `row_id`: the dataset changed;
  2. it downloads the row's PNG, only from the API's own host, and stops if it isn't 16 × 16;
  3. it stops if the sprite's pixels don't match the pick's `sha256`: the API's image cache served another row's
     sprite. A rebuild gives exactly the sprites picked, or fails naming the robot;
  4. it recolors every near-black pixel (each channel at most 40) to our outline color `0x3f3550`, and snaps alpha
     to 0 or 255.
- Each request gets 90 s and up to 5 tries, 5 s apart: the filter scans the whole dataset, often for more than
  30 s, and the API rate-limits and sometimes fails.
- It writes the PNG and the JSON, and prints each robot as it's done.
- Needs the network only when it runs. Duplicate slugs, or slugs not matching the format, stop it before anything is
  downloaded.

### Where the sprites come from

- They are AI-made (Krea 2 Turbo with a pixel art LoRA). The dataset is CC0, but the sprites are also Krea 2
  "Outputs", under the [Krea 2 Community License](https://krea.ai/krea-2-licensing):
  - commercial use only while the company makes under $1M a year (all revenue, trailing 12 months); above that, a
    Krea enterprise license;
  - whoever uses them is responsible for them, including if one resembles someone else's IP;
  - AI-made content must be disclosed where required (Steam asks).
- AI-made art has little copyright: others may reuse these robots.
- `CREDITS.md` says all of this next to the assets, for whoever ships the game.

### Logs

Debug logs for:
- the sheet loaded (robots, time) or failed;
- the robot picked for a new guest, kept, or replaced (and why);
- a player drawn with the default robot because its slug is unknown;
- the server: the robot in the `ws open` and `join` logs.

### How it is built

- Back:
  - `presence/domain/player.go`: `Robot` on `Player`, and `CleanRobot`, the format check.
  - `presence/transport/ws`: `robot` read from the URL, refused with a 400 when malformed, and sent in every player.
  - `presence/app/room.go`: `Join` takes the robot and keeps it on the member.
  - `internal/spritesheet`: the picks file, the recolor, the sheet's PNG and JSON (pure), and the dataset client
    behind an interface.
  - `cmd/spritesheet`: flags, the HTTP client, files. The `sprites` Makefile target.
- Client:
  - `art/robotSheet.ts`: reading and checking `robots.json`, and the slug to draw for any slug (default included).
  - `art/playerMarks.ts`: the shadow and your ring, painted in code.
  - `net/guest.ts`: the robot picked, kept or replaced, with the list of slugs passed in.
  - `net/protocol.ts`: `robot` in `PlayerInfo` and in the websocket URL.
  - `render/robotTextures.ts`: loading the PNG, crisp, and cutting one texture per robot.
  - `render/playersView.ts`: a player as a container of its layers.
  - `main.ts`: the sheet loaded with the map, the notice, the guest created after.

### Security review

- The robot reaches the server only as a format-checked slug: at most 32 characters, no spaces, quotes or markup.
  It's never shown as text, only used to pick a texture.
- A client can claim any robot, including unknown slugs: it only changes how they're drawn.
- The tool only runs on a developer's machine. It reads one known host over HTTPS, and downloads sprites from that
  host only. Each response is at most 1 MB, and a PNG's declared size is checked before it's decoded. It writes
  only `<out>.png` and `<out>.json`.

### Not now

- Choosing or changing your robot, and the server picking it once accounts exist.
- Mapping the robots' colors onto our palette.
- Walking frames and facing directions.
- Item icons from the same atlas (`docs/FOR_LATER.md`).

## Test cases

### Back: the robot (`presence/domain`)

1. `CleanRobot` accepts `dome-head`, `a`, `robot-2` and 32 characters.
2. It accepts `""` as `""`.
3. It refuses 33 characters, uppercase, spaces, `_`, `/`, quotes, `<`, and non-ASCII letters, as a bad request.

### Back: the websocket (`presence/transport/ws`)

4. A malformed `robot` gets a 400 before the upgrade.
5. A missing `robot` connects, and the player is sent with `robot: ""`.
6. `welcome` and `joined` carry each player's `robot`.

### Back: the room (`presence/app`)

7. A guest joining with a robot is told to the others with it.
8. A guest connecting again while connected keeps the robot of their first connection.

### Back: the sheet (`internal/spritesheet`)

9. A near-black pixel (every channel ≤ 40) becomes `0x3f3550`; a pixel with one channel at 41 stays; transparent
   pixels stay transparent.
10. Alpha is snapped: under 128 becomes 0, the rest 255.
11. The sheet is the robots in one row, in the picks' order, each in its 16 × 16 cell; the JSON lists each slug with
    its cell and `row_id`.
12. An image that isn't 16 × 16 stops the tool, naming the robot.
13. A row whose `row_id` isn't the expected one stops the tool, naming the robot.
14. A failed request is tried again, up to its number of tries (5 by default), then stops the tool.
15. Duplicate or malformed slugs, or a missing or malformed `sha256`, in the picks stop the tool before any request.
16. A sprite whose pixels don't match its pick's `sha256` stops the tool, naming the robot.

### Client: the sheet (`art/robotSheet`)

17. A well-formed `robots.json` is read: the cell size, and each slug with its cell.
18. A malformed one is refused: not an object, missing or empty sprites, a duplicate slug, a cell outside the sheet
    or not a whole number.
19. A known slug draws itself; an unknown or empty slug draws the first robot.

### Client: the guest (`net/guest`)

20. A new guest gets a robot from the list, picked with `randomInt`, and it's saved.
21. A saved guest with a known robot is read back unchanged.
22. A saved guest without a robot keeps their id and name, gets a random robot, and it's saved.
23. A saved guest whose robot isn't in the list keeps their id and name, gets a new random robot, and it's saved.
24. Without storage, the guest still gets a robot.

### Client: the protocol (`net/protocol`)

25. The websocket URL carries `robot`.
26. A player with a `robot` string is decoded with it; a missing or non-string `robot` gives `""`.

### Client: the marks (`art/playerMarks`)

27. Your ring is yellow `0xffd84a`, a closed one-pixel ellipse around the ground center, wider than the shadow, and
    empty inside.
28. The shadow is the usual semi-transparent ellipse at the ground center.

# ADR 0010: A chatbox with a game log, and the `/teleport` command

- Status: accepted, 2026-11-11
- Replaces, from ADR 0004: "Keys typed in a text field are left alone (there are none yet: the chat will need it)":
  there is one now.

## Context

Getting somewhere on a 2048 × 1024 world takes a long walk. Developers need to jump to a place to look at it: the
inner ridge, a continent's coast, a bridge. The game also has no written trace of what happens around you: a player
appears, walks, vanishes, and nothing says so.

A chatbox at the bottom of the screen gives both: a place to type commands, and a log of what happens in view. Chat
between players comes later.

## Decisions

### The chatbox

- **Where:** a bar across the bottom of the screen, DOM over the canvas like the banner and the menu: a history of
  6 visible lines above one input line. Cream text on the notice's dark purple, half transparent, in Zen Maru Gothic.
- **Opening:** `Enter` focuses the input: a new `chat` action of the world map state, bound to `Enter`. The map
  screen (ADR 0005) has no chat. While the input has focus, the game's keys don't fire (ADR 0004 already leaves text
  fields alone): `Y`, `Z`, `X` and `Space` type letters and spaces.
- **Sending:** `Enter` sends the line and clears it; the input keeps focus. `Escape` blurs it: the game's keys work
  again. An empty line does nothing.
- **Lines** are either yours (what you typed, echoed as is), the log's, a warning or an error, each styled apart.
  Text only, never HTML: names and reasons are set as text nodes.
- **History:** every line is kept, up to 1,000: past that, the oldest goes. A new line scrolls the view to the
  bottom.
  - `Up` and `Down` scroll one line, while the input has focus only. In the game, the arrows stay free.
  - The wheel over the chatbox scrolls it, and doesn't zoom the game.
  - No command recall, no timestamps, nothing saved across loads.

### Commands

- A line starting with `/` is a command. The only one is **`/teleport <map> <col> <row>`**: `col` and `row` are
  whole numbers. `/teleport world 1023 511`.
- Answered on the client, without the server:
  - a line that isn't a command: "Chat comes later: only /teleport <map> <col> <row> works for now";
  - an unknown command: "Unknown command: /foo";
  - bad arguments (not 3, or col or row not a whole number): "Usage: /teleport <map> <col> <row>".
- A well-formed teleport is sent to the server: `{"type": "teleport", "map": "world", "col": 1023, "row": 511}`.
  The client echoes the line and waits for the answer.
- **Listed while typed:** the input's placeholder says "Enter to type. Commands start with /". While the input has
  focus and its line starts with `/`, the commands it may be are listed in a panel just above the input, over the
  history: each one's usage (`/teleport <map> <col> <row>`) and what it does. While the name is typed, the commands
  it starts (`/` lists them all, `/te` the teleport); once a space follows the name, only the one it names. Case
  sensitive, like the parsing. Nothing is listed for plain text, an unknown name, or when the input loses focus. The
  list only shows: it can't be clicked or picked with keys.

### The teleport, on the server

- **Anyone may teleport**, for now: there are no accounts nor roles. See the security review.
- It runs under the room's lock like `walk`, and counts in the same limiter (10 requests per second).
- **Refused**, with a `refused` message to the asker and nothing else:
  - a map that isn't the room's: "unknown map: mars". Other maps are other rooms, so a cross-map teleport will mean
    reconnecting: later;
  - a cell outside the map: "(5000, 2) is outside the map". Cells are told within the map, as everywhere (ADR 0003).
- **Done** otherwise, walkable or not: a teleport onto the ocean, a mountain or a rock lands there. The asker gets
  `teleported {col, row, walkable}`, and the client warns when `walkable` is false: "(1023, 511) isn't walkable:
  teleport again to leave". There is no walking off an unwalkable cell: a path needs a walkable start. On reload, a
  player whose last cell isn't walkable respawns at the spawn point, as before.
- A teleport **drops the walk** under way, and the client cancels the selection and its path.
- **Then, as after a step** onto another chunk: the asker's chunks follow them (the ones coming into range are sent,
  the far ones forgotten, with `joined` and `left` for the players on them), and every other player gets `moved`,
  `joined`, `left` or nothing, by which of the two cells they see. A far jump forgets every chunk held and sends 9 new
  ones. Nothing new is sent to the others: from their side a teleport is a player leaving view or coming into it.
- **Order** for the asker: `teleported`, then the chunks that came, `joined`, `left`, `forget`, like a step.

### The game log

The client writes a line for what it learns, in one voice, from the messages it already gets:

| Message                                 | Line                                                       |
|-----------------------------------------|------------------------------------------------------------|
| `joined`, another player                | "Robo entered the area"                                    |
| `left`                                  | "Robo left the area"                                       |
| `moved`, another player, after a pause  | "Robo is walking": the first step after 1 s without one   |
| Move chosen in the menu                 | "You head to (1040, 500)"                                  |
| `stopped`                               | "You stopped at (1030, 505)"                               |
| `teleported`                            | "Teleported to world (1023, 511)", and the warning above   |
| `refused`                               | its reason, as an error                                    |

- A player's steps aren't logged one by one: a walk is one line, and a new walk starts after 1 s without a step.
- Nothing is logged for `welcome`, chunks, `planned`, `walking`, nor your own steps.
- Entities other than players will get their lines when they exist. Rocks do nothing.

### Websocket messages

- Client to server, new: `{"type": "teleport", "map": "world", "col": 1023, "row": 511}`. `plan` and `walk` are
  unchanged.
- Server to client, new:
  - `{"type": "teleported", "col": 1023, "row": 511, "walkable": true}`;
  - `{"type": "refused", "request": "teleport", "reason": "unknown map: mars"}`.

  The client drops a `teleported` without whole-number cells, or a `refused` whose reason isn't a string. It shows
  the reason as text.

### Logs

Debug logs for:

- the server: a teleport asked, refused with why, or done with the cell, its walkability and the chunks swapped;
- the client: a command parsed or refused locally, a teleport sent, and the chatbox opened or closed.

### How it is built

- Back:
  - `presence/domain/event.go`: `Teleported`, `Refused`; `presence/domain/teleport.go`: the rules: the map, the
    cell inside the map, walkable or not;
  - `presence/app/room.go`: `Teleport`, placing the member, then following them like `step` does (the chunk
    following and the telling of the others are shared with `step`);
  - `presence/transport/ws/messages.go` and `handler.go`: the `teleport` request, the two new messages.
- Client:
  - `game/commands.ts`: parsing a line into a command or a local answer, pure;
  - `game/gameLog.ts`: the lines from messages and actions, with the walk pause, pure;
  - `ui/chatbox.ts`: the DOM: the history, the input, the keys and the wheel;
  - `net/protocol.ts`: `encodeTeleport`, and decoding `teleported` and `refused`;
  - `game/players.ts`: `teleported` places you; `game/command.ts`: it cancels the selection and the walk;
  - `game/session.ts`: wiring the chatbox, the log and the socket;
  - `input/bindings.ts`, `config.ts`: the `chat` action on `Enter`;
  - `index.html`: the chatbox's styles.

### Security review

- Checked:
  - The `teleport` request is bounded like the others: under the 1 KB message limit, decoded strictly (the map a
    string, the cells whole numbers), and counted by the per-player limiter. A teleport runs no pathfinding: it's
    cheaper than a walk.
  - The map name is compared, never used to look anything up: a request can't open another room.
  - A cell outside the map is refused before anything moves; inside, the cell is normalized nowhere: it's used as
    given, within the map.
  - Everything shown in the chatbox is set as text: names, reasons, the echoed line.
  - A far teleport makes the server send 9 chunks, like a join: about 29 KB. Ten per second per player is the
    limiter's bound, so a player can pull 290 KB/s of chunks by teleporting in a loop. A limit per IP at the reverse
    proxy (ADR 0001) covers it, as for reconnecting in a loop.
- Known risks:
  - Anyone can teleport anywhere, onto anything, as decided: it's a developer's tool on a game with no accounts.
    When roles come, the command moves behind one. Until then, a player can stand on the ocean or inside a mountain
    pack.
  - The chatbox keeps up to 1,000 lines of DOM: a few hundred KB at most.

### Not now

- Chat between players, and names in front of lines.
- Teleporting to another map: switching rooms and sockets.
- Roles: who may run which command.
- Command recall with the arrows, and tab completion.
- Lines for entities: NPCs, traps, chunk changes.

## Test cases

### Back: the teleport's rules (`presence/domain`)

1. A teleport is checked: a map that isn't the room's is refused with "unknown map: <map>"; a cell outside the map
   with "(col, row) is outside the map"; a cell inside is accepted, with whether it's walkable.

### Back: the room (`presence/app`, fake world and clock)

2. A teleport on a walkable cell: the asker gets `teleported` with `walkable` true, then the chunks around the new
   cell, their players as `joined`, `left` for the players on the chunks forgotten, and `forget` for those chunks.
   Their walk under way is dropped: the next tick moves nothing.
3. A teleport on an unwalkable cell lands there, with `walkable` false. A `walk` from there gets an empty path.
4. A far teleport forgets every chunk held and sends the 9 new ones; a teleport within the same chunk sends no chunk.
5. The others: an observer who saw the old cell and sees the new one gets `moved`; only the new one, `joined`; only
   the old one, `left`; neither, nothing.
6. Refused: an unknown map, or a cell outside the map, gets `refused` with its reason and moves nobody; nothing is
   sent to the others.
7. A teleport counts in the limiter: the 11th request within a second is ignored. A replaced connection's teleport is
   ignored.

### Back: websocket

8. A `teleport` message in the documented shape reaches the room; one with a missing field, a non-integer cell or a
   non-string map is ignored like any unknown message.
9. `teleported` and `refused` are sent in the documented shapes.

### Client

10. Commands (`game/commands.ts`):
    - "/teleport world 1023 511" gives a teleport to that map and cell;
    - "hello" gives the chat-comes-later line; "/fly" the unknown-command line; "/teleport world 1" and
      "/teleport world a b" the usage line; " /teleport world 1 2 " with spaces around still parses;
    - "/teleport world 1.5 2" gives the usage line; "/teleport world -1 2" is sent as is: the server refuses it as
      outside the map.
    - The commands listed: "/" lists them all; "/te" those whose name starts with it, " /w" too; "/fly" and "/Te"
      none; "/tel " (a space after) none, "/tell " and "/teleport world 1 2" the one named; "" and "hello" none. The
      teleport's usage is the one its usage line gives.
    - In `ui/chatbox.ts`, checked by hand: the list shows on "/" and filters as you type, hides on send, on blur and
      for plain text.
11. Protocol: `encodeTeleport` gives the documented JSON; `teleported` and `refused` decode; a `teleported` with a
    missing or fractional cell, or a `refused` without a string reason, gives nothing.
12. Players: `teleported` moves your player to its cell; the view places it (no glide).
13. Command state: `teleported` clears the selection, its path and the walk.
14. The log (`game/gameLog.ts`):
    - `joined` and `left` for another player give their lines with the name; for you, nothing;
    - another player's first `moved` gives "is walking"; the next within 1 s nothing; one after 1 s a new line;
    - Move chosen, `stopped`, `teleported` (walkable or not: the warning too) and `refused` give their lines;
    - `welcome`, `chunk`, `forget`, `planned`, `walking` and your own `moved` give nothing.
15. The chatbox's logic (`game/chatHistory.ts`, pure: the tests run without a DOM, so `ui/chatbox.ts` stays a thin
    layer over it):
    - 6 lines are visible, the last ones; the 1,001st line drops the first; a new line scrolls back to the bottom;
    - scrolling up shows earlier lines, one per step, and stops at the first line; scrolling down stops at the
      last;
    - `Enter` sends, `Escape` blurs, `ArrowUp` and `ArrowDown` scroll; other keys do nothing to the chatbox.
    - In `ui/chatbox.ts`, checked by hand: lines are set as text, so "<b>x</b>" shows the tags; an empty line sends
      nothing; a wheel over the chatbox scrolls it and doesn't zoom the game.
16. Controls: `Enter` in the world map state triggers `chat`; while the input has focus, `KeyY` doesn't toggle the
    lock; in the map state, `Enter` does nothing.

# ADR 0005: The map screen, with the world laid flat and the donut seen from above

- Status: accepted, 2026-11-10

## Context

You only see the few chunks around you, and nothing tells you where you are on the planet. The world is the surface
of a donut (ADR 0003): laid flat, it's a rectangle that wraps both ways. A map screen should show that rectangle,
what you've explored, and where you stand on the donut itself.

## Decisions

### The map state

- **M** opens the map from the world map. M goes by its printed letter (`m` or `M`), so it's M on any layout: an
  exception to letters going by position (ADR 0004), like + and −.
- **M or Escape** closes it.
- The map is its own control state, `map` (ADR 0004). While it's open, the world map's controls stop: no zoom, pan
  or Y. A dim backdrop over the game catches the pointer, so tiles get no hover or click, and the camera icon no
  click. Clicks on the map do nothing.
- Opening it cancels the selection, which closes the action menu.
- The game keeps running under it: your player keeps walking, and the map follows live.

### The screen

```
┌───────────────────────────────────────────────┐
│                     World               ╭─╮   │
│                  Jelly Ridge            ╰─╯   │   the donut, seen from above
│                                               │
│          the world laid flat, 1024 × 512      │
│                                               │
│                                 1.8% explored │
└───────────────────────────────────────────────┘
```

- No window around it: the map is the whole block, centered over the dim backdrop, with a thin dark outline.
- **The rectangle:** columns across, rows down, column 0 on the left. It never scrolls. It's shown at ×2, ×1 or ×½,
  the biggest that fits the screen with a 24 px margin, drawn crisp.
- **Over the map**, in cream letters with a soft dark outline, like the region banner:
  - **top center:** the map's name, with your current region's name under it, nothing for region 0;
  - **top right:** the donut, on a map that wraps;
  - **bottom right:** the explored share, with one decimal: "1.8% explored".
- Your mark on the rectangle stays above all of it; the other players' dots go under the names and the donut.

### The image, from the server

- `GET /api/maps/{slug}/image.png`: the whole map, 1 pixel per cell.
  - Each region has its own shade of the grass green: 12 shades, close to each other but easy to tell apart.
    Region 0 is the grass green itself.
  - A cell whose region differs from one of its 4 neighbors (across the seams on a wrapping map) is a border: a
    darker shade of its region.
  - Rocks are light gray, drawn over borders.
  - `void` is transparent.
- It's rendered from the map's chunks, regions and rocks the first time it's asked for, then kept in memory: chunks
  don't change live yet. Renders go one at a time, so a burst of first requests renders once and the others wait
  for it. Only maps that exist are kept.
- Sent with `Content-Type: image/png`, `X-Content-Type-Options: nosniff`, `Cache-Control: no-cache` and an `ETag`
  (a hash of the PNG). A request whose `If-None-Match` matches gets `304 Not Modified`, without the image.
- An unknown slug gets a 404.
- The client loads it the first time the map opens, so a visit that never opens the map doesn't pay for it: about
  60 KB for `world`, as a paletted PNG. If it fails, the block says "Couldn't load the map." instead of the map,
  and the names, the donut and the share still show.

### Explored

- A chunk is explored once you've held it this session. That's chunk by chunk, and it includes the ring of chunks
  around where you walked, since you're sent those too. Forgetting a chunk doesn't unexplore it.
- The rest of the map is under a dark blue-gray veil, about 55% opaque, so the land still shows through.
- A chunk arriving while the map is open clears its veil at once.
- The share is explored chunks out of all the map's chunks (512 for `world`). It starts again on each load: the
  server remembering it comes later (`docs/FOR_LATER.md`).

### Marks

- **You, on the rectangle:** a blinking gold square with a dark outline, at the middle of your cell's pixel.
- **The players you see** (on the chunks you hold), on the rectangle: small steady cream dots, under your mark.
- **You, on the donut:** your column is the angle around the ring, column 0 at the top, going clockwise. Your row is
  the distance from the center: the outer edge at row 0, the inner edge at row `height / 2` (256), the outer edge
  again at the last row. From above, the top and bottom faces land on the same ring, so the mark's shape tells them
  apart:
  - a **square** within 32 rows (a chunk) of an equator, row 0 or row `height / 2`, counting across the seam;
  - **▲** on the top face, rows 1 to `height / 2 − 1`;
  - **▼** on the bottom face, the other rows.
- Both of your marks blink together: 0.5 s shown, 0.5 s hidden.
- Only you are on the donut.

### The donut

- Pixel art drawn in code, seen from above: a green ring in the world's colors, with darker patches, a colored dark
  outline, and a transparent hole. 40 × 40 art pixels, scaled by `cursorScale` like the icons: 80 px on screen.
- Only on a map that wraps.

### Logs

Debug logs for:
- the map opened and closed, with the explored share;
- the image loaded (size, time) or failed;
- the server rendering an image (size, time), and serving it from memory or as not modified.

### How it is built

- Back:
  - `worldmap/domain/image.go`: the map's picture, as a paletted image (under 30 colors, so a small PNG): region
    shades, borders, rocks, `void`.
  - `worldmap/app`: `Image(slug)`, the PNG and its ETag, rendered once and kept in memory per slug.
  - `worldmap/transport/http`: the `image.png` route, its headers and the 304.
- Client:
  - `config.ts`: the `map` state's controls, and the world map's `openMap`.
  - `input/bindings.ts`: the `map` state and `openMap`.
  - `game/exploration.ts`: the explored chunks, the share and its text.
  - `game/mapMarks.ts`: where the marks go on the rectangle and on the donut, the donut mark's shape, the blink, and
    the rectangle's scale.
  - `game/screens.ts`: opening and closing the map: switching the control state, cancelling the selection.
  - `game/session.ts`: `whereabouts()`, what the map shows of the session: your cell, the players you see, your
    region.
  - `art/donut.ts` and `art/mapMarks.ts`: the donut and the mark sprites.
  - `net/mapApi.ts`: the image's URL.
  - `ui/mapScreen.ts`: the map's block, its veil, its marks, and what's laid over it.
  - `main.ts`: exploration recorded as chunks arrive, and the map wired to the controls.

### Security review

- The image is public, like the map's description: it shows what the description and chunks would. The slug only
  picks a map; an unknown one is a 404.
- The server renders each map at most once per process, one render at a time, so the route can't be used to make it
  render again and again, nor many times at once. Unknown slugs are never kept, so they can't fill the memory. A map
  is at most 4096 × 4096 cells: 16 MB of paletted pixels while rendering, the PNG much smaller.
- The slug reaches the database only as a query parameter.
- Region names are shown as text, never as HTML.
- Known risk: the client gets the whole world's picture, beyond what its player has seen. Strict fog of war will
  have to replace it with what the server remembers each player explored (`docs/FOR_LATER.md`).

### Not now

- Exploration remembered by the server, and fog of war.
- Zooming or panning the map, clicking it to travel, and marks for places.
- Maps of interiors.
- Other players on the donut.

## Test cases

### Back: the image (`worldmap/domain`)

1. The image is the map's size, 1 pixel per cell.
2. Each region has its own shade, and region 0 the grass green.
3. A cell next to another region is a darker border, and across the seam too on a wrapping map. Not across the
   edge of a bounded map.
4. A rock is light gray, even on a border. `void` is transparent.

### Back: serving it (`worldmap/app`, `worldmap/transport/http`)

5. The first request renders the PNG; the next ones are served from memory, with the same ETag. A burst of 8 first
   requests renders once.
6. `GET /api/maps/world/image.png` gives `200`, `image/png`, `nosniff`, `Cache-Control: no-cache`, an ETag and a
   PNG that decodes to 1024 × 512.
7. With a matching `If-None-Match`: `304` and no body. With another ETag: `200`.
8. An unknown slug gives `404`.

### Client: controls (`input/bindings.ts`)

9. On the world map, `m` and `M` open the map, whatever key position prints them.
10. On the map, `m`, `M` and Escape close it, and the world map's keys (Z, Y, Space) do nothing.

### Client: opening and closing (`game/screens.ts`)

11. Opening the map enters the `map` state, cancels the selection and shows the map screen. Closing it hides it
    and enters `worldmap` again.

### Client: explored (`game/exploration.ts`)

12. A chunk counts once, and forgetting it doesn't unexplore it.
13. 9 chunks of 512 give "1.8% explored"; none give "0.0% explored"; all give "100.0% explored".

### Client: marks (`game/mapMarks.ts`)

14. On the rectangle, a cell's mark is at the middle of its pixel, times the scale.
15. On the donut, on a 1024 × 512 map:
    - column 0 is at the top, 256 on the right, 512 at the bottom, 768 on the left;
    - row 0 is on the outer edge, row 256 on the inner edge, row 128 halfway.
16. The donut mark's shape:
    - rows 0, 32, 480, 511, 224, 256 and 288: a square;
    - rows 33, 128 and 223: ▲;
    - rows 289, 384 and 479: ▼.
17. The blink: shown for 0.5 s, hidden for 0.5 s.
18. The scale: ×2, ×1 or ×½, the biggest that fits; ×½ when nothing fits.

### Client: the session (`game/session.ts`)

19. `whereabouts()` gives your cell, the other players you see, and your region; nothing before the welcome.

### Client: art (`art/donut.ts`, `art/mapMarks.ts`)

20. The donut is 40 × 40, with a transparent hole and corners, an opaque ring, and a small palette.
21. The square, ▲ and ▼ marks: ▲ is wider at the bottom, ▼ at the top.

# ADR 0004: Controls per game state, and a free camera on the world map

- Status: accepted, 2026-11-10
- Replaces:
  - from ADR 0003, "Camera": "There's no clamping and no panning", and the fixed zoom;
  - from ADR 0002, "a right click anywhere cancels": only a right click that doesn't drag cancels now.

## Context

The camera is glued to your player at zoom 3. You can't look around, zoom, or check what's beyond the screen.

More game states will come (menus, dialogs, combat), each with its own controls. Games usually group controls per
state: Unity calls them action maps, Unreal input mapping contexts. Only the state on screen reacts to its inputs.

## Decisions

### Controls, per state

- Each game state has its own controls: for each action, the inputs that trigger it. Only the current state's
  controls react.
- There is one state for now, `worldmap`. The tile's action menu is part of it: it's mouse only. It becomes its own
  state when it's driven by the keyboard.
- The controls are data in `config.ts`. There's no rebinding screen, and they aren't saved.
- An input is one of:
  - a key by its **position** (`KeyboardEvent.code`): `KeyZ` is the key printed W on AZERTY;
  - a key by its **printed character** (`KeyboardEvent.key`), used for `+`, `=` and `-`, whose positions vary too
    much between layouts. The numpad's + and − print `+` and `-`, so they match too;
  - the **wheel**, up or down;
  - the **right button**, clicked or dragged.
- A key can need the **modifier**: Cmd on macOS, Ctrl elsewhere. A key without it doesn't match while Cmd or Ctrl is
  held, so Cmd + Z isn't a zoom and the page keeps its shortcut. Shift and Alt are ignored.
- A held key acts once: its repeats do nothing.
- A key bound in the current state wins over the page: its default action is blocked, on key down and key up. Space
  never presses a focused menu button again, and Cmd/Ctrl + and − never zoom the page.
- Keys typed in a text field are left alone (there are none yet: the chat will need it).

### The world map's controls

| Action                    | Inputs                       |
| ------------------------- | ---------------------------- |
| Lock or free the camera   | `KeyY`                       |
| Put it back on you        | `Space`                      |
| Zoom in                   | `KeyZ`, wheel up, mod + `+` or mod + `=` |
| Zoom out                  | `KeyX`, wheel down, mod + `-` |
| Pan                       | right drag                   |
| Cancel                    | right click                  |

A left click still selects a tile, or cancels outside the map, as in ADR 0002.

### The camera

- **Locked:** your player stays in the middle of the screen, as before.
- **Free:** the camera stays where it is while your player walks.
- **Y** switches between them. Freeing leaves the camera where it is. Locking jumps it back onto your player, at once.
- **Space**, free, jumps the camera back onto your player and leaves it free. Locked, it does nothing.
- **Right drag**, free, pans: the world follows the mouse, pixel for pixel. Locked, it does nothing.
  - Under 4 px of movement between press and release, it's a click, and it cancels.
  - A drag never cancels: the selection stays, and the menu follows its tile.
  - A drag never unlocks the camera: only Y does.
- **The leash:** the free camera's center stays within **32 cells** (a chunk) of your player, along both the columns
  and the rows.
  - A drag stops at the limit.
  - If your player walks farther, the camera is pulled along.
  - It's counted in drawn cells, so it holds across the seams.
  - Past about 11 cells at zoom 3, the screen shows beyond the chunks you hold: the background shows there.
- **Zoom:** whole steps, from **2 to 6**, starting at **3**. Past the limits, a step does nothing.
  - Locked, your player stays in the middle.
  - Free, the keys keep the middle of the screen still, and the wheel keeps the point under the pointer still.
    The leash applies after.
- **The wheel:** one zoom step per 100 px of scroll added up, so a trackpad's small scrolls work as well as a mouse's
  notches.
  - Line and page scrolls are counted as 40 px a line and 800 px a page.
  - A change of direction restarts the count.
  - After a step, scrolls are dropped for 150 ms, so a trackpad's momentum doesn't run through every zoom.
  - The page never zooms or scrolls: Cmd/Ctrl + wheel and trackpad pinches zoom the game instead.
- The camera is placed on whole pixels, as before.
- **Hover:** the border shows the tile under the pointer, also when the camera moves under a still mouse.
- **Saved:** only the lock, in `localStorage` under `donut.camera`, for every map. The first time, or when it can't
  be read, the camera starts locked. The zoom starts at 3 on every load, and a free camera starts on your player.

### The camera icon

- A small pixel-art camera in the top-right corner, always shown: with a gold padlock in front of its bottom-right
  corner when locked, cut out from it by a transparent halo; without when free. Both icons have the same size.
- Drawn in code like the finger, and scaled like it (`cursorScale`).
- Clicking it does what Y does.
- A proper design for icons on screen comes later (`docs/FOR_LATER.md`).

### Logs

Debug logs for:
- an action triggered, with its state and input;
- the lock switched and saved, or failing to save;
- the zoom changed, and a step refused at a limit;
- a right drag's start and end, and the leash pulling the camera and letting go.

A drag's moves and the wheel's scrolls aren't logged one by one.

### How it is built

Client only:

- `config.ts`: `controls` (the inputs of each state's actions), and `camera` (zoom range and start, leash, wheel
  step and pause, drag threshold). It replaces `zoom`.
- `core/store.ts`: `KeyValueStore` and `browserStore`, moved from `net/guest.ts` to be shared. Tests share an
  in-memory and a throwing store in `testing/store.ts`.
- `input/bindings.ts`: which action of a state an input triggers, as pure functions.
- `input/wheel.ts`: adding scrolls up into zoom steps.
- `input/drag.ts`: telling a right click from a right drag, and the drag's moves.
- `input/controls.ts`: the DOM listeners (keys and wheel on the window, the right button), routing to the current
  state's actions.
- `game/camera.ts`: the camera's state (locked, zoom, the free center), its actions and the leash, as pure
  functions.
- `game/cameraSettings.ts`: reading and saving the lock.
- `iso/projection.ts`: the fractional tile of a world point, for the leash.
- `ui/pointer.ts`: the hover tracker picks again when the camera moves. The right click leaves it for
  `input/controls.ts`.
- `art/cameraIcon.ts` and `ui/cameraButton.ts`: the icon. `art/drawing.ts` gets `stampRows`, which paints rows over
  a sprite and can erase (the halo), and `ui/cursor.ts` gets `spriteDataUrl`, shared with the cursor.
- `main.ts`: the camera drives the world's position and scale each frame.

### Security review

- Nothing goes to the server. The lock is the only thing saved, and a bad value starts locked.
- The leash keeps the camera within what you hold: the free camera shows nothing the server didn't send.

### Not now

- Rebinding the controls, and saving them.
- Keyboard panning (arrow keys or WASD).
- A smooth camera (sliding back onto the player, smooth zoom).
- Gamepad and touch.
- The two-part skybox behind the world (`docs/FOR_LATER.md`).

## Test cases

### Bindings (`input/bindings.ts`)

1. A key by position: `KeyZ` zooms in whatever it prints (`w` on AZERTY); the key printed `z` at `KeyW` does nothing.
2. A key by character, with the modifier: mod + `+`, mod + `=` and the numpad's mod + `+` zoom in; mod + `-` zooms
   out.
3. The modifier is Cmd on macOS and Ctrl elsewhere:
   - on macOS, Cmd + `+` zooms in and Ctrl + `+` does nothing;
   - elsewhere, Ctrl + `+` zooms in and Cmd + `+` does nothing.
4. A key without the modifier doesn't match while it's held: Cmd + Z does nothing. Shift is ignored: Shift + Y frees
   the camera.
5. An unbound key, or a key bound only in another state, gives nothing.
6. A held key's repeats give nothing.
7. The wheel up zooms in and down zooms out, with or without the modifier (a pinch comes with Ctrl).
8. The right button: a click cancels, a drag pans.

### The wheel (`input/wheel.ts`)

9. One mouse notch (100 px) gives one step, up or down.
10. Ten trackpad scrolls of 10 px give one step.
11. Line scrolls count 40 px a line: 3 lines give one step.
12. Scrolls within 150 ms after a step are dropped, and count again after.
13. A change of direction restarts the count.

### The right button (`input/drag.ts`)

14. Press and release within 4 px: a click.
15. Past 4 px: a drag. Its first move carries everything since the press, then each move since the last one; the
    release gives no click.

### The camera (`game/camera.ts`)

16. Locked, it's on your player, and stays on them as they walk.
17. Y:
    - freeing keeps the camera where it is, and it stays there as your player walks;
    - locking puts it back on your player.
18. Space: free, it puts the camera on your player and it stays free; locked, nothing changes.
19. Pan: free, dragging 30 px right at zoom 3 moves the camera 10 art pixels left; locked, nothing changes.
20. The leash:
    - a pan stops at 32 cells from your player, along the columns and along the rows;
    - your player walking 40 cells away pulls the camera to 32 cells;
    - past the seam (your player drawn at column 1030), it counts in drawn cells.
21. Zoom:
    - steps of 1, from 2 to 6: a step past a limit changes nothing;
    - locked, your player stays in the middle;
    - free, with the keys, the middle of the screen stays still;
    - free, with the wheel, the world point under the pointer stays under it;
    - a zoom under a pointer near the edge can't push the camera past the leash.
22. The world's position puts the camera's center in the middle of the screen, on whole pixels, at the current zoom.

### Saved lock (`game/cameraSettings.ts`)

23. Nothing saved, a broken value, no storage or a storage that throws: locked.
24. A saved free camera is read back free, and switching the lock saves it.

### Hover (`ui/pointer.ts`)

25. The camera moving under a still pointer gives the hover of the new tile under it, and nothing when it's the same
    tile.

### The icon (`art/cameraIcon.ts`)

26. The locked and free icons have the same size and the same camera; only the locked one has the padlock.

import { silent, type Log } from '../debug/log';
import { isClick, moveTo, press, type Press } from '../input/drag';
import type { ScreenPoint, TilePoint } from '../iso/projection';

export interface PointerHandlers {
  /** The tile under the pointer, null outside the map or off the canvas. Called only when it changes. */
  hover(cell: TilePoint | null): void;
  /** A click on a tile. */
  select(cell: TilePoint): void;
  /** A click outside the map. The right click is the controls' (ADR 0004). */
  cancel(): void;
}

/** Canvas point (CSS pixels from its top-left) to the tile there, or null. */
export type PickCell = (point: ScreenPoint) => TilePoint | null;

export interface PointerListener {
  /** Picks the tile under a still pointer again: after the camera moved. */
  repick(): void;
  stop(): void;
}

export interface HoverTracker {
  moveTo(point: ScreenPoint): void;
  leave(): void;
  /** The tile under the last point, picked again: the camera may have moved under it. */
  repick(): void;
}

/** Tells the hovered tile from the pointer's point, and calls `onHover` only when it changes. */
export function createHoverTracker(pick: PickCell, onHover: (cell: TilePoint | null) => void, log: Log = silent): HoverTracker {
  let point: ScreenPoint | null = null;
  let hovered: TilePoint | null = null;
  const hover = (cell: TilePoint | null) => {
    if (cell?.col === hovered?.col && cell?.row === hovered?.row) return;
    hovered = cell;
    log('hover', cell);
    onHover(cell);
  };
  return {
    moveTo(at) {
      point = at;
      hover(pick(at));
    },
    leave() {
      point = null;
      hover(null);
    },
    repick() {
      if (point) hover(pick(point));
    },
  };
}

/** Tells a left click from the end of a left drag, which pans (ADR 0007): the browser sends a click for both. */
export interface ClickGuard {
  down(at: ScreenPoint): void;
  move(at: ScreenPoint): void;
  /** The release at `at` ends a click, not a drag. */
  isClick(at: ScreenPoint): boolean;
}

export function createClickGuard(thresholdPx: number): ClickGuard {
  let held: Press | null = null;
  return {
    down: (at) => (held = press(at)),
    move: (at) => {
      if (held) held = moveTo(held, at, thresholdPx).held;
    },
    isClick: (at) => {
      const click = held === null || isClick(held, at, thresholdPx);
      held = null;
      return click;
    },
  };
}

/**
 * Turns the mouse on the canvas into tile hovers, selections and cancels. A left drag pans: its release is no click.
 */
export function listenToPointer(
  canvas: HTMLElement,
  pick: PickCell,
  handlers: PointerHandlers,
  dragThresholdPx: number,
  log: Log = silent,
): PointerListener {
  const tracker = createHoverTracker(pick, handlers.hover, log);
  const guard = createClickGuard(dragThresholdPx);
  const pointAt = (event: MouseEvent) => {
    const rect = canvas.getBoundingClientRect();
    return { x: event.clientX - rect.left, y: event.clientY - rect.top };
  };
  const onDown = (event: PointerEvent) => {
    if (event.button === 0) guard.down(pointAt(event));
  };
  const onMove = (event: PointerEvent) => {
    tracker.moveTo(pointAt(event));
    if (event.buttons & 1) guard.move(pointAt(event));
  };
  const onLeave = () => tracker.leave();
  const onClick = (event: MouseEvent) => {
    if (event.button !== 0) return;
    const point = pointAt(event);
    if (!guard.isClick(point)) return void log('left drag released: no click', { point });
    const cell = pick(point);
    log('click', { point, cell });
    if (cell) handlers.select(cell);
    else handlers.cancel();
  };

  canvas.addEventListener('pointerdown', onDown);
  canvas.addEventListener('pointermove', onMove);
  canvas.addEventListener('pointerleave', onLeave);
  canvas.addEventListener('click', onClick);
  return {
    repick: () => tracker.repick(),
    stop() {
      canvas.removeEventListener('pointerdown', onDown);
      canvas.removeEventListener('pointermove', onMove);
      canvas.removeEventListener('pointerleave', onLeave);
      canvas.removeEventListener('click', onClick);
    },
  };
}

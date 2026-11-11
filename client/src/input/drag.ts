import type { ScreenPoint } from '../iso/projection';

/** A button held down: where it was pressed, where it was last, and whether it moved enough to be a drag. */
export interface Press {
  start: ScreenPoint;
  last: ScreenPoint;
  dragging: boolean;
}

/** Pressing the button. */
export function press(at: ScreenPoint): Press {
  return { start: at, last: at, dragging: false };
}

/**
 * The pointer moved with the button held. Once it's `threshold` px or more from where it was pressed, it's a drag,
 * and each move gives how far it went: the first one since the press, the next ones since the last move.
 */
export function moveTo(held: Press, at: ScreenPoint, threshold: number): { held: Press; delta: ScreenPoint | null } {
  if (!held.dragging && distance(held.start, at) < threshold) return { held, delta: null };
  const delta = { x: at.x - held.last.x, y: at.y - held.last.y };
  return { held: { ...held, last: at, dragging: true }, delta };
}

/** Releasing the button: a click when it never became a drag and is released within `threshold` px of the press. */
export function isClick(held: Press, at: ScreenPoint, threshold: number): boolean {
  return !held.dragging && distance(held.start, at) < threshold;
}

function distance(a: ScreenPoint, b: ScreenPoint): number {
  return Math.hypot(b.x - a.x, b.y - a.y);
}

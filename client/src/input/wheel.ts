import type { WheelDirection } from './bindings';

export interface WheelConfig {
  /** Scroll, in pixels added up, that makes one step. */
  stepPx: number;
  /** After a step, scrolls are dropped for this long, so a trackpad's momentum doesn't run through every step. */
  pauseMs: number;
}

/** The scroll added up since the last step, and until when scrolls are dropped. */
export interface WheelState {
  sum: number;
  pausedUntil: number;
}

/** WheelEvent.deltaMode: 0 pixels, 1 lines, 2 pages. */
const PX_PER_MODE = [1, 40, 800] as const;

export const initialWheel: WheelState = { sum: 0, pausedUntil: -Infinity };

/** A wheel event's vertical scroll in pixels: lines count 40 px, pages 800. Negative is up. */
export function scrollPx(deltaY: number, deltaMode: number): number {
  return deltaY * (PX_PER_MODE[deltaMode as 0 | 1 | 2] ?? 1);
}

/**
 * Adds a scroll up: one step each time the sum reaches `stepPx`, up or down. A change of direction restarts the
 * count.
 */
export function addScroll(
  state: WheelState,
  px: number,
  now: number,
  config: WheelConfig,
): { state: WheelState; step: WheelDirection | null } {
  if (now < state.pausedUntil || px === 0) return { state, step: null };
  const sameWay = Math.sign(px) === Math.sign(state.sum);
  const sum = sameWay ? state.sum + px : px;
  if (Math.abs(sum) < config.stepPx) return { state: { ...state, sum }, step: null };
  return { state: { sum: 0, pausedUntil: now + config.pauseMs }, step: sum < 0 ? 'up' : 'down' };
}

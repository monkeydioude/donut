import type { IsoProjection, ScreenPoint, TilePoint } from '../iso/projection';
import { nearestCopy, type Grid } from '../map/grid';

/** A sprite going from one world position to another in a straight line, at a fixed pace. */
export interface Glide {
  from: ScreenPoint;
  to: ScreenPoint;
  startedAt: number;
  durationMs: number;
}

/** 'glide' for a step, 'place' to show the sprite on its cell at once (arriving, coming back, stopped). */
export type Arrival = 'glide' | 'place';

/** Where a glide is at `now`: on its way, or at its end once its time is over. */
export function glideAt(glide: Glide, now: number): ScreenPoint {
  const t = glide.durationMs <= 0 ? 1 : Math.min(1, Math.max(0, (now - glide.startedAt) / glide.durationMs));
  return { x: glide.from.x + (glide.to.x - glide.from.x) * t, y: glide.from.y + (glide.to.y - glide.from.y) * t };
}

/** The glide toward a new position. It starts from where the sprite is shown, mid-glide or not. */
export function nextGlide(previous: Glide | undefined, to: ScreenPoint, arrival: Arrival, now: number, durationMs: number): Glide {
  if (!previous || arrival === 'place') return { from: to, to, startedAt: now, durationMs: 0 };
  return { from: glideAt(previous, now), to, startedAt: now, durationMs };
}

/** A player sprite's glide and draw order, going to `cell`. The draw order is the target cell's, at once. */
export function placePlayer(
  previous: Glide | undefined,
  cell: TilePoint,
  arrival: Arrival,
  now: number,
  stepMs: number,
  projection: IsoProjection,
): { glide: Glide; zIndex: number } {
  return { glide: nextGlide(previous, projection.toScreen(cell), arrival, now, stepMs), zIndex: projection.depth(cell) };
}

/**
 * The drawn cell a player goes to, in coordinates that don't wrap: one step from where they are drawn when gliding,
 * so a step across the seam glides one cell; else the copy nearest to your drawn cell (the anchor).
 */
export function drawnTarget(
  grid: Grid,
  cell: TilePoint,
  drawnFrom: TilePoint | undefined,
  arrival: Arrival,
  anchor: TilePoint | null,
): TilePoint {
  const near = arrival === 'glide' && drawnFrom ? drawnFrom : (anchor ?? drawnFrom ?? cell);
  return nearestCopy(grid, cell, near);
}

/** The same glide, moved by a whole number of worlds: a player redrawn at another copy. */
export function shiftGlide(glide: Glide, by: ScreenPoint): Glide {
  return { ...glide, from: { x: glide.from.x + by.x, y: glide.from.y + by.y }, to: { x: glide.to.x + by.x, y: glide.to.y + by.y } };
}

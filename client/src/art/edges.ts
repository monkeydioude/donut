import type { TilePoint } from '../iso/projection';
import { groundCenter, isOnGround, type TileFrame } from './tileFrame';

/**
 * A side of a cell's ground diamond, by the neighbor across it: `se` is the next column (lower right on screen), `sw`
 * the next row (lower left), `nw` the column before (upper left) and `ne` the row before (upper right).
 */
export type Edge = 'ne' | 'nw' | 'se' | 'sw';

export const EDGES: readonly Edge[] = ['ne', 'nw', 'se', 'sw'];

/** The neighbor across each edge, as a column and row offset. */
export const EDGE_STEPS: Readonly<Record<Edge, TilePoint>> = {
  se: { col: 1, row: 0 },
  sw: { col: 0, row: 1 },
  nw: { col: -1, row: 0 },
  ne: { col: 0, row: -1 },
};

/**
 * A corner of a cell's ground diamond, by the diagonal neighbor beyond it: `n` is the top corner, toward the cell one
 * column and one row before; `e` the right one, `s` the bottom one, `w` the left one.
 */
export type Corner = 'n' | 'e' | 's' | 'w';

export const CORNERS: readonly Corner[] = ['n', 'e', 's', 'w'];

/** The diagonal neighbor beyond each corner, as a column and row offset. */
export const CORNER_STEPS: Readonly<Record<Corner, TilePoint>> = {
  n: { col: -1, row: -1 },
  e: { col: 1, row: -1 },
  s: { col: 1, row: 1 },
  w: { col: -1, row: 1 },
};

/** The two edges meeting at each corner. */
const CORNER_EDGES: Readonly<Record<Corner, readonly [Edge, Edge]>> = { n: ['nw', 'ne'], e: ['ne', 'se'], s: ['se', 'sw'], w: ['sw', 'nw'] };

/** Each edge's line is (su · u) + (sv · v) = 1, u and v the pixel's place from the ground center, in half tiles. */
const EDGE_SIGNS: Readonly<Record<Edge, { su: number; sv: number }>> = {
  ne: { su: 1, sv: -1 },
  nw: { su: -1, sv: -1 },
  se: { su: 1, sv: 1 },
  sw: { su: -1, sv: 1 },
};

/** A set of edges or corners as a short key, the same in any order: what is painted for them is cached by it. */
export function edgesKey(edges: readonly (Edge | Corner)[]): string {
  return [...EDGES, ...CORNERS].filter((edge) => edges.includes(edge)).join(',');
}

/**
 * How many pixels a texture pixel is inside the ground diamond from one of its edges, measured up or down: 0 to 1 on
 * the edge itself. Infinity off the ground, or in the quarter of another edge.
 */
export function depthFromEdge(frame: TileFrame, edge: Edge, x: number, y: number): number {
  if (!isOnGround(frame, x, y)) return Infinity;
  const center = groundCenter(frame);
  const halfHeight = frame.tileHeight / 2;
  const u = (x + 0.5 - center.x) / (frame.tileWidth / 2);
  const v = (y + 0.5 - center.y) / halfHeight;
  const inQuarter = { se: u >= 0 && v >= 0, sw: u <= 0 && v >= 0, nw: u <= 0 && v <= 0, ne: u >= 0 && v <= 0 }[edge];
  return inQuarter ? (1 - Math.abs(u) - Math.abs(v)) * halfHeight : Infinity;
}

/** How close a pixel must be to both edges' lines to be a corner's: the side corners are 2 pixels tall. */
const CORNER_PX = 1.5;

/** True for the 2 pixels of the ground diamond at a corner: on the ground, close to the lines of both its edges. */
export function isAtCorner(frame: TileFrame, corner: Corner, x: number, y: number): boolean {
  if (!isOnGround(frame, x, y)) return false;
  const { u, v } = diamondPlace(frame, x, y);
  return CORNER_EDGES[corner].every((edge) => (1 - EDGE_SIGNS[edge].su * u - EDGE_SIGNS[edge].sv * v) * (frame.tileHeight / 2) < CORNER_PX);
}

function diamondPlace(frame: TileFrame, x: number, y: number): { u: number; v: number } {
  const center = groundCenter(frame);
  return { u: (x + 0.5 - center.x) / (frame.tileWidth / 2), v: (y + 0.5 - center.y) / (frame.tileHeight / 2) };
}

/** Where a texture pixel is in its cell, in fractions of a cell from its center: along the columns and the rows. */
export function cellFraction(frame: TileFrame, x: number, y: number): TilePoint {
  const center = groundCenter(frame);
  const u = (x + 0.5 - center.x) / (frame.tileWidth / 2);
  const v = (y + 0.5 - center.y) / (frame.tileHeight / 2);
  return { col: (u + v) / 2, row: (v - u) / 2 };
}

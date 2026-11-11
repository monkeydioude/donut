import type { TilePoint } from '../iso/projection';
import type { FaceSide } from './cliff';
import type { Edge } from './edges';
import { forEachPixel } from './pixelBuffer';
import { groundCenter, type TileFrame } from './tileFrame';

/**
 * How a mountain's or glacier's top rises over its cell's height (ADR 0011), in levels: its back corner `n` and its
 * side corners `w` and `e` rise to the highest cell of its kind touching them behind or beside; the front corner
 * stays. A peak raises the top's middle into a point.
 */
export interface BlockShape {
  n: number;
  w: number;
  e: number;
  peak: boolean;
}

export const FLAT_BLOCK: BlockShape = { n: 0, w: 0, e: 0, peak: false };

/** How high a peak's point rises over its cell, in levels. */
export const PEAK_LEVELS = 1.5;

/**
 * What a pixel of a block shows: its top, at a point of the cell (from −0.5 to 0.5 along the columns and the rows,
 * 0 at its middle), lit from 0 (dark) to 1; or one of its front sides, at a point of its front edge.
 */
export type BlockPixel = { part: 'top'; at: TilePoint; light: number } | { part: 'side'; side: FaceSide; at: TilePoint };

/**
 * The frame a block needs: room above its ground diamond's top for its highest point. The back corner is on the
 * diamond's top row; the side corners and the middle are half a tile lower, so they need that much less.
 */
export function blockFrame(tileWidth: number, tileHeight: number, shape: BlockShape, levelPx: number): TileFrame {
  const half = tileHeight / 2;
  const over = [shape.n * levelPx, shape.w * levelPx - half, shape.e * levelPx - half, shape.peak ? PEAK_LEVELS * levelPx - half : 0];
  return { tileWidth, tileHeight, headroom: Math.ceil(Math.max(0, ...over)) };
}

/** A short key for a shape: what a block's sprite is cached by, with its other looks. */
export function shapeKey(shape: BlockShape): string {
  return `${shape.n},${shape.w},${shape.e}${shape.peak ? ' peak' : ''}`;
}

/**
 * Visits every pixel of a cell's block, in its block frame. The top is 4 triangles from the middle to each edge,
 * through its corners' heights and its middle's (the corners' average, or the peak's point), so a lower block in
 * front leans up to the higher one behind, and a peak is a pyramid. Under the top's front edges, down to the ground
 * diamond's, the block's sides.
 */
export function forEachBlockPixel(frame: TileFrame, shape: BlockShape, levelPx: number, visit: (x: number, y: number, pixel: BlockPixel) => void): void {
  const geometry = blockGeometry(frame, shape, levelPx);
  forEachPixel(frame.tileWidth, frame.tileHeight + frame.headroom, (x, y) => {
    const pixel = geometry.pixelAt(x, y);
    if (pixel) visit(x, y, pixel);
  });
}

/** A triangle of the top as a plane: its rise over the cell is middle + p·col + q·row, in pixels. */
interface Plane {
  middle: number;
  p: number;
  q: number;
  light: number;
}

type Corner4 = 'n' | 'e' | 's' | 'w';
const CORNER_AT: Readonly<Record<Corner4, TilePoint>> = {
  n: { col: -0.5, row: -0.5 },
  e: { col: 0.5, row: -0.5 },
  s: { col: 0.5, row: 0.5 },
  w: { col: -0.5, row: 0.5 },
};
/** The top's triangles, by the edge each one holds, with that edge's two corners. */
const TRIANGLES: readonly { edge: Edge; corners: readonly [Corner4, Corner4] }[] = [
  { edge: 'ne', corners: ['n', 'e'] },
  { edge: 'se', corners: ['e', 's'] },
  { edge: 'sw', corners: ['s', 'w'] },
  { edge: 'nw', corners: ['w', 'n'] },
];

/**
 * A pixel at d = col − row across the cell (−1 at its left corner, 1 at its right) sees, along its line of sight, the
 * points s = col + row from the back to the front, each higher the closer it is. The frontmost point inside the block
 * is what it shows: on a front side when the line passes under the top's front edge, else on the top.
 */
function blockGeometry(frame: TileFrame, shape: BlockShape, levelPx: number) {
  const center = groundCenter(frame);
  const halfWidth = frame.tileWidth / 2;
  const halfHeight = frame.tileHeight / 2;
  const rises: Record<Corner4, number> = { n: shape.n * levelPx, e: shape.e * levelPx, s: 0, w: shape.w * levelPx };
  const middle = shape.peak ? PEAK_LEVELS * levelPx : (rises.n + rises.e + rises.s + rises.w) / 4;
  const planes = TRIANGLES.map(({ edge, corners }) => ({ edge, plane: planeThrough(middle, corners.map((c) => ({ at: CORNER_AT[c], rise: rises[c] })), frame) }));
  const planeAt = (at: TilePoint) => (planes.find(({ edge }) => inTriangle(edge, at)) ?? planes[0])!.plane;
  const riseAt = (at: TilePoint) => {
    const plane = planeAt(at);
    return plane.middle + plane.p * at.col + plane.q * at.row;
  };
  return {
    pixelAt(x: number, y: number): BlockPixel | null {
      const d = (x + 0.5 - center.x) / halfWidth;
      const dy = y + 0.5 - center.y;
      if (Math.abs(d) > 1) return null;
      const reach = 1 - Math.abs(d);
      const front = place(reach, d);
      const zFront = reach * halfHeight - dy;
      if (zFront < 0) return null;
      if (zFront < riseAt(front)) return { part: 'side', side: d < 0 ? 'left' : 'right', at: front };
      const hit = frontmostHit(planes, d, dy, reach, halfHeight);
      return hit && { part: 'top', at: hit.at, light: hit.light };
    },
  };
}

/** The frontmost point of the top on a pixel's line of sight: each triangle's plane met, kept if inside it. */
function frontmostHit(planes: readonly { edge: Edge; plane: Plane }[], d: number, dy: number, reach: number, halfHeight: number) {
  let best: { s: number; at: TilePoint; light: number } | null = null;
  for (const { edge, plane } of planes) {
    // along the line, the plane's rise is middle + (p − q)·d/2 + (p + q)·s/2, and the line's is s·halfHeight − dy
    const slope = halfHeight - (plane.p + plane.q) / 2;
    if (slope <= 0) continue;
    const s = (dy + plane.middle + ((plane.p - plane.q) * d) / 2) / slope;
    const at = place(s, d);
    if (s < -reach - 1e-9 || s > reach + 1e-9 || !inTriangle(edge, at)) continue;
    if (!best || s > best.s) best = { s, at, light: plane.light };
  }
  return best;
}

/** The plane through the top's middle and two corners, and how much light it gets. */
function planeThrough(middle: number, corners: readonly { at: TilePoint; rise: number }[], frame: TileFrame): Plane {
  const [a, b] = corners as [{ at: TilePoint; rise: number }, { at: TilePoint; rise: number }];
  // solve p·a.col + q·a.row = a.rise − middle and the same for b
  const det = a.at.col * b.at.row - a.at.row * b.at.col;
  const ra = a.rise - middle;
  const rb = b.rise - middle;
  const p = (ra * b.at.row - rb * a.at.row) / det;
  const q = (a.at.col * rb - b.at.col * ra) / det;
  return { middle, p, q, light: lightOf(p, q, frame) };
}

/**
 * The light on a plane rising p pixels per column and q per row: from the left of the screen (the cell's left corner)
 * and above, like the faces (ADR 0007). A flat top gets about 0.75; a slope facing the left more, the right less.
 */
function lightOf(p: number, q: number, frame: TileFrame): number {
  // a cell's side is about 0.7 tile widths long on the ground, and a screen pixel up is about 1.15 pixels of height
  const run = frame.tileWidth * 0.707 * 0.866;
  const normal = normalize([-p / run, -q / run, 1]);
  const light = normalize([-1, 1, 1.6]);
  return Math.max(0, normal[0] * light[0] + normal[1] * light[1] + normal[2] * light[2]);
}

function normalize(v: readonly [number, number, number]): [number, number, number] {
  const length = Math.hypot(v[0], v[1], v[2]);
  return [v[0] / length, v[1] / length, v[2] / length];
}

/** Whether a point of the cell is in the triangle from its middle to an edge. */
function inTriangle(edge: Edge, at: TilePoint): boolean {
  const eps = 1e-9;
  switch (edge) {
    case 'ne':
      return -at.row + eps >= Math.abs(at.col);
    case 'se':
      return at.col + eps >= Math.abs(at.row);
    case 'sw':
      return at.row + eps >= Math.abs(at.col);
    case 'nw':
      return -at.col + eps >= Math.abs(at.row);
  }
}

/** The point of the cell at s = col + row and d = col − row. */
function place(s: number, d: number): TilePoint {
  return { col: (s + d) / 2, row: (s - d) / 2 };
}

/** How close to an edge, in cells, a top's outline is: about a pixel on a flat top. */
const EDGE_LINE = 1 / 16;

/** True for a point of a top near one of the edges given: within about a pixel of it. */
export function isOnEdge(at: TilePoint, edges: readonly Edge[]): boolean {
  const from: Readonly<Record<Edge, number>> = { se: 0.5 - at.col, nw: at.col + 0.5, sw: 0.5 - at.row, ne: at.row + 0.5 };
  return edges.some((edge) => from[edge] < EDGE_LINE);
}

/** One of a palette's shades for a light from 0 to 1: a flat top gets the second lightest. */
export function shadeIndex(light: number, shades: number): number {
  const bands = [0.5, 0.68, 0.82];
  const index = bands.filter((limit) => light >= limit).length;
  return Math.min(shades - 1, index);
}

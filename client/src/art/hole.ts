import { depthFromEdge, type Edge } from './edges';
import { createPixelBuffer, forEachPixel, setPixel, type PixelBuffer } from './pixelBuffer';
import { isOnGround, textureSize, type TileFrame } from './tileFrame';

export const HOLE_COLORS = {
  rim: 0x3f3550,
  /** The inner wall, from its top down: earth fading to near black. */
  walls: [0x9b7655, 0x76583f, 0x52402f, 0x3a2e2a],
  pit: 0x221c30,
} as const;

/** How many pixels each shade of the inner wall takes. */
const WALL_SHADE_PX = 1.5;

/**
 * A hole: a dark pit with no bottom in sight. Under each back edge (ne, nw) toward a cell that isn't a hole, its inner
 * wall shows, earth fading to near black; every edge toward a cell that isn't a hole has a thin darker rim.
 */
export function paintHole(frame: TileFrame, rim: readonly Edge[]): PixelBuffer {
  const { width, height } = textureSize(frame);
  const buffer = createPixelBuffer(width, height);
  forEachPixel(width, height, (x, y) => {
    if (isOnGround(frame, x, y)) setPixel(buffer, x, y, holeColor(frame, x, y, rim));
  });
  return buffer;
}

function holeColor(frame: TileFrame, x: number, y: number, rim: readonly Edge[]): number {
  if (rim.some((edge) => depthFromEdge(frame, edge, x, y) < 1)) return HOLE_COLORS.rim;
  for (const edge of rim) {
    if (edge !== 'ne' && edge !== 'nw') continue;
    const shade = Math.floor((depthFromEdge(frame, edge, x, y) - 1) / WALL_SHADE_PX);
    const wall = HOLE_COLORS.walls[shade];
    if (wall !== undefined) return wall;
  }
  return HOLE_COLORS.pit;
}

import { cellFraction, depthFromEdge, type Edge } from './edges';
import { createPixelBuffer, forEachPixel, setPixel, type PixelBuffer } from './pixelBuffer';
import { isOnGround, textureSize, type TileFrame } from './tileFrame';

export const OCEAN_COLORS = { base: 0x4f9fd8, ripple: 0x66b1e2, foam: 0xe8f6fb } as const;

/** Short still ripples across a cell: where they are along the rows, how far along the columns they start. */
const RIPPLE_LANES = [
  { row: -0.28, offset: 0.1 },
  { row: 0.06, offset: 0.55 },
  { row: 0.34, offset: 0.3 },
] as const;
const RIPPLE = { halfThickness: 0.035, length: 0.2 } as const;
const FOAM_PX = 1;

/**
 * An ocean tile (ADR 0008): a deeper, still blue with a few lighter ripples, the same on every cell so neighbors
 * match, and a foam line on the edges given, toward cells that aren't ocean: the coast, a river's mouth.
 */
export function paintOcean(frame: TileFrame, foam: readonly Edge[]): PixelBuffer {
  const { width, height } = textureSize(frame);
  const buffer = createPixelBuffer(width, height);
  forEachPixel(width, height, (x, y) => {
    if (isOnGround(frame, x, y)) setPixel(buffer, x, y, oceanColor(frame, x, y, foam));
  });
  return buffer;
}

function oceanColor(frame: TileFrame, x: number, y: number, foam: readonly Edge[]): number {
  if (foam.some((edge) => depthFromEdge(frame, edge, x, y) < FOAM_PX)) return OCEAN_COLORS.foam;
  return isRipple(cellFraction(frame, x, y)) ? OCEAN_COLORS.ripple : OCEAN_COLORS.base;
}

function isRipple(at: { col: number; row: number }): boolean {
  return RIPPLE_LANES.some((lane) => Math.abs(at.row - lane.row) < RIPPLE.halfThickness && fraction(at.col - lane.offset) < RIPPLE.length);
}

function fraction(value: number): number {
  return value - Math.floor(value);
}

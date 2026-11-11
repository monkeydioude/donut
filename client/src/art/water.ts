import { cellFraction, depthFromEdge, type Edge } from './edges';
import { createPixelBuffer, forEachPixel, setPixel, type PixelBuffer } from './pixelBuffer';
import { isOnGround, textureSize, type TileFrame } from './tileFrame';

export const WATER_COLORS = { base: 0x7cc4e8, shadow: 0x66b0dc, streak: 0xd2eef9, foam: 0xf2fbfd } as const;

/** The flow's loop: this many frames, the streaks moving a quarter of a cell east each (ADR 0007). */
export const WATER_FRAMES = 4;

/** Lines of streaks across a cell: where they are along the rows, how far along the columns they start. */
const STREAK_LANES = [
  { row: -0.3, offset: 0 },
  { row: 0.02, offset: 0.45 },
  { row: 0.28, offset: 0.2 },
] as const;
const STREAK = { halfThickness: 0.035, length: 0.28 } as const;
const FOAM_PX = 1;
/** Under a back edge toward a bank, the water is in the bank's shadow for this many pixels past the foam. */
const SHADOW_PX = 2;

/**
 * A water tile at one frame of its flow. Light streaks run along the columns (east), a quarter of a cell further each
 * frame: the pattern repeats every cell, so neighbors match and the loop is seamless. Foam lines the edges given,
 * toward cells that aren't water, and the back ones (ne, nw) cast a little shadow on the water.
 */
export function paintWater(frame: TileFrame, flowFrame: number, foam: readonly Edge[]): PixelBuffer {
  const { width, height } = textureSize(frame);
  const buffer = createPixelBuffer(width, height);
  const shift = flowFrame / WATER_FRAMES;
  forEachPixel(width, height, (x, y) => {
    if (!isOnGround(frame, x, y)) return;
    setPixel(buffer, x, y, waterColor(frame, x, y, shift, foam));
  });
  return buffer;
}

function waterColor(frame: TileFrame, x: number, y: number, shift: number, foam: readonly Edge[]): number {
  const depths = foam.map((edge) => ({ edge, depth: depthFromEdge(frame, edge, x, y) }));
  if (depths.some(({ depth }) => depth < FOAM_PX)) return WATER_COLORS.foam;
  if (depths.some(({ edge, depth }) => (edge === 'ne' || edge === 'nw') && depth < FOAM_PX + SHADOW_PX)) return WATER_COLORS.shadow;
  return isStreak(cellFraction(frame, x, y), shift) ? WATER_COLORS.streak : WATER_COLORS.base;
}

function isStreak(at: { col: number; row: number }, shift: number): boolean {
  return STREAK_LANES.some(
    (lane) => Math.abs(at.row - lane.row) < STREAK.halfThickness && fraction(at.col - lane.offset - shift) < STREAK.length,
  );
}

function fraction(value: number): number {
  return value - Math.floor(value);
}

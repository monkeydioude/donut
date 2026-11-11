import { cellFraction, depthFromEdge, type Edge } from './edges';
import { createPixelBuffer, forEachPixel, setPixel, type PixelBuffer } from './pixelBuffer';
import { groundCenter, isOnGround, textureSize, type TileFrame } from './tileFrame';

export const MOUNTAIN_COLORS = { top: 0x8a8694, crack: 0x6b6775, outline: 0x4f4a57, peakLit: 0xaca8b4, peakDark: 0x736f7d, snow: 0xf2f4f7 } as const;

/** How high the peak rises over the ground diamond's top, as a share of the tile's width: the frame's headroom. */
const PEAK_SHARE = 0.2;
/** How many rows of the peak, from its apex, are snow. */
const SNOW_PX = 3;
/** The peak's base, on the ground center's row, spans this share of the tile's width. */
const PEAK_BASE = 0.5;
/** Short cracks across the top: where they are along the rows, how far along the columns they start. */
const CRACK_LANES = [
  { row: -0.3, offset: 0.2 },
  { row: 0.15, offset: 0.6 },
] as const;
const CRACK = { halfThickness: 0.035, length: 0.3 } as const;

/** The headroom a mountain's frame needs for its peak. */
export function peakHeadroom(tileWidth: number): number {
  return Math.max(2, Math.round(tileWidth * PEAK_SHARE));
}

/**
 * A mountain tile (ADR 0009): a gray stone top with darker cracks, the same on every cell so a pack reads as one
 * massif, a dark outline on the edges given, toward cells that aren't mountain, and a peak rising from the cell's
 * middle to the top of the frame's headroom: a triangle, lit on its left, dark on its right, snow on top.
 */
export function paintMountain(frame: TileFrame, outline: readonly Edge[]): PixelBuffer {
  const { width, height } = textureSize(frame);
  const buffer = createPixelBuffer(width, height);
  forEachPixel(width, height, (x, y) => {
    if (isOnGround(frame, x, y)) setPixel(buffer, x, y, topColor(frame, x, y, outline));
  });
  forEachPixel(width, height, (x, y) => {
    const peak = peakColor(frame, x, y);
    if (peak !== null) setPixel(buffer, x, y, peak);
  });
  return buffer;
}

function topColor(frame: TileFrame, x: number, y: number, outline: readonly Edge[]): number {
  if (outline.some((edge) => depthFromEdge(frame, edge, x, y) < 1)) return MOUNTAIN_COLORS.outline;
  return isCrack(cellFraction(frame, x, y)) ? MOUNTAIN_COLORS.crack : MOUNTAIN_COLORS.top;
}

function isCrack(at: { col: number; row: number }): boolean {
  return CRACK_LANES.some((lane) => Math.abs(at.row - lane.row) < CRACK.halfThickness && fraction(at.col - lane.offset) < CRACK.length);
}

function fraction(value: number): number {
  return value - Math.floor(value);
}

/**
 * The peak's color at a pixel, null off it. The peak is a triangle from the texture's top row, 2 pixels wide there,
 * down to the ground center's row, where it spans PEAK_BASE of the tile.
 */
function peakColor(frame: TileFrame, x: number, y: number): number | null {
  const center = groundCenter(frame);
  if (y + 0.5 > center.y) return null;
  const halfWidth = 1 + ((frame.tileWidth * PEAK_BASE) / 2 - 1) * ((y + 0.5) / center.y);
  const dx = x + 0.5 - center.x;
  if (Math.abs(dx) > halfWidth) return null;
  if (y < SNOW_PX) return MOUNTAIN_COLORS.snow;
  return dx < 0 ? MOUNTAIN_COLORS.peakLit : MOUNTAIN_COLORS.peakDark;
}

import { cellFraction, depthFromEdge, type Edge } from './edges';
import { createPixelBuffer, forEachPixel, setPixel, type PixelBuffer } from './pixelBuffer';
import { groundCenter, isOnGround, textureSize, type TileFrame } from './tileFrame';

export const GLACIER_COLORS = { top: 0xdff4fb, crack: 0xb4e2f4, rim: 0xffffff, sideLit: 0xa6d8ee, sideDark: 0x86c3e3 } as const;

/** How high the block's top is raised over the ground diamond, as a share of the tile's width: the frame's headroom. */
const ICE_SHARE = 0.1;
/** Light crack lines across the top: where they are along the rows, how far along the columns they start. */
const CRACK_LANES = [
  { row: -0.22, offset: 0.5 },
  { row: 0.2, offset: 0.1 },
] as const;
const CRACK = { halfThickness: 0.035, length: 0.35 } as const;

/** The headroom a glacier's frame needs for its raised top. */
export function iceHeadroom(tileWidth: number): number {
  return Math.max(2, Math.round(tileWidth * ICE_SHARE));
}

/**
 * A glacier tile (ADR 0009): an ice block. Its top is the ground diamond raised by the frame's headroom, pale ice with
 * lighter cracks, the same on every cell, and a white rim on the edges given, toward cells that aren't glacier. Under
 * the top's front edges, down to the ground diamond's, the block's sides: a deeper blue, darker on the right.
 */
export function paintGlacier(frame: TileFrame, rim: readonly Edge[]): PixelBuffer {
  const { width, height } = textureSize(frame);
  const buffer = createPixelBuffer(width, height);
  forEachPixel(width, height, (x, y) => {
    const color = glacierColor(frame, x, y, rim);
    if (color !== null) setPixel(buffer, x, y, color);
  });
  return buffer;
}

/** The top where the ground diamond is, `headroom` pixels up; the sides where the ground diamond shows under it. */
function glacierColor(frame: TileFrame, x: number, y: number, rim: readonly Edge[]): number | null {
  const rise = frame.headroom;
  if (isOnGround(frame, x, y + rise)) return topColor(frame, x, y + rise, rim);
  if (!isOnGround(frame, x, y)) return null;
  return x + 0.5 < groundCenter(frame).x ? GLACIER_COLORS.sideLit : GLACIER_COLORS.sideDark;
}

/** The top's color at its pixel, given where that pixel would be on the ground diamond. */
function topColor(frame: TileFrame, x: number, onGroundY: number, rim: readonly Edge[]): number {
  if (rim.some((edge) => depthFromEdge(frame, edge, x, onGroundY) < 1)) return GLACIER_COLORS.rim;
  return isCrack(cellFraction(frame, x, onGroundY)) ? GLACIER_COLORS.crack : GLACIER_COLORS.top;
}

function isCrack(at: { col: number; row: number }): boolean {
  return CRACK_LANES.some((lane) => Math.abs(at.row - lane.row) < CRACK.halfThickness && fraction(at.col - lane.offset) < CRACK.length);
}

function fraction(value: number): number {
  return value - Math.floor(value);
}

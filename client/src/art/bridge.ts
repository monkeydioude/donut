import { cellFraction, depthFromEdge, type Edge } from './edges';
import { createPixelBuffer, forEachPixel, setPixel, type PixelBuffer } from './pixelBuffer';
import { isOnGround, textureSize, type TileFrame } from './tileFrame';

export const BRIDGE_COLORS = { plank: 0xc8996a, seam: 0x8e6a45, rail: 0x7a5537, railTop: 0xe0b582 } as const;

/** How many planks a cell holds, across the rows. */
const PLANKS_PER_CELL = 4;
/** How high a rail stands over the deck's edge, in pixels: the frame needs this much headroom. */
export const RAIL_PX = 3;

/**
 * A bridge's deck: planks laid along the columns, across the river, a seam between each. A low rail stands on each of
 * its side edges (nw, se) toward a cell that isn't bridge: the river. Its frame needs RAIL_PX of headroom.
 */
export function paintBridge(frame: TileFrame, rails: readonly Edge[]): PixelBuffer {
  const { width, height } = textureSize(frame);
  const buffer = createPixelBuffer(width, height);
  const sides = rails.filter((edge) => edge === 'nw' || edge === 'se');
  forEachPixel(width, height, (x, y) => {
    if (isOnGround(frame, x, y)) setPixel(buffer, x, y, isSeam(frame, x, y) ? BRIDGE_COLORS.seam : BRIDGE_COLORS.plank);
  });
  forEachPixel(width, height, (x, y) => {
    const rise = railRise(frame, sides, x, y);
    if (rise > 0) setPixel(buffer, x, y, rise === RAIL_PX ? BRIDGE_COLORS.railTop : BRIDGE_COLORS.rail);
  });
  return buffer;
}

function isSeam(frame: TileFrame, x: number, y: number): boolean {
  const across = (cellFraction(frame, x, y).row + 0.5) * PLANKS_PER_CELL;
  return across - Math.floor(across) < 0.12;
}

/** How high over a side edge a pixel stands, 1 to RAIL_PX; 0 when it isn't on a rail. */
function railRise(frame: TileFrame, sides: readonly Edge[], x: number, y: number): number {
  for (let rise = RAIL_PX; rise >= 1; rise--) {
    if (sides.some((edge) => depthFromEdge(frame, edge, x, y + rise) < 1)) return rise;
  }
  return 0;
}

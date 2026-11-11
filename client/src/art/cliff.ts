import { depthFromEdge, isAtCorner, type Corner, type Edge } from './edges';
import { createPixelBuffer, forEachPixel, setPixel, type PixelBuffer } from './pixelBuffer';
import { isOnGround, textureSize, type TileFrame } from './tileFrame';

/**
 * What a cliff face is made of: earth under grass and holes, wood posts under a bridge's deck, stone under a mountain
 * and ice under a glacier (ADR 0009).
 */
export type FaceMaterial = 'earth' | 'wood' | 'stone' | 'ice';
/** Which side of a cell a face shows: `left` toward the next row, lit from the top-left; `right` toward the next column. */
export type FaceSide = 'left' | 'right';

export const FACE_COLORS: Readonly<Record<FaceMaterial, Readonly<Record<FaceSide, { face: number; band: number }>>>> = {
  earth: { left: { face: 0xc99b6b, band: 0xa9805a }, right: { face: 0xa77a51, band: 0x8a6243 } },
  wood: { left: { face: 0xa8794b, band: 0x7d5734 }, right: { face: 0x8c633c, band: 0x6a4a2c } },
  stone: { left: { face: 0x8a8694, band: 0x6b6775 }, right: { face: 0x736f7d, band: 0x56525f } },
  ice: { left: { face: 0xa6d8ee, band: 0x86c3e3 }, right: { face: 0x8ccbe8, band: 0x6fb3d8 } },
};

/** How many pixels at the bottom of each level are its darker band. */
const BAND_PX = 2;
/** Wood faces are posts: a darker seam every this many pixels. */
const POST_PX = 4;

/**
 * One level of a cliff face: the `levelPx` pixels right under one side of the ground diamond. The texture is a tile
 * frame (no headroom) made `levelPx` taller, so it shares the tile's ground center. Each level has a darker band at its
 * bottom. Over a face, the grass top has a rim of the face's color: see paintRim.
 */
export function paintFace(frame: TileFrame, levelPx: number, material: FaceMaterial, side: FaceSide): PixelBuffer {
  const buffer = createPixelBuffer(frame.tileWidth, frame.tileHeight + frame.headroom + levelPx);
  const colors = FACE_COLORS[material];
  const half = frame.tileWidth / 2;
  for (let x = side === 'left' ? 0 : half; x < (side === 'left' ? half : frame.tileWidth); x++) {
    const bottom = lowestGroundPixel(frame, x);
    if (bottom === null) continue;
    for (let i = 1; i <= levelPx; i++) {
      const seam = material === 'wood' && x % POST_PX === 0;
      const rgb = i > levelPx - BAND_PX || seam ? colors[side].band : colors[side].face;
      setPixel(buffer, x, bottom + i, rgb);
    }
  }
  return buffer;
}

/** The lowest pixel row of the ground diamond in a texture column, null when none. */
function lowestGroundPixel(frame: TileFrame, x: number): number | null {
  for (let y = frame.headroom + frame.tileHeight - 1; y >= 0; y--) {
    if (isOnGround(frame, x, y)) return y;
  }
  return null;
}

/** The side of a face each edge's or corner's rim takes its color from: the left ones are lit, like the left faces. */
const RIM_SIDES: Readonly<Record<Edge | Corner, FaceSide>> = { nw: 'left', sw: 'left', ne: 'right', se: 'right', w: 'left', n: 'left', e: 'right', s: 'right' };

/**
 * The rim of a grass top toward lower cells, one pixel wide, inside the diamond, in the earth faces' color: drawn over
 * the grass, it marks where the ground steps down, the back edges included, which show no face. Along each edge toward
 * a lower cell, and on each corner toward a lower diagonal neighbor: there, the rims of the cells around it meet.
 */
export function paintRim(frame: TileFrame, edges: readonly Edge[], corners: readonly Corner[] = []): PixelBuffer {
  const { width, height } = textureSize(frame);
  const buffer = createPixelBuffer(width, height);
  forEachPixel(width, height, (x, y) => {
    const along = edges.find((e) => depthFromEdge(frame, e, x, y) < 1) ?? corners.find((c) => isAtCorner(frame, c, x, y));
    if (along) setPixel(buffer, x, y, FACE_COLORS.earth[RIM_SIDES[along]].face);
  });
  return buffer;
}

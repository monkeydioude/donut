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
  ice: { left: { face: 0x6cc0e0, band: 0x5aaed3 }, right: { face: 0x56acd4, band: 0x4796c2 } },
};

/** Ice faces have lighter vertical streaks (ADR 0011), every few pixels, the same on a glacier's sides. */
export const ICE_STREAKS: Readonly<Record<FaceSide, number>> = { left: 0x8ad0ea, right: 0x6fbfe0 };
const ICE_STREAK_PX = 4;

/** True for the texture columns of an ice streak: tiles are placed 16 pixels apart, so streaks line up. */
export function isIceStreak(x: number): boolean {
  return x % ICE_STREAK_PX === 1;
}

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
  const half = frame.tileWidth / 2;
  for (let x = side === 'left' ? 0 : half; x < (side === 'left' ? half : frame.tileWidth); x++) {
    const bottom = lowestGroundPixel(frame, x);
    if (bottom === null) continue;
    for (let i = 1; i <= levelPx; i++) setPixel(buffer, x, bottom + i, faceColor(material, side, x, i > levelPx - BAND_PX));
  }
  return buffer;
}

/** Stone faces have a darker crack every this many pixels: tiles are 16 apart, so they fall unevenly, like rock. */
const STONE_CRACK_PX = 7;

/**
 * A face's pixel. Earth and wood have a band at the bottom of each level, wood its seams. Stone and ice are one mass
 * (ADR 0011): no bands, stone with uneven vertical cracks, ice with lighter streaks.
 */
function faceColor(material: FaceMaterial, side: FaceSide, x: number, inBand: boolean): number {
  const colors = FACE_COLORS[material][side];
  switch (material) {
    case 'stone':
      return x % STONE_CRACK_PX === 3 ? colors.band : colors.face;
    case 'ice':
      return isIceStreak(x) ? ICE_STREAKS[side] : colors.face;
    case 'wood':
      return inBand || x % POST_PX === 0 ? colors.band : colors.face;
    default:
      return inBand ? colors.band : colors.face;
  }
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

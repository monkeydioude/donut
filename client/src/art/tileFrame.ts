import { NEIGHBORS } from './pixelBuffer';

/**
 * Layout of a tile texture: the ground diamond at the bottom, plus `headroom` pixels above it
 * for art that rises off the ground (rocks, bushes). A plain ground tile has no headroom.
 */
export interface TileFrame {
  tileWidth: number;
  tileHeight: number;
  headroom: number;
}

export function textureSize(frame: TileFrame): { width: number; height: number } {
  return { width: frame.tileWidth, height: frame.tileHeight + frame.headroom };
}

/** Point of the texture that sits on the center of the tile's ground diamond. */
export function groundCenter(frame: TileFrame): { x: number; y: number } {
  return { x: frame.tileWidth / 2, y: frame.headroom + frame.tileHeight / 2 };
}

/** True when the texture pixel (x, y) is on the tile's ground diamond. */
export function isOnGround(frame: TileFrame, x: number, y: number): boolean {
  if (!isInsideTexture(frame, x, y)) return false;
  const center = groundCenter(frame);
  // Pixel centers never land exactly on a diamond edge, so neighbor tiles join without gaps or overlaps.
  const dx = Math.abs(x + 0.5 - center.x) / (frame.tileWidth / 2);
  const dy = Math.abs(y + 0.5 - center.y) / (frame.tileHeight / 2);
  return dx + dy <= 1;
}

/** On the ground and away from its border, so details drawn there are never cut. */
export function isInnerGround(frame: TileFrame, x: number, y: number): boolean {
  return isOnGround(frame, x, y) && NEIGHBORS.every(([dx, dy]) => isOnGround(frame, x + dx, y + dy));
}

/**
 * True where the tile may draw: on its ground diamond, or anywhere above its ground center.
 * Art there only overlaps tiles behind it (drawn earlier), never the tiles in front.
 */
export function isDrawable(frame: TileFrame, x: number, y: number): boolean {
  if (!isInsideTexture(frame, x, y)) return false;
  return y + 0.5 < groundCenter(frame).y || isOnGround(frame, x, y);
}

function isInsideTexture(frame: TileFrame, x: number, y: number): boolean {
  const { width, height } = textureSize(frame);
  return x >= 0 && y >= 0 && x < width && y < height;
}

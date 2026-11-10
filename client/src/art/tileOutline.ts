import { createPixelBuffer, forEachPixel, setPixel, type PixelBuffer } from './pixelBuffer';
import { isInnerGround, isOnGround, textureSize, type TileFrame } from './tileFrame';

/** The rim of a tile's ground diamond, one pixel wide, inside the tile: the hover and selection borders. */
export function paintTileOutline(frame: TileFrame, rgb: number, alpha = 0xff): PixelBuffer {
  const { width, height } = textureSize(frame);
  const buffer = createPixelBuffer(width, height);
  forEachPixel(width, height, (x, y) => {
    if (isOnGround(frame, x, y) && !isInnerGround(frame, x, y)) setPixel(buffer, x, y, rgb, alpha);
  });
  return buffer;
}

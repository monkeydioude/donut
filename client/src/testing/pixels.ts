import { forEachPixel, readPixel, type PixelBuffer } from '../art/pixelBuffer';

/** Coordinates as "x,y" of the pixels matching the predicate, for readable test failures. */
export function pixelsWhere(buffer: PixelBuffer, predicate: (x: number, y: number) => boolean): string[] {
  const found: string[] = [];
  forEachPixel(buffer.width, buffer.height, (x, y) => {
    if (predicate(x, y)) found.push(`${x},${y}`);
  });
  return found;
}

export function isOpaque(buffer: PixelBuffer, x: number, y: number): boolean {
  return readPixel(buffer, x, y).alpha === 255;
}

export function distinctColors(buffer: PixelBuffer): Set<number> {
  const colors = new Set<number>();
  forEachPixel(buffer.width, buffer.height, (x, y) => {
    if (isOpaque(buffer, x, y)) colors.add(readPixel(buffer, x, y).rgb);
  });
  return colors;
}

/** RGBA image in memory, independent of any rendering library. */
export interface PixelBuffer {
  readonly width: number;
  readonly height: number;
  readonly data: Uint8ClampedArray<ArrayBuffer>;
}

/** Offsets of the 4 pixels touching a pixel by a side. */
export const NEIGHBORS = [[1, 0], [-1, 0], [0, 1], [0, -1]] as const;

export interface Pixel {
  rgb: number;
  alpha: number;
}

export function createPixelBuffer(width: number, height: number): PixelBuffer {
  return { width, height, data: new Uint8ClampedArray(width * height * 4) };
}

/** Sets a pixel from a 0xRRGGBB color, opaque unless an alpha (0 to 255) is given. */
export function setPixel(buffer: PixelBuffer, x: number, y: number, rgb: number, alpha = 0xff): void {
  const i = offsetOf(buffer, x, y);
  buffer.data[i] = (rgb >> 16) & 0xff;
  buffer.data[i + 1] = (rgb >> 8) & 0xff;
  buffer.data[i + 2] = rgb & 0xff;
  buffer.data[i + 3] = alpha;
}

export function readPixel(buffer: PixelBuffer, x: number, y: number): Pixel {
  const i = offsetOf(buffer, x, y);
  const channel = (k: number) => buffer.data[i + k] ?? 0;
  return { rgb: (channel(0) << 16) | (channel(1) << 8) | channel(2), alpha: channel(3) };
}

export function forEachPixel(width: number, height: number, visit: (x: number, y: number) => void): void {
  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) visit(x, y);
  }
}

function offsetOf(buffer: PixelBuffer, x: number, y: number): number {
  if (x < 0 || y < 0 || x >= buffer.width || y >= buffer.height) {
    throw new RangeError(`pixel (${x}, ${y}) is outside the ${buffer.width}x${buffer.height} buffer`);
  }
  return (y * buffer.width + x) * 4;
}

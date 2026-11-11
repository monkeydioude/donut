import type { PixelBuffer } from '../art/pixelBuffer';

/**
 * A CSS cursor value from a pixel sprite, scaled up without smoothing. The hotspot is in sprite pixels.
 * Falls back to the default arrow if the browser refuses the image.
 */
export function spriteCursor(sprite: PixelBuffer, hotspot: { x: number; y: number }, scale: number): string {
  const url = spriteDataUrl(sprite, scale);
  return url ? `url("${url}") ${hotspot.x * scale} ${hotspot.y * scale}, auto` : 'auto';
}

/** A pixel sprite as a PNG data URL, scaled up without smoothing. Null when the browser gives no 2D canvas. */
export function spriteDataUrl(sprite: PixelBuffer, scale: number): string | null {
  const source = document.createElement('canvas');
  source.width = sprite.width;
  source.height = sprite.height;
  source.getContext('2d')?.putImageData(new ImageData(sprite.data, sprite.width, sprite.height), 0, 0);

  const scaled = document.createElement('canvas');
  scaled.width = sprite.width * scale;
  scaled.height = sprite.height * scale;
  const context = scaled.getContext('2d');
  if (!context) return null;
  context.imageSmoothingEnabled = false;
  context.drawImage(source, 0, 0, scaled.width, scaled.height);
  return scaled.toDataURL('image/png');
}

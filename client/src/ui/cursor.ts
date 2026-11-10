import type { PixelBuffer } from '../art/pixelBuffer';

/**
 * A CSS cursor value from a pixel sprite, scaled up without smoothing. The hotspot is in sprite pixels.
 * Falls back to the default arrow if the browser refuses the image.
 */
export function spriteCursor(sprite: PixelBuffer, hotspot: { x: number; y: number }, scale: number): string {
  const source = document.createElement('canvas');
  source.width = sprite.width;
  source.height = sprite.height;
  source.getContext('2d')?.putImageData(new ImageData(sprite.data, sprite.width, sprite.height), 0, 0);

  const scaled = document.createElement('canvas');
  scaled.width = sprite.width * scale;
  scaled.height = sprite.height * scale;
  const context = scaled.getContext('2d');
  if (!context) return 'auto';
  context.imageSmoothingEnabled = false;
  context.drawImage(source, 0, 0, scaled.width, scaled.height);
  return `url("${scaled.toDataURL('image/png')}") ${hotspot.x * scale} ${hotspot.y * scale}, auto`;
}

import { describe, expect, it } from 'vitest';
import { isOpaque, pixelsWhere } from '../testing/pixels';
import { isInEllipse, stampOnGround } from './drawing';
import { createPixelBuffer, readPixel } from './pixelBuffer';
import type { TileFrame } from './tileFrame';

const frame: TileFrame = { tileWidth: 32, tileHeight: 16, headroom: 0 };
const dot = [{ dx: 0, dy: 0, rgb: 0x123456 }, { dx: 1, dy: 0, rgb: 0x654321 }];

describe('stampOnGround', () => {
  it('draws the pattern when it fits on the inner ground', () => {
    const buffer = createPixelBuffer(32, 16);
    expect(stampOnGround(buffer, frame, { x: 15, y: 8 }, dot)).toBe(true);
    expect(readPixel(buffer, 15, 8).rgb).toBe(0x123456);
    expect(readPixel(buffer, 16, 8).rgb).toBe(0x654321);
  });

  it('draws nothing when part of the pattern would leave the inner ground', () => {
    const buffer = createPixelBuffer(32, 16);
    expect(stampOnGround(buffer, frame, { x: 16, y: 0 }, dot)).toBe(false);
    expect(pixelsWhere(buffer, (x, y) => isOpaque(buffer, x, y))).toEqual([]);
  });
});

describe('isInEllipse', () => {
  const ellipse = { cx: 10, cy: 5, rx: 4, ry: 2 };

  it('tests pixel centers against the ellipse', () => {
    expect(isInEllipse(ellipse, 9, 4)).toBe(true);
    expect(isInEllipse(ellipse, 13, 4)).toBe(true);
    expect(isInEllipse(ellipse, 14, 4)).toBe(false);
    expect(isInEllipse(ellipse, 9, 7)).toBe(false);
  });
});

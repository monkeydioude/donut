import { describe, expect, it } from 'vitest';
import { createSeededRng } from '../core/rng';
import { distinctColors, isOpaque, pixelsWhere } from '../testing/pixels';
import { readPixel } from './pixelBuffer';
import { paintRock } from './rock';
import { groundCenter, isDrawable, isOnGround, type TileFrame } from './tileFrame';

const frames: TileFrame[] = [
  { tileWidth: 32, tileHeight: 16, headroom: 6 },
  { tileWidth: 64, tileHeight: 32, headroom: 12 },
];
const seeds = Array.from({ length: 20 }, (_, i) => i);

// ADR 0001 test 27
describe.each(frames)('paintRock on a $tileWidth x $tileHeight tile', (frame) => {
  const isPainted = (buffer: ReturnType<typeof paintRock>, x: number, y: number) => readPixel(buffer, x, y).alpha > 0;

  it('never draws over the tiles in front of it', () => {
    for (const seed of seeds) {
      const buffer = paintRock(frame, createSeededRng(seed));
      expect(pixelsWhere(buffer, (x, y) => isPainted(buffer, x, y) && !isDrawable(frame, x, y))).toEqual([]);
    }
  });

  it('is only a boulder and its shadow: the ground around it stays transparent', () => {
    const { y } = groundCenter(frame);
    for (const seed of seeds) {
      const buffer = paintRock(frame, createSeededRng(seed));
      expect(isPainted(buffer, 1, y)).toBe(false);
      expect(isPainted(buffer, frame.tileWidth - 2, y)).toBe(false);
    }
  });

  it('casts a see-through shadow', () => {
    const buffer = paintRock(frame, createSeededRng(1));
    const shadow = pixelsWhere(buffer, (x, y) => readPixel(buffer, x, y).alpha > 0 && !isOpaque(buffer, x, y));
    expect(shadow.length).toBeGreaterThan(0);
  });

  it('has a boulder rising above the ground', () => {
    for (const seed of seeds) {
      const buffer = paintRock(frame, createSeededRng(seed));
      expect(pixelsWhere(buffer, (x, y) => isOpaque(buffer, x, y) && !isOnGround(frame, x, y)).length).toBeGreaterThan(0);
    }
  });

  it('uses a small palette', () => {
    for (const seed of seeds) {
      expect(distinctColors(paintRock(frame, createSeededRng(seed))).size).toBeLessThanOrEqual(6);
    }
  });

  it('paints the same rock for the same seed', () => {
    expect(paintRock(frame, createSeededRng(5)).data).toEqual(paintRock(frame, createSeededRng(5)).data);
  });
});

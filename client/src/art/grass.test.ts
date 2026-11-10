import { describe, expect, it } from 'vitest';
import { createSeededRng } from '../core/rng';
import { distinctColors, isOpaque, pixelsWhere } from '../testing/pixels';
import { GRASS_COLORS, paintGrass } from './grass';
import { forEachPixel, readPixel } from './pixelBuffer';
import { isOnGround, type TileFrame } from './tileFrame';

const frame: TileFrame = { tileWidth: 32, tileHeight: 16, headroom: 0 };

describe('paintGrass', () => {
  it('paints exactly the ground diamond, opaque', () => {
    const buffer = paintGrass(frame, createSeededRng(3));
    expect(pixelsWhere(buffer, (x, y) => isOpaque(buffer, x, y) !== isOnGround(frame, x, y))).toEqual([]);
  });

  it('is mostly the flat base color, with small details', () => {
    const buffer = paintGrass(frame, createSeededRng(3));
    let base = 0;
    let total = 0;
    forEachPixel(buffer.width, buffer.height, (x, y) => {
      if (!isOnGround(frame, x, y)) return;
      total++;
      if (readPixel(buffer, x, y).rgb === GRASS_COLORS.base) base++;
    });
    expect(base / total).toBeGreaterThan(0.85);
  });

  it('uses a small palette', () => {
    for (let seed = 0; seed < 20; seed++) {
      expect(distinctColors(paintGrass(frame, createSeededRng(seed))).size).toBeLessThanOrEqual(4);
    }
  });

  it('paints the same tile for the same seed', () => {
    expect(paintGrass(frame, createSeededRng(5)).data).toEqual(paintGrass(frame, createSeededRng(5)).data);
  });
});

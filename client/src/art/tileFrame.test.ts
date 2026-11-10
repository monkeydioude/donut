import { describe, expect, it } from 'vitest';
import { createIsoProjection } from '../iso/projection';
import { groundCenter, isDrawable, isInnerGround, isOnGround, textureSize, type TileFrame } from './tileFrame';

const flat: TileFrame = { tileWidth: 32, tileHeight: 16, headroom: 0 };
const tall: TileFrame = { tileWidth: 32, tileHeight: 16, headroom: 6 };

describe('textureSize and groundCenter', () => {
  it('adds the headroom above the ground diamond', () => {
    expect(textureSize(flat)).toEqual({ width: 32, height: 16 });
    expect(textureSize(tall)).toEqual({ width: 32, height: 22 });
    expect(groundCenter(flat)).toEqual({ x: 16, y: 8 });
    expect(groundCenter(tall)).toEqual({ x: 16, y: 14 });
  });
});

describe('isOnGround', () => {
  it('is true on the diamond, false in the corners and outside the texture', () => {
    expect(isOnGround(flat, 16, 8)).toBe(true);
    for (const [x, y] of [[0, 0], [31, 0], [0, 15], [31, 15], [-1, 8], [32, 8]] as const) {
      expect(isOnGround(flat, x, y)).toBe(false);
    }
  });

  it('puts the ground of a tall frame below its headroom', () => {
    expect(isOnGround(tall, 16, 2)).toBe(false);
    expect(isOnGround(tall, 16, 14)).toBe(true);
  });

  it('covers every screen pixel exactly once when tiles are laid side by side', () => {
    const projection = createIsoProjection(32, 16);
    const pivot = groundCenter(flat);
    const tiles = Array.from({ length: 25 }, (_, i) => projection.toScreen({ col: i % 5, row: Math.floor(i / 5) }));
    const coverage = (x: number, y: number) =>
      tiles.filter((t) => isOnGround(flat, x - t.x + pivot.x, y - t.y + pivot.y)).length;

    const badPixels: string[] = [];
    // Area around the center tile (col 2, row 2), whose ground center is at (0, 32).
    for (let y = 20; y < 44; y++) {
      for (let x = -24; x < 24; x++) if (coverage(x, y) !== 1) badPixels.push(`${x},${y}`);
    }
    expect(badPixels).toEqual([]);
  });
});

describe('isInnerGround', () => {
  it('excludes the border of the diamond', () => {
    expect(isInnerGround(flat, 16, 8)).toBe(true);
    expect(isInnerGround(flat, 16, 0)).toBe(false);
  });
});

describe('isDrawable', () => {
  it('allows anything above the ground center, only the diamond below it', () => {
    expect(isDrawable(tall, 0, 0)).toBe(true);
    expect(isDrawable(tall, 0, 13)).toBe(true);
    expect(isDrawable(tall, 16, 20)).toBe(true);
    expect(isDrawable(tall, 2, 20)).toBe(false);
    expect(isDrawable(tall, 0, 14)).toBe(false);
  });

  it('refuses pixels outside the texture', () => {
    expect(isDrawable(tall, -1, 0)).toBe(false);
    expect(isDrawable(tall, 0, 22)).toBe(false);
  });
});

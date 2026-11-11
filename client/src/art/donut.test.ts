import { describe, expect, it } from 'vitest';
import { distinctColors } from '../testing/pixels';
import { DONUT_MARK_RING, DONUT_SIZE, paintDonut } from './donut';
import { paintOtherMark, paintYouMark } from './mapMarks';
import { readPixel, type PixelBuffer } from './pixelBuffer';

const opaque = (sprite: PixelBuffer, x: number, y: number) => readPixel(sprite, x, y).alpha === 255;

// ADR 0005 test 20
describe('paintDonut', () => {
  const donut = paintDonut();
  const middle = DONUT_SIZE / 2;

  it('is 40 x 40, with a transparent hole and corners', () => {
    expect([donut.width, donut.height]).toEqual([40, 40]);
    expect(opaque(donut, middle, middle)).toBe(false);
    for (const [x, y] of [[0, 0], [39, 0], [0, 39], [39, 39]] as const) expect(opaque(donut, x, y)).toBe(false);
  });

  it('has an opaque ring all the way around, where the mark moves', () => {
    const radius = (DONUT_MARK_RING.outer + DONUT_MARK_RING.inner) / 2;
    for (let i = 0; i < 16; i += 1) {
      const angle = (i / 16) * 2 * Math.PI;
      const x = Math.floor(middle + radius * Math.sin(angle));
      const y = Math.floor(middle - radius * Math.cos(angle));
      expect(opaque(donut, x, y), `at ${x},${y}`).toBe(true);
    }
  });

  it('uses a small palette', () => {
    expect(distinctColors(donut).size).toBeLessThanOrEqual(6);
  });
});

/** How many opaque pixels a row of a sprite has. */
const widthOfRow = (sprite: PixelBuffer, y: number) =>
  Array.from({ length: sprite.width }, (_, x) => opaque(sprite, x, y)).filter(Boolean).length;

// ADR 0005 test 21
describe('the map marks', () => {
  it('▲ is wider at the bottom, ▼ at the top', () => {
    const up = paintYouMark('up');
    const down = paintYouMark('down');
    expect(widthOfRow(up, 0)).toBeLessThan(widthOfRow(up, up.height - 1));
    expect(widthOfRow(down, 0)).toBeGreaterThan(widthOfRow(down, down.height - 1));
  });

  it('the square is as wide at the top as at the bottom', () => {
    const square = paintYouMark('square');
    expect(widthOfRow(square, 0)).toBe(widthOfRow(square, square.height - 1));
    expect(square.width).toBe(square.height);
  });

  it("your marks are gold, the other players' cream", () => {
    expect(readPixel(paintYouMark('square'), 2, 2).rgb).toBe(0xffd84a);
    expect(readPixel(paintOtherMark(), 1, 1).rgb).toBe(0xfff4dc);
  });
});

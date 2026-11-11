import { describe, expect, it } from 'vitest';
import { blockFrame, FLAT_BLOCK, forEachBlockPixel, shadeIndex, type BlockPixel, type BlockShape } from './block';
import { isOnGround, type TileFrame } from './tileFrame';

const LEVEL_PX = 8;

/** Every pixel of a block, by "x,y", in its frame. */
function pixelsOf(shape: BlockShape): { frame: TileFrame; pixels: Map<string, BlockPixel> } {
  const frame = blockFrame(32, 16, shape, LEVEL_PX);
  const pixels = new Map<string, BlockPixel>();
  forEachBlockPixel(frame, shape, LEVEL_PX, (x, y, pixel) => pixels.set(`${x},${y}`, pixel));
  return { frame, pixels };
}

function sidesOf(pixels: Map<string, BlockPixel>, side: 'left' | 'right'): number {
  return [...pixels.values()].filter((p) => p.part === 'side' && p.side === side).length;
}

// ADR 0011 test 12
describe('forEachBlockPixel', () => {
  it('a flat block is its ground diamond, all top, lit like flat ground', () => {
    const { frame, pixels } = pixelsOf(FLAT_BLOCK);
    expect(frame.headroom).toBe(0);
    for (let y = 0; y < 16; y++) for (let x = 0; x < 32; x++) expect(pixels.has(`${x},${y}`), `${x},${y}`).toBe(isOnGround(frame, x, y));
    for (const p of pixels.values()) expect(p.part === 'top' && p.light).toBeCloseTo(0.75, 2);
  });

  it('its back corner raised 2 levels, the top leans up to it, and its front corner stays on the ground', () => {
    const { frame, pixels } = pixelsOf({ ...FLAT_BLOCK, n: 2 });
    expect(frame.headroom).toBe(16);
    const tipRows = [0, 1].filter((y) => pixels.has(`15,${y}`) || pixels.has(`16,${y}`));
    expect(tipRows.length, "the raised back corner's sharp tip, at the top of the frame").toBeGreaterThan(0);
    const lowest = Math.max(...[...pixels.keys()].map((k) => Number(k.split(',')[1])));
    expect(lowest, "the front corner, on the ground diamond's bottom row").toBe(frame.headroom + 15);
    expect(sidesOf(pixels, 'left') + sidesOf(pixels, 'right'), 'no side: the front corners stay down').toBe(0);
  });

  it('a raised side corner shows a side under the front edge going to it', () => {
    const { pixels } = pixelsOf({ ...FLAT_BLOCK, n: 2, w: 2 });
    expect(sidesOf(pixels, 'left')).toBeGreaterThan(0);
    expect(sidesOf(pixels, 'right')).toBe(0);
    const tops = [...pixels.values()].filter((p) => p.part === 'top').map((p) => (p.part === 'top' ? p.light : 0));
    expect(Math.min(...tops), 'leaning back toward the column before, it faces the right: darker than flat').toBeLessThan(0.7);
  });

  it('a peak is a pyramid: its left faces lit, its right ones darker', () => {
    const { pixels } = pixelsOf({ ...FLAT_BLOCK, peak: true });
    const light = (x: number, y: number) => {
      const p = pixels.get(`${x},${y}`);
      return p?.part === 'top' ? p.light : NaN;
    };
    expect(light(10, 6)).toBeGreaterThan(light(21, 6));
    expect(pixels.has('16,0') || pixels.has('15,0'), 'its point').toBe(true);
  });
});

describe('shadeIndex', () => {
  it('gives a flat top the second lightest of 4 shades, steeper slopes the others', () => {
    expect(shadeIndex(0.748, 4)).toBe(2);
    expect(shadeIndex(0.9, 4)).toBe(3);
    expect(shadeIndex(0.6, 4)).toBe(1);
    expect(shadeIndex(0.3, 4)).toBe(0);
    expect(shadeIndex(0.9, 2), 'never past the last shade').toBe(1);
  });
});

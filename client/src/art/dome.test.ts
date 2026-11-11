import { describe, expect, it } from 'vitest';
import { createSeededRng } from '../core/rng';
import { distinctColors, isOpaque, pixelsWhere } from '../testing/pixels';
import { domeFrame, paintDome } from './dome';
import { NEIGHBORS, readPixel } from './pixelBuffer';
import { isDrawable } from './tileFrame';

const OUTLINE = 0x3d3646;
const INSIDE = 0x2f2838;

// ADR 0012: the dome's art, approved on its preview image
describe('paintDome', () => {
  const frame = domeFrame(32, 16);
  const dome = paintDome(frame, createSeededRng(42));
  const painted = (x: number, y: number) => readPixel(dome, x, y).alpha > 0;

  it('fills the 2 × 2 footprint of a 32 × 16 tile, with room above for the dome', () => {
    expect(frame).toEqual({ tileWidth: 64, tileHeight: 32, headroom: 14 });
    expect(pixelsWhere(dome, (x, y) => isOpaque(dome, x, y) && y < frame.headroom).length).toBeGreaterThan(0);
  });

  it('never draws over the tiles in front of its footprint', () => {
    expect(pixelsWhere(dome, (x, y) => painted(x, y) && !isDrawable(frame, x, y))).toEqual([]);
  });

  it('is the same for the same seed', () => {
    expect(paintDome(frame, createSeededRng(42)).data).toEqual(dome.data);
  });

  it('shows the dark inside through its missing cells', () => {
    expect(pixelsWhere(dome, (x, y) => isOpaque(dome, x, y) && readPixel(dome, x, y).rgb === INSIDE).length).toBeGreaterThan(0);
  });

  it('is outlined on its silhouette only', () => {
    const outline = pixelsWhere(dome, (x, y) => readPixel(dome, x, y).rgb === OUTLINE && isOpaque(dome, x, y));
    expect(outline.length).toBeGreaterThan(0);
    const onEdge = (x: number, y: number) =>
      NEIGHBORS.some(([dx, dy]) => {
        const nx = x + dx;
        const ny = y + dy;
        return nx < 0 || ny < 0 || nx >= dome.width || ny >= dome.height || !isOpaque(dome, nx, ny);
      });
    expect(outline.filter((at) => !onEdge(...(at.split(',').map(Number) as [number, number])))).toEqual([]);
  });

  it('casts a see-through shadow, which picking ignores', () => {
    expect(pixelsWhere(dome, (x, y) => painted(x, y) && !isOpaque(dome, x, y)).length).toBeGreaterThan(0);
  });

  it('keeps to a small palette, as pixel art', () => {
    expect(distinctColors(dome).size).toBeLessThanOrEqual(24);
  });
});

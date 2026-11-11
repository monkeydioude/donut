import { describe, expect, it } from 'vitest';
import { createIsoProjection } from '../iso/projection';
import { cellAt, followPosition, tileOnCanvas } from './camera';

const projection = createIsoProjection(64, 32);
const heights = { projection, levelPx: 16 };
const flat = () => 0;

// ADR 0003 test 30
describe('followPosition', () => {
  it('puts your drawn position in the middle of the viewport', () => {
    expect(followPosition({ x: 0, y: 32 }, { width: 800, height: 600 }, 1)).toEqual({ x: 400, y: 268 });
  });

  it('takes the zoom into account', () => {
    expect(followPosition({ x: 0, y: 32 }, { width: 800, height: 600 }, 2)).toEqual({ x: 400, y: 236 });
  });

  it('lands on whole pixels, mid-glide too', () => {
    const position = followPosition({ x: 10.3, y: 7.7 }, { width: 801, height: 601 }, 3);
    expect(Number.isInteger(position.x) && Number.isInteger(position.y)).toBe(true);
  });
});

// ADR 0003 test 29, and ADR 0002 test 17
describe('cellAt', () => {
  const world = { origin: { x: 400, y: 100 }, zoom: 2 };
  const bounded = { width: 4, height: 4, wraps: false };

  it('finds the cell under a canvas point, through the world origin and zoom', () => {
    // tile (2, 1) is centered on world (32, 48), so on the canvas at (400 + 64, 100 + 96)
    expect(cellAt({ x: 464, y: 196 }, world, heights, flat, bounded)).toEqual({ col: 2, row: 1 });
  });

  // ADR 0012: a click there leaves a city
  it('gives the cell past the edge of a bounded map', () => {
    expect(cellAt({ x: 400, y: 60 }, world, heights, flat, bounded)).toEqual({ col: -1, row: -1 });
    expect(cellAt({ x: 400, y: 100 + 8 * 32 * 2 }, world, heights, flat, bounded)).toEqual({ col: 8, row: 8 });
  });

  it('gives the dome drawn there first (ADR 0012)', () => {
    const dome = () => ({ col: 3, row: 3 });
    expect(cellAt({ x: 464, y: 196 }, world, heights, flat, bounded, dome)).toEqual({ col: 3, row: 3 });
  });

  it('gives the cell within the map for a drawn copy past the seam', () => {
    // drawn tile (-1, 0) is centered on world (-32, -16): on the canvas at (400 - 64, 100 - 32)
    expect(cellAt({ x: 336, y: 68 }, world, heights, flat, { width: 160, height: 160, wraps: true })).toEqual({ col: 159, row: 0 });
  });
});

describe('tileOnCanvas', () => {
  it('gives the corners and middle of a tile on the canvas', () => {
    const world = { origin: { x: 400, y: 100 }, zoom: 2 };
    expect(tileOnCanvas({ col: 2, row: 1 }, 0, world, heights, 64)).toEqual({ left: 400, right: 528, y: 196 });
  });

  it('at its height (ADR 0007)', () => {
    const world = { origin: { x: 400, y: 100 }, zoom: 2 };
    expect(tileOnCanvas({ col: 2, row: 1 }, 2, world, heights, 64)).toEqual({ left: 400, right: 528, y: 196 - 2 * 16 * 2 });
  });
});

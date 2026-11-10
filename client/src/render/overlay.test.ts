import { describe, expect, it } from 'vitest';
import { createIsoProjection } from '../iso/projection';
import { pathPoints } from './overlay';

// ADR 0002 test 23
describe('pathPoints', () => {
  const projection = createIsoProjection(32, 16);
  const bounded = { width: 64, height: 64, wraps: false };

  it('starts where your player is drawn, then goes through every cell of the path, at their ground centers', () => {
    const gliding = { x: 4, y: 34 }; // a quarter of the way from (2, 2) to (3, 2)
    const path = [{ col: 3, row: 2 }, { col: 4, row: 3 }];
    expect(pathPoints(gliding, { col: 3, row: 2 }, path, projection, bounded)).toEqual([
      { x: 4, y: 34 },
      { x: 16, y: 40 },
      { x: 16, y: 56 },
    ]);
  });

  it('draws nothing for an empty path', () => {
    expect(pathPoints({ x: 0, y: 32 }, { col: 2, row: 2 }, [], projection, bounded)).toEqual([]);
  });

  // ADR 0003 test 28
  it('never jumps across the world: each cell is the copy nearest to the one before', () => {
    const world = { width: 1024, height: 512, wraps: true };
    const start = projection.toScreen({ col: 1023, row: 5 });
    const path = [{ col: 0, row: 5 }, { col: 1, row: 5 }];
    expect(pathPoints(start, { col: 1023, row: 5 }, path, projection, world)).toEqual([
      start,
      projection.toScreen({ col: 1024, row: 5 }),
      projection.toScreen({ col: 1025, row: 5 }),
    ]);
  });
});

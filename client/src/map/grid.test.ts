import { describe, expect, it } from 'vitest';
import { chunkId, chunkOf, chunkOrigin, nearestCopy, normalize, type Grid } from './grid';

const torus: Grid = { width: 64, height: 64, wraps: true };
const bounded: Grid = { width: 64, height: 64, wraps: false };

// ADR 0003 test 24
describe('normalize', () => {
  it('wraps a cell into the map on a wrapping grid', () => {
    expect(normalize(torus, { col: -1, row: 0 })).toEqual({ col: 63, row: 0 });
    expect(normalize(torus, { col: 64, row: 65 })).toEqual({ col: 0, row: 1 });
    expect(normalize(torus, { col: 1024 + 5, row: -129 })).toEqual({ col: 5, row: 63 });
  });

  it('gives null outside a bounded grid', () => {
    expect(normalize(bounded, { col: -1, row: 0 })).toBeNull();
    expect(normalize(bounded, { col: 3, row: 64 })).toBeNull();
    expect(normalize(bounded, { col: 3, row: 4 })).toEqual({ col: 3, row: 4 });
  });
});

describe('nearestCopy', () => {
  it('gives the copy of a cell nearest to the anchor, across the seams', () => {
    expect(nearestCopy(torus, { col: 0, row: 5 }, { col: 63, row: 5 })).toEqual({ col: 64, row: 5 });
    expect(nearestCopy(torus, { col: 63, row: 63 }, { col: 0, row: 0 })).toEqual({ col: -1, row: -1 });
    expect(nearestCopy(torus, { col: 10, row: 10 }, { col: 12, row: 9 })).toEqual({ col: 10, row: 10 });
  });

  it('follows an anchor that went around the world', () => {
    expect(nearestCopy(torus, { col: 3, row: 0 }, { col: 64 * 2 + 1, row: 0 })).toEqual({ col: 64 * 2 + 3, row: 0 });
  });

  it('is the cell itself on a bounded grid', () => {
    expect(nearestCopy(bounded, { col: 0, row: 5 }, { col: 63, row: 5 })).toEqual({ col: 0, row: 5 });
  });
});

describe('chunks', () => {
  it('finds the chunk of a cell', () => {
    expect(chunkOf({ col: 31, row: 31 })).toEqual({ cx: 0, cy: 0 });
    expect(chunkOf({ col: 32, row: 0 })).toEqual({ cx: 1, cy: 0 });
  });

  it("gives a chunk's first cell and its id", () => {
    expect(chunkOrigin({ cx: 2, cy: 1 })).toEqual({ col: 64, row: 32 });
    expect(chunkId({ cx: 2, cy: 1 })).toBe('2,1');
  });
});

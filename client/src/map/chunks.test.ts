import { describe, expect, it } from 'vitest';
import { emptyChunks, groundAt, isHeld, regionAt, withChunk, withoutChunk, type Chunk } from './chunks';

function chunk(cx: number, cy: number, region = 1): Chunk {
  const ground = Array<'grass' | 'void'>(1024).fill('grass');
  ground[33] = 'void'; // the cell (1, 1) of the chunk
  return { key: { cx, cy }, version: 1, ground, height: Array(1024).fill(0), region: Array(1024).fill(region), objects: [] };
}

// ADR 0003 test 26
describe('held chunks', () => {
  it('a chunk adds its cells, a forget removes them', () => {
    const held = withChunk(emptyChunks(), chunk(1, 0));
    expect(isHeld(held, { col: 32, row: 0 })).toBe(true);
    expect(isHeld(held, { col: 63, row: 31 })).toBe(true);
    expect(isHeld(held, { col: 31, row: 0 }), 'the next chunk west').toBe(false);
    expect(isHeld(withoutChunk(held, { cx: 1, cy: 0 }), { col: 32, row: 0 })).toBe(false);
    expect(isHeld(held, { col: 32, row: 0 }), 'the previous chunks are left untouched').toBe(true);
  });

  it("a cell's ground and region come from its chunk", () => {
    const held = withChunk(withChunk(emptyChunks(), chunk(0, 0, 2)), chunk(1, 0, 5));
    expect(groundAt(held, { col: 33, row: 1 })).toBe('void');
    expect(groundAt(held, { col: 34, row: 1 })).toBe('grass');
    expect(regionAt(held, { col: 3, row: 3 })).toBe(2);
    expect(regionAt(held, { col: 40, row: 3 })).toBe(5);
  });

  it('a cell of a chunk not held is unknown', () => {
    expect(groundAt(emptyChunks(), { col: 0, row: 0 })).toBeNull();
    expect(regionAt(emptyChunks(), { col: 0, row: 0 })).toBeNull();
  });
});

import { describe, expect, it } from 'vitest';
import { emptyChunks, groundAt, heightAt, heightForDrawing, isHeld, regionAt, terrainAt, withChunk, withoutChunk, type Chunk } from './chunks';

function chunk(cx: number, cy: number, region = 1): Chunk {
  const ground = Array<'grass' | 'water' | 'void'>(1024).fill('grass');
  ground[33] = 'void'; // the cell (1, 1) of the chunk
  ground[35] = 'water'; // (3, 1)
  const height = Array<number>(1024).fill(0);
  height[2] = 3; // (2, 0)
  height[35] = -1;
  return { key: { cx, cy }, version: 1, ground, height, region: Array(1024).fill(region), objects: [] };
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
    expect(heightAt(emptyChunks(), { col: 0, row: 0 })).toBeNull();
    expect(terrainAt(emptyChunks(), { col: 0, row: 0 })).toBeNull();
  });

  // ADR 0007 test 27
  it("a cell's height comes from its chunk", () => {
    const held = withChunk(emptyChunks(), chunk(1, 0));
    expect(heightAt(held, { col: 34, row: 0 })).toBe(3);
    expect(heightAt(held, { col: 35, row: 0 })).toBe(0);
    expect(terrainAt(held, { col: 35, row: 1 })).toEqual({ kind: 'water', height: -1 });
  });

  it('a drawn cell is drawn at the height of its cell within the map, 0 when not held', () => {
    const heightOf = heightForDrawing({ current: withChunk(emptyChunks(), chunk(0, 0)) }, { width: 160, height: 160, wraps: true });
    expect(heightOf({ col: 2, row: 0 })).toBe(3);
    expect(heightOf({ col: 162, row: 160 }), 'a copy past both seams').toBe(3);
    expect(heightOf({ col: 40, row: 0 }), 'a chunk not held').toBe(0);
  });
});

import { describe, expect, it } from 'vitest';
import { createIsoProjection } from '../iso/projection';
import type { Chunk } from '../map/chunks';
import type { Grid } from '../map/grid';
import { drawnChunkOrigin, layoutChunkGround } from './chunkView';

const projection = createIsoProjection(64, 32);

function chunkWithVoid(...voidCells: number[]): Chunk {
  const ground = Array<'grass' | 'void'>(1024).fill('grass');
  for (const i of voidCells) ground[i] = 'void';
  return { key: { cx: 2, cy: 1 }, version: 1, ground, height: Array(1024).fill(0), region: Array(1024).fill(1), objects: [] };
}

// ADR 0003 test 31
describe('layoutChunkGround', () => {
  it("places each tile from the chunk's first cell, with its map cell", () => {
    const placed = layoutChunkGround(chunkWithVoid(), projection);
    expect(placed).toHaveLength(1024);
    expect(placed.find((t) => t.cell.col === 65 && t.cell.row === 32)).toEqual({
      kind: 'grass',
      cell: { col: 65, row: 32 },
      local: { x: 32, y: 16 },
      depth: 1,
    });
  });

  it('sorts tiles back to front', () => {
    const depths = layoutChunkGround(chunkWithVoid(), projection).map((t) => t.depth);
    expect(depths).toEqual([...depths].sort((a, b) => a - b));
  });

  it('skips void cells', () => {
    const placed = layoutChunkGround(chunkWithVoid(0, 33), projection);
    expect(placed).toHaveLength(1022);
    expect(placed.some((t) => t.cell.col === 64 && t.cell.row === 32)).toBe(false);
    expect(layoutChunkGround(chunkWithVoid(...Array.from({ length: 1024 }, (_, i) => i)), projection)).toEqual([]);
  });
});

// ADR 0003 test 28
describe('drawnChunkOrigin', () => {
  const world: Grid = { width: 1024, height: 512, wraps: true };

  it('draws a chunk at its copy nearest to your drawn cell', () => {
    expect(drawnChunkOrigin(world, { cx: 1, cy: 1 }, { col: 40, row: 40 })).toEqual({ col: 32, row: 32 });
    expect(drawnChunkOrigin(world, { cx: 31, cy: 0 }, { col: 5, row: 5 }), 'across the seam, to the west').toEqual({ col: -32, row: 0 });
    expect(drawnChunkOrigin(world, { cx: 0, cy: 0 }, { col: 1030, row: 5 }), 'once you went around').toEqual({ col: 1024, row: 0 });
  });

  it('draws a chunk where it is on a bounded map', () => {
    expect(drawnChunkOrigin({ ...world, wraps: false }, { cx: 31, cy: 0 }, { col: 5, row: 5 })).toEqual({ col: 992, row: 0 });
  });
});

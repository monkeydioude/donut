import { describe, expect, it } from 'vitest';
import { createIsoProjection } from '../iso/projection';
import type { CellTerrain, Chunk } from '../map/chunks';
import type { GroundKind } from '../map/worldMap';
import type { Grid } from '../map/grid';
import { chunksToDraw, drawnChunkOrigin, faceLevels, layoutChunk, waterFrame, type TerrainAround } from './chunkView';

const projection = createIsoProjection(64, 32);
const heights = { projection, levelPx: 16 };

/** Chunk (2, 1), all grass at 0, with some cells changed, by their place in the chunk: "col,row". */
function chunkOf(cells: Record<string, { kind?: GroundKind; height?: number }> = {}): Chunk {
  const ground = Array<GroundKind>(1024).fill('grass');
  const height = Array<number>(1024).fill(0);
  for (const [at, cell] of Object.entries(cells)) {
    const [col, row] = at.split(',').map(Number) as [number, number];
    ground[row * 32 + col] = cell.kind ?? 'grass';
    height[row * 32 + col] = cell.height ?? 0;
  }
  return { key: { cx: 2, cy: 1 }, version: 1, ground, height, region: Array(1024).fill(1), objects: [] };
}

const notHeld: TerrainAround = () => null;
const flatAround: TerrainAround = () => ({ kind: 'grass', height: 0 });

/** The tile of a cell of chunk (2, 1), by its place in the chunk. */
function tileAt(chunk: Chunk, col: number, row: number, around: TerrainAround = flatAround) {
  return layoutChunk(chunk, heights, around).tiles.find((t) => t.cell.col === 64 + col && t.cell.row === 32 + row);
}

// ADR 0003 test 31
describe('layoutChunk', () => {
  it("places each tile from the chunk's first cell, with its map cell", () => {
    const { tiles } = layoutChunk(chunkOf(), heights, flatAround);
    expect(tiles).toHaveLength(1024);
    expect(tiles.find((t) => t.cell.col === 65 && t.cell.row === 32)).toEqual({
      kind: 'grass',
      cell: { col: 65, row: 32 },
      local: { x: 32, y: 16 },
      depth: 1,
      faces: [],
      edges: [],
      rims: [],
      rimCorners: [],
    });
  });

  it('sorts tiles back to front', () => {
    const depths = layoutChunk(chunkOf(), heights, flatAround).tiles.map((t) => t.depth);
    expect(depths).toEqual([...depths].sort((a, b) => a - b));
  });

  it('skips void cells', () => {
    const { tiles } = layoutChunk(chunkOf({ '0,0': { kind: 'void' }, '1,1': { kind: 'void' } }), heights, flatAround);
    expect(tiles).toHaveLength(1022);
    expect(tiles.some((t) => t.cell.col === 64 && t.cell.row === 32)).toBe(false);
  });

  // ADR 0007 test 28
  it('draws a tile at its height', () => {
    expect(tileAt(chunkOf({ '1,0': { height: 2 } }), 1, 0)?.local).toEqual({ x: 32, y: 16 - 2 * 16 });
  });
});

// ADR 0007 test 30
describe('cliff faces', () => {
  it('a cell at 1 next to a front neighbor at 0 has a face 1 level high on that side', () => {
    const chunk = chunkOf({ '5,5': { height: 1 }, '5,6': { height: 1 } });
    expect(tileAt(chunk, 5, 5)?.faces, 'its next row is at 1 too: only the next column').toEqual([{ side: 'right', levels: 1 }]);
    expect(tileAt(chunk, 5, 6)?.faces).toEqual([
      { side: 'left', levels: 1 },
      { side: 'right', levels: 1 },
    ]);
  });

  it('no face toward a neighbor as high or higher, nor toward a cell behind', () => {
    const chunk = chunkOf({ '5,5': { height: 1 }, '6,5': { height: 2 }, '5,6': { height: 1 } });
    expect(tileAt(chunk, 5, 5)?.faces).toEqual([]);
    expect(tileAt(chunk, 4, 5)?.faces, 'at 0, before a cell at 1').toEqual([]);
  });

  it('toward void or a chunk not held, the face goes down to -3', () => {
    expect(tileAt(chunkOf({ '6,5': { kind: 'void' } }), 5, 5)?.faces).toEqual([{ side: 'right', levels: 3 }]);
    expect(tileAt(chunkOf({ '31,5': { height: 1 } }), 31, 5, notHeld)?.faces).toEqual([
      { side: 'left', levels: 1 },
      { side: 'right', levels: 4 },
    ]);
    expect(tileAt(chunkOf({ '31,5': { height: 1 } }), 31, 5, flatAround)?.faces, 'once that chunk is held').toEqual([
      { side: 'left', levels: 1 },
      { side: 'right', levels: 1 },
    ]);
  });

  it('counts the levels down to the neighbor, or to the lowest level', () => {
    const at = (kind: GroundKind, height: number): CellTerrain => ({ kind, height });
    expect(faceLevels(2, at('grass', -1))).toBe(3);
    expect(faceLevels(0, at('water', -1))).toBe(1);
    expect(faceLevels(-1, at('grass', 0))).toBe(0);
    expect(faceLevels(-3, null)).toBe(0);
    expect(faceLevels(3, at('void', 0))).toBe(6);
  });
});

// ADR 0007 test 37
describe('terrace rims', () => {
  it('are on the edges toward a lower cell, back edges included', () => {
    const chunk = chunkOf({ '5,5': { height: 1 }, '4,5': { height: 1 }, '5,4': { height: 2 } });
    expect(tileAt(chunk, 5, 5)?.rims.sort(), 'not toward the cell at 1, nor the one at 2').toEqual(['se', 'sw']);
    expect(tileAt(chunk, 5, 4)?.rims.sort()).toEqual(['ne', 'nw', 'se', 'sw']);
    expect(tileAt(chunk, 6, 5)?.rims, 'a lower cell has none toward the higher one').toEqual([]);
  });

  it('close an inner corner: a cell whose diagonal neighbor is lower gets its corner, where the rims around meet', () => {
    // all at 1 but (5, 4): (5, 5) and (6, 4) have rims toward it, and (6, 5) has it beyond its top corner
    const raised: Record<string, { height: number }> = {};
    for (let col = 3; col <= 8; col++) for (let row = 2; row <= 7; row++) if (col !== 5 || row !== 4) raised[`${col},${row}`] = { height: 1 };
    const chunk = chunkOf(raised);
    expect(tileAt(chunk, 5, 5)?.rims).toEqual(['ne']);
    expect(tileAt(chunk, 6, 4)?.rims).toEqual(['nw']);
    expect(tileAt(chunk, 6, 5)?.rims).toEqual([]);
    expect(tileAt(chunk, 6, 5)?.rimCorners).toEqual(['n']);
  });

  it('are on the edges toward void or a chunk not held, like faces', () => {
    expect(tileAt(chunkOf(), 31, 5, notHeld)?.rims).toEqual(['se']);
    expect(tileAt(chunkOf(), 31, 5, flatAround)?.rims).toEqual([]);
  });
});

// ADR 0007 test 31
describe('water', () => {
  it('is left out of the ground, and has one sprite per cell at its height', () => {
    const chunk = chunkOf({ '5,5': { kind: 'water', height: -1 }, '6,5': { kind: 'water', height: -1 } });
    const { tiles, water } = layoutChunk(chunk, heights, flatAround);
    expect(tiles).toHaveLength(1022);
    expect(water.map((w) => w.cell)).toEqual([
      { col: 69, row: 37 },
      { col: 70, row: 37 },
    ]);
    expect(water[0]?.local).toEqual({ x: 0, y: 10 * 16 + 16 });
  });

  // ADR 0007 test 33
  it('has foam on its edges toward cells that are not water, not toward water nor a chunk not held', () => {
    const chunk = chunkOf({ '5,5': { kind: 'water', height: -1 }, '4,5': { kind: 'water', height: -1 }, '31,0': { kind: 'water', height: -1 } });
    const { water } = layoutChunk(chunk, heights, notHeld);
    expect(water.find((w) => w.cell.col === 69)?.foam.sort()).toEqual(['ne', 'se', 'sw']);
    expect(water.find((w) => w.cell.col === 95)?.foam.sort(), 'the chunk past its corner is not held').toEqual(['nw', 'sw']);
  });
});

// ADR 0007 test 34
describe('hole rims', () => {
  it('are on the edges toward cells that are not holes', () => {
    const chunk = chunkOf({ '10,10': { kind: 'hole' }, '11,10': { kind: 'hole' } });
    expect(tileAt(chunk, 10, 10)?.edges.sort()).toEqual(['ne', 'nw', 'sw']);
    expect(tileAt(chunk, 11, 10)?.kind).toBe('hole');
  });
});

// ADR 0007 test 35
describe('chunksToDraw', () => {
  const grid: Grid = { width: 160, height: 160, wraps: true };

  it('draws the chunks that came, then again the shown ones beside them, each once', () => {
    const shown = new Set(['1,1', '3,3']);
    const isShown = (key: { cx: number; cy: number }) => shown.has(`${key.cx},${key.cy}`);
    expect(chunksToDraw([{ cx: 2, cy: 1 }, { cx: 1, cy: 2 }], grid, isShown)).toEqual([
      { cx: 2, cy: 1 },
      { cx: 1, cy: 2 },
      { cx: 1, cy: 1 },
    ]);
  });

  it('across the seams on a wrapping map, not on a bounded one', () => {
    const isShown = (key: { cx: number; cy: number }) => key.cx === 4 && key.cy === 0;
    expect(chunksToDraw([{ cx: 0, cy: 0 }], grid, isShown)).toEqual([{ cx: 0, cy: 0 }, { cx: 4, cy: 0 }]);
    expect(chunksToDraw([{ cx: 0, cy: 0 }], { ...grid, wraps: false }, isShown)).toEqual([{ cx: 0, cy: 0 }]);
  });
});

// ADR 0007 test 32
describe('waterFrame', () => {
  it('changes every 250 ms and loops after 4', () => {
    expect([0, 249, 250, 999, 1000, 1250].map((t) => waterFrame(t, 250))).toEqual([0, 0, 1, 3, 0, 1]);
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

// ADR 0008 tests 19, 20 and 21
describe('ocean', () => {
  it('is baked into the ground: a chunk all ocean has its tiles and no water sprite', () => {
    const cells: Record<string, { kind: GroundKind; height: number }> = {};
    for (let i = 0; i < 1024; i++) cells[`${i % 32},${Math.floor(i / 32)}`] = { kind: 'ocean', height: -1 };
    const { tiles, water } = layoutChunk(chunkOf(cells), heights, flatAround);
    expect(tiles).toHaveLength(1024);
    expect(tiles.every((tile) => tile.kind === 'ocean')).toBe(true);
    expect(water).toHaveLength(0);
  });

  it('has foam on its edges toward cells that are not ocean, not toward ocean nor a chunk not held', () => {
    const chunk = chunkOf({ '5,5': { kind: 'ocean', height: -1 }, '4,5': { kind: 'ocean', height: -1 }, '31,0': { kind: 'ocean', height: -1 } });
    expect(tileAt(chunk, 5, 5, notHeld)?.edges.sort()).toEqual(['ne', 'se', 'sw']);
    expect(tileAt(chunk, 31, 0, notHeld)?.edges.sort(), 'the chunk past its corner is not held').toEqual(['nw', 'sw']);
  });

  it('a grass cell at 0 shows a face toward ocean in front of it, none toward ocean behind it', () => {
    const chunk = chunkOf({ '11,12': { kind: 'ocean', height: -1 }, '9,8': { kind: 'ocean', height: -1 } });
    expect(tileAt(chunk, 11, 11)?.faces).toEqual([{ side: 'left', levels: 1 }]);
    expect(tileAt(chunk, 9, 9)?.faces, 'the ocean is behind it').toEqual([]);
    expect(faceLevels(0, { kind: 'ocean', height: -1 })).toBe(1);
  });
});

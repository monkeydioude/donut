import { describe, expect, it } from 'vitest';
import { createIsoProjection } from '../iso/projection';
import type { CellTerrain, Chunk } from '../map/chunks';
import type { GroundKind } from '../map/worldMap';
import type { Grid } from '../map/grid';
import { chunksToDraw, drawnChunkOrigin, faceLevels, faceMaterial, layoutChunk, pickObject, waterFrame, type DrawnObject, type TerrainAround } from './chunkView';
import type { ObjectGraphic } from './tileGraphics';

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
      height: 0,
      local: { x: 32, y: 16 },
      depth: 1,
      faces: [],
      edges: [],
      rims: [],
      rimCorners: [],
      block: null,
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

  it('draws the chunks that came, then again the shown ones around them, diagonals included, each once (ADR 0011)', () => {
    const shown = new Set(['1,1', '3,3', '3,0', '4,4']);
    const isShown = (key: { cx: number; cy: number }) => shown.has(`${key.cx},${key.cy}`);
    expect(chunksToDraw([{ cx: 2, cy: 1 }, { cx: 1, cy: 2 }], grid, isShown)).toEqual([
      { cx: 2, cy: 1 },
      { cx: 1, cy: 2 },
      { cx: 1, cy: 1 },
      { cx: 3, cy: 0 },
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

// ADR 0009 test 13
describe('mountains and glaciers', () => {
  it('are baked into the ground, with edges toward cells of another kind only', () => {
    const chunk = chunkOf({ '5,5': { kind: 'mountain', height: 3 }, '6,5': { kind: 'mountain', height: 3 }, '5,6': { kind: 'glacier' }, '5,7': { kind: 'glacier' } });
    const { tiles, water } = layoutChunk(chunk, heights, flatAround);
    expect(water).toHaveLength(0);
    expect(tileAt(chunk, 5, 5)?.kind).toBe('mountain');
    expect(tileAt(chunk, 5, 5)?.edges, 'toward the grass behind and the glacier in front: not the mountain beside').toEqual(['ne', 'nw', 'sw']);
    expect(tileAt(chunk, 5, 6)?.edges, 'toward the mountain behind and the grass on both sides: not the glacier in front').toEqual(['ne', 'nw', 'se']);
    expect(tiles.filter((t) => t.kind === 'glacier')).toHaveLength(2);
  });

  it('show faces toward a lower front neighbor, in their own material', () => {
    const chunk = chunkOf({ '5,5': { kind: 'mountain', height: 3 }, '6,5': { height: 2 }, '5,6': { height: 2 }, '8,8': { kind: 'glacier', height: 1 } });
    expect(tileAt(chunk, 5, 5)?.faces).toEqual([
      { side: 'left', levels: 1 },
      { side: 'right', levels: 1 },
    ]);
    expect(tileAt(chunk, 8, 8)?.faces).toEqual([
      { side: 'left', levels: 1 },
      { side: 'right', levels: 1 },
    ]);
    expect(faceMaterial('mountain')).toBe('stone');
    expect(faceMaterial('glacier')).toBe('ice');
    expect(faceMaterial('bridge')).toBe('wood');
    expect(faceMaterial('grass')).toBe('earth');
  });
});

// ADR 0011 test 10
describe('mountain and glacier blocks', () => {
  /** The block of a cell at (5, 5), with its neighbors given by their offset from it: "dcol,drow". */
  const blockAt = (kind: GroundKind, height: number, around: Record<string, { kind?: GroundKind; height?: number }>, outside = flatAround) => {
    const cells: Record<string, { kind?: GroundKind; height?: number }> = { '5,5': { kind, height } };
    for (const [offset, cell] of Object.entries(around)) {
      const [dc, dr] = offset.split(',').map(Number) as [number, number];
      cells[`${5 + dc},${5 + dr}`] = cell;
    }
    return tileAt(chunkOf(cells), 5, 5, outside)?.block;
  };
  const rock = (height: number) => ({ kind: 'mountain' as const, height });

  it('rise each back and side corner to the highest cell of their kind touching it behind or beside', () => {
    expect(blockAt('mountain', 1, { '-1,-1': rock(3) })?.shape, 'straight above on screen: the back corner').toEqual({ n: 2, w: 0, e: 0, peak: false });
    expect(blockAt('mountain', 1, { '-1,0': rock(2) })?.shape, 'the column before: its 2 corners').toEqual({ n: 1, w: 1, e: 0, peak: false });
    expect(blockAt('mountain', 1, { '0,-1': rock(2) })?.shape, 'the row before: its 2 corners').toEqual({ n: 1, w: 0, e: 1, peak: false });
    expect(blockAt('mountain', 1, { '-1,1': rock(3) })?.shape, 'beside on the left: the left corner').toEqual({ n: 0, w: 2, e: 0, peak: false });
    expect(blockAt('mountain', 1, { '-1,-1': rock(2), '-1,0': rock(4) })?.shape, 'the highest of them').toEqual({ n: 3, w: 3, e: 0, peak: false });
  });

  it('rise nothing toward another kind, a lower cell, a cell in front or a cell not held', () => {
    const flat = { n: 0, w: 0, e: 0 };
    expect(blockAt('mountain', 1, { '-1,-1': { kind: 'glacier', height: 3 } })?.shape).toMatchObject(flat);
    expect(blockAt('mountain', 3, { '-1,-1': rock(1), '1,1': rock(5) })?.shape, 'lower behind, higher in front').toEqual({ ...flat, peak: false });
    expect(blockAt('mountain', 1, {}, notHeld)?.shape).toMatchObject(flat);
  });

  it('a mountain with no higher mountain around is a peak; a glacier never is', () => {
    expect(blockAt('mountain', 3, { '1,0': rock(2), '0,1': rock(3) })?.shape.peak).toBe(true);
    expect(blockAt('glacier', 3, {})?.shape.peak).toBe(false);
  });

  it('glaciers rise toward the glaciers behind them the same way', () => {
    expect(blockAt('glacier', 1, { '-1,0': { kind: 'glacier', height: 2 } })?.shape).toEqual({ n: 1, w: 1, e: 0, peak: false });
  });

  it('are outlined toward another kind, and toward a lower cell of their kind behind them only', () => {
    expect(blockAt('mountain', 3, { '-1,0': rock(2), '1,0': rock(2), '0,-1': rock(3), '0,1': rock(3) })?.outline).toEqual(['nw']);
    expect(blockAt('mountain', 3, { '-1,0': rock(3), '1,0': rock(3), '0,1': rock(3) })?.outline, 'grass behind').toEqual(['ne']);
  });

  it('a mountain at +6 has an icy top, at +5 not', () => {
    expect(blockAt('mountain', 6, {})?.iced).toBe(true);
    expect(blockAt('mountain', 5, {})?.iced).toBe(false);
    expect(blockAt('glacier', 6, {})?.iced).toBe(false);
  });

  it('other kinds have none', () => {
    expect(tileAt(chunkOf(), 5, 5)?.block).toBeNull();
  });
});

// ADR 0012 test 28
describe('picking a dome by its pixels', () => {
  /** A 10 × 10 sprite whose pivot is its bottom middle, opaque on its upper half only. */
  const graphic: ObjectGraphic = {
    texture: undefined as never,
    pivot: { x: 5, y: 10 },
    opaqueAt: (x, y) => x >= 0 && x < 10 && y >= 0 && y < 5,
  };
  const dome = (x: number, zIndex: number, col: number): DrawnObject => ({ position: { x, y: 100 }, zIndex, graphic, cell: { col, row: 7 } });

  it('the pointer over its opaque pixels picks its cell', () => {
    expect(pickObject([dome(50, 1, 4)], { x: 50, y: 92 })).toEqual({ col: 4, row: 7 });
  });

  it('over a transparent pixel, or off the sprite, nothing: the ground behind is picked', () => {
    expect(pickObject([dome(50, 1, 4)], { x: 50, y: 97 })).toBeNull();
    expect(pickObject([dome(50, 1, 4)], { x: 70, y: 92 })).toBeNull();
  });

  it('the frontmost of two overlapping domes wins', () => {
    expect(pickObject([dome(50, 3, 4), dome(52, 5, 9)], { x: 51, y: 92 })).toEqual({ col: 9, row: 7 });
  });
});

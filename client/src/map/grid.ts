import type { TilePoint } from '../iso/projection';

/** The side of a chunk, in cells: the server sends maps 32 x 32 cells at a time (ADR 0003). */
export const CHUNK_SIZE = 32;
export const CELLS_PER_CHUNK = CHUNK_SIZE * CHUNK_SIZE;

/** A map's shape: its size in cells, and whether it wraps around both ways (the donut's surface) or has edges. */
export interface Grid {
  width: number;
  height: number;
  wraps: boolean;
}

/** Chunk (cx, cy) holds the cells from (32·cx, 32·cy) to (32·cx + 31, 32·cy + 31). */
export interface ChunkKey {
  cx: number;
  cy: number;
}

/** The cell within the map: wrapped on a wrapping grid, null outside a bounded one. */
export function normalize(grid: Grid, cell: TilePoint): TilePoint | null {
  if (grid.wraps) return { col: mod(cell.col, grid.width), row: mod(cell.row, grid.height) };
  const inside = cell.col >= 0 && cell.row >= 0 && cell.col < grid.width && cell.row < grid.height;
  return inside ? cell : null;
}

/**
 * The copy of a cell nearest to an anchor. The client draws in coordinates that don't wrap, around your player:
 * everything is drawn at its copy nearest to you, so the seams never show. On a bounded grid, the cell itself.
 */
export function nearestCopy(grid: Grid, cell: TilePoint, anchor: TilePoint): TilePoint {
  if (!grid.wraps) return cell;
  return {
    col: anchor.col + shortest(cell.col - anchor.col, grid.width),
    row: anchor.row + shortest(cell.row - anchor.row, grid.height),
  };
}

/** The chunk holding a cell within the map. */
export function chunkOf(cell: TilePoint): ChunkKey {
  return { cx: Math.floor(cell.col / CHUNK_SIZE), cy: Math.floor(cell.row / CHUNK_SIZE) };
}

/** The chunk's first cell, its top corner. */
export function chunkOrigin(key: ChunkKey): TilePoint {
  return { col: key.cx * CHUNK_SIZE, row: key.cy * CHUNK_SIZE };
}

/** The 4 chunks beside a chunk, across the seams on a wrapping grid; past a bounded grid's edges, none. */
export function chunksBeside(grid: Grid, key: ChunkKey): ChunkKey[] {
  const columns = Math.ceil(grid.width / CHUNK_SIZE);
  const rows = Math.ceil(grid.height / CHUNK_SIZE);
  const beside = [
    { cx: key.cx + 1, cy: key.cy },
    { cx: key.cx - 1, cy: key.cy },
    { cx: key.cx, cy: key.cy + 1 },
    { cx: key.cx, cy: key.cy - 1 },
  ];
  if (grid.wraps) return beside.map(({ cx, cy }) => ({ cx: mod(cx, columns), cy: mod(cy, rows) }));
  return beside.filter(({ cx, cy }) => cx >= 0 && cy >= 0 && cx < columns && cy < rows);
}

/** A chunk's key as a string, to index chunks by. */
export function chunkId(key: ChunkKey): string {
  return `${key.cx},${key.cy}`;
}

/** The difference d made as short as can be on an axis of `size` that wraps: between −size/2 and size/2. */
function shortest(d: number, size: number): number {
  const wrapped = mod(d, size);
  return wrapped > size / 2 ? wrapped - size : wrapped;
}

/** The remainder of a / b, never negative. */
function mod(a: number, b: number): number {
  return ((a % b) + b) % b;
}

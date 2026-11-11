import { CHUNK_SIZE, chunkId, type ChunkKey, type Grid } from '../map/grid';

/** The chunks held at least once this session, by id (ADR 0005). Forgetting a chunk doesn't unexplore it. */
export type Exploration = ReadonlySet<string>;

export const noExploration: Exploration = new Set();

/** The exploration with one more chunk: the same one when it was already explored. */
export function explore(explored: Exploration, key: ChunkKey): Exploration {
  const id = chunkId(key);
  return explored.has(id) ? explored : new Set([...explored, id]);
}

export function isExplored(explored: Exploration, key: ChunkKey): boolean {
  return explored.has(chunkId(key));
}

/** How many chunks the map has. */
export function chunkCount(grid: Grid): number {
  return Math.ceil(grid.width / CHUNK_SIZE) * Math.ceil(grid.height / CHUNK_SIZE);
}

/** The share of the map's chunks explored, from 0 to 1. */
export function exploredShare(explored: Exploration, grid: Grid): number {
  return Math.min(1, explored.size / chunkCount(grid));
}

/** The share as shown under the map: "1.8% explored". */
export function exploredText(share: number): string {
  return `${(share * 100).toFixed(1)}% explored`;
}

import type { TilePoint } from '../iso/projection';
import { CHUNK_SIZE, chunkId, chunkOf, chunkOrigin, type ChunkKey } from './grid';
import type { GroundKind, MapObject } from './worldMap';

/** 32 x 32 cells of the map, as the server sent them: each cell's ground, height and region, row by row. */
export interface Chunk {
  key: ChunkKey;
  version: number;
  ground: readonly GroundKind[];
  height: readonly number[];
  region: readonly number[];
  objects: readonly MapObject[];
}

/** The chunks your connection holds, by chunk id. */
export type HeldChunks = ReadonlyMap<string, Chunk>;

export function emptyChunks(): HeldChunks {
  return new Map();
}

/** The chunks held after a chunk comes in. The previous ones are left untouched. */
export function withChunk(held: HeldChunks, chunk: Chunk): HeldChunks {
  return new Map(held).set(chunkId(chunk.key), chunk);
}

/** The chunks held after one is forgotten. */
export function withoutChunk(held: HeldChunks, key: ChunkKey): HeldChunks {
  const next = new Map(held);
  next.delete(chunkId(key));
  return next;
}

/** The chunk holding a cell within the map, if it is held. */
export function chunkAt(held: HeldChunks, cell: TilePoint): Chunk | undefined {
  return held.get(chunkId(chunkOf(cell)));
}

export function isHeld(held: HeldChunks, cell: TilePoint): boolean {
  return chunkAt(held, cell) !== undefined;
}

/** A cell's ground, null when its chunk isn't held. */
export function groundAt(held: HeldChunks, cell: TilePoint): GroundKind | null {
  const chunk = chunkAt(held, cell);
  return chunk ? (chunk.ground[localIndex(chunk.key, cell)] ?? null) : null;
}

/** A cell's region id (0 for none), null when its chunk isn't held. */
export function regionAt(held: HeldChunks, cell: TilePoint): number | null {
  const chunk = chunkAt(held, cell);
  return chunk ? (chunk.region[localIndex(chunk.key, cell)] ?? null) : null;
}

/** A cell's position in its chunk's arrays. */
export function localIndex(key: ChunkKey, cell: TilePoint): number {
  const origin = chunkOrigin(key);
  return (cell.row - origin.row) * CHUNK_SIZE + (cell.col - origin.col);
}

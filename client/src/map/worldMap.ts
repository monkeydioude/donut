import type { Grid } from './grid';

/** What the ground of a cell is made of. `void` is no ground at all: not drawn, not walkable. */
export type GroundKind = 'grass' | 'void';
/** The ground kinds that are drawn. */
export type DrawnGroundKind = Exclude<GroundKind, 'void'>;
export type ObjectKind = 'rock';

export const GROUND_KINDS: readonly GroundKind[] = ['grass', 'void'];
export const OBJECT_KINDS: readonly ObjectKind[] = ['rock'];

/** Something standing on a cell of the map. */
export interface MapObject {
  id: string;
  kind: ObjectKind;
  col: number;
  row: number;
}

/** A named area of the map. Its cells hold its id. */
export interface Region {
  id: number;
  name: string;
  biome: string;
}

/** A map as the back describes it: its size and regions. Its cells come over the websocket, chunk by chunk. */
export interface MapDescription {
  slug: string;
  name: string;
  width: number;
  height: number;
  wraps: boolean;
  chunkSize: number;
  regions: readonly Region[];
}

export function gridOf(map: MapDescription): Grid {
  return { width: map.width, height: map.height, wraps: map.wraps };
}

/** The name of a region, null for none (id 0) or an unknown one. */
export function regionName(map: MapDescription, id: number): string | null {
  return map.regions.find((region) => region.id === id)?.name ?? null;
}

import type { TilePoint } from '../iso/projection';
import type { Grid } from './grid';

/**
 * What the ground of a cell is made of (ADR 0007, ADR 0008, ADR 0009). `void` is no ground at all: not drawn, not
 * walkable. Water, the ocean, holes, mountains and glaciers are drawn but not walkable; grass and bridges are walkable.
 */
export type GroundKind = 'grass' | 'water' | 'hole' | 'bridge' | 'ocean' | 'mountain' | 'glacier' | 'floor' | 'wall' | 'void';
/** The ground kinds that are drawn. */
export type DrawnGroundKind = Exclude<GroundKind, 'void'>;
/** The ground kinds baked into a chunk's ground texture: water has a layer of its own, under it. The ocean is still, so it's baked (ADR 0008). */
export type BakedGroundKind = Exclude<DrawnGroundKind, 'water'>;
/** What stands on the map: rocks, and domes, a city seen from the world map (ADR 0012). */
export type ObjectKind = 'rock' | 'dome';

export const GROUND_KINDS: readonly GroundKind[] = ['grass', 'water', 'hole', 'bridge', 'ocean', 'mountain', 'glacier', 'floor', 'wall', 'void'];
/** A cell's height, in levels (ADR 0007): above +3, only mountains and glaciers (ADR 0011). */
export const MIN_HEIGHT = -3;
export const MAX_HEIGHT = 6;
export const OBJECT_KINDS: readonly ObjectKind[] = ['rock', 'dome'];
/** A dome covers 2 × 2 cells from its back cell, its lowest column and row (ADR 0012). */
export const DOME_CELLS = 2;

/** Something standing on the map: a rock on its cell, a dome on its back cell. */
export interface MapObject {
  id: string;
  kind: ObjectKind;
  col: number;
  row: number;
  /** The map a dome leads to; rocks have none. */
  leadsTo?: string;
}

/** The cells an object covers, from its back cell, row by row. */
export function footprint(object: { kind: ObjectKind; col: number; row: number }): TilePoint[] {
  const side = object.kind === 'dome' ? DOME_CELLS : 1;
  const cells: TilePoint[] = [];
  for (let dr = 0; dr < side; dr++) {
    for (let dc = 0; dc < side; dc++) cells.push({ col: object.col + dc, row: object.row + dr });
  }
  return cells;
}

/**
 * Where an object's sprite sits, in cells, from where its back cell is drawn: the middle of its footprint. And the
 * cell it's sorted by among players and rocks: its front cell, so a player behind a dome is hidden by it.
 */
export function objectPlacement(kind: ObjectKind, back: TilePoint): { at: TilePoint; sortedBy: TilePoint } {
  const side = kind === 'dome' ? DOME_CELLS : 1;
  const half = (side - 1) / 2;
  return {
    at: { col: back.col + half, row: back.row + half },
    sortedBy: { col: back.col + side - 1, row: back.row + side - 1 },
  };
}

/** A named area of the map. Its cells hold its id. */
export interface Region {
  id: number;
  name: string;
  biome: string;
}

/** Where a city stands (ADR 0012): its dome's back cell on another map, with that map's size. */
export interface Place {
  map: string;
  col: number;
  row: number;
  width: number;
  height: number;
}

/**
 * A map as the back describes it: its size and regions, and for a city where it stands. `welcome` carries it (ADR
 * 0012). Its cells come over the websocket, chunk by chunk.
 */
export interface MapDescription {
  slug: string;
  name: string;
  width: number;
  height: number;
  wraps: boolean;
  chunkSize: number;
  regions: readonly Region[];
  standsOn: Place | null;
}

export function gridOf(map: MapDescription): Grid {
  return { width: map.width, height: map.height, wraps: map.wraps };
}

/** The name of a region, null for none (id 0) or an unknown one. */
export function regionName(map: MapDescription, id: number): string | null {
  return map.regions.find((region) => region.id === id)?.name ?? null;
}

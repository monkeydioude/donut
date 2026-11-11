import { Container } from 'pixi.js';
import { groundCenterAt, type Heights } from '../iso/heights';
import type { TilePoint } from '../iso/projection';
import { objectPlacement, type ObjectKind } from '../map/worldMap';

export interface WorldView {
  /** The water, the ground over it, the overlay on the ground, the entities above. */
  root: Container;
  /** The river's water, one sprite per cell, under the ground: a bank in front hides the water behind it (ADR 0007). */
  water: Container;
  /** The chunks' ground, one texture per chunk, sorted back to front. */
  ground: Container;
  /** Drawn on the ground, under the entities: tile borders and paths. */
  overlay: Container;
  /** Rocks and players, sorted together by depth: a player walks behind a rock, in front of another. */
  entities: Container;
}

/** The world's layers, empty: chunks fill them as they come. */
export function createWorldView(): WorldView {
  const water = new Container();
  const ground = new Container({ sortableChildren: true });
  const overlay = new Container();
  const entities = new Container({ sortableChildren: true });
  const root = new Container();
  root.addChild(water, ground, overlay, entities);
  return { root, water, ground, overlay, entities };
}

/** Puts a sprite on a cell at its height, drawn after everything behind it. */
export function placeOnCell(sprite: Container, cell: TilePoint, heights: Heights, height: number): void {
  positionOnCell(sprite, cell, heights, height);
  sprite.zIndex = heights.projection.depth(cell);
}

/**
 * Puts an object's sprite on its footprint at its height, its back cell being drawn at `back`: a rock on its cell, a
 * dome on the middle of its 2 × 2, sorted by its front cell (ADR 0012).
 */
export function placeObject(sprite: Container, kind: ObjectKind, back: TilePoint, heights: Heights, height: number): void {
  const { at, sortedBy } = objectPlacement(kind, back);
  positionOnCell(sprite, at, heights, height);
  sprite.zIndex = heights.projection.depth(sortedBy);
}

/** Puts a sprite on a cell at its height, without touching the draw order: Pixi sorts a layer once a child has a zIndex. */
export function positionOnCell(sprite: Container, cell: TilePoint, heights: Heights, height: number): void {
  const at = groundCenterAt(heights, cell, height);
  sprite.position.set(at.x, at.y);
}

import { Container, Sprite } from 'pixi.js';
import type { IsoProjection, TilePoint } from '../iso/projection';
import type { SpriteGraphic } from './tileGraphics';

export interface WorldView {
  /** The ground below, the overlay on it, the entities above. */
  root: Container;
  /** The chunks' ground, one texture per chunk, sorted back to front. */
  ground: Container;
  /** Drawn on the ground, under the entities: tile borders and paths. */
  overlay: Container;
  /** Rocks and players, sorted together by depth: a player walks behind a rock, in front of another. */
  entities: Container;
}

/** The world's layers, empty: chunks fill them as they come. */
export function createWorldView(): WorldView {
  const ground = new Container({ sortableChildren: true });
  const overlay = new Container();
  const entities = new Container({ sortableChildren: true });
  const root = new Container();
  root.addChild(ground, overlay, entities);
  return { root, ground, overlay, entities };
}

export function createCellSprite(graphic: SpriteGraphic, cell: TilePoint, projection: IsoProjection): Sprite {
  const sprite = new Sprite(graphic.texture);
  sprite.pivot.set(graphic.pivot.x, graphic.pivot.y);
  placeOnCell(sprite, cell, projection);
  return sprite;
}

/** Puts a sprite on a cell, drawn after everything behind it. */
export function placeOnCell(sprite: Container, cell: TilePoint, projection: IsoProjection): void {
  positionOnCell(sprite, cell, projection);
  sprite.zIndex = projection.depth(cell);
}

/** Puts a sprite on a cell without touching the draw order: Pixi sorts a layer as soon as a child has a zIndex. */
export function positionOnCell(sprite: Container, cell: TilePoint, projection: IsoProjection): void {
  const screen = projection.toScreen(cell);
  sprite.position.set(screen.x, screen.y);
}

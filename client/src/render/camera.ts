import { groundCenterAt, pickOnHeights, type HeightOf, type Heights } from '../iso/heights';
import type { ScreenPoint, TilePoint } from '../iso/projection';
import { normalize, type Grid } from '../map/grid';

export interface Viewport {
  width: number;
  height: number;
}

/** Where the world is drawn on the page: its origin, and how many page pixels an art pixel takes. */
export interface WorldPlacement {
  origin: ScreenPoint;
  zoom: number;
}

/** World position that puts a world point (your player, as drawn) in the middle of the viewport, on whole pixels. */
export function followPosition(target: ScreenPoint, viewport: Viewport, zoom: number): ScreenPoint {
  return {
    x: Math.round(viewport.width / 2 - target.x * zoom),
    y: Math.round(viewport.height / 2 - target.y * zoom),
  };
}

/**
 * The map cell under a point of the canvas: the topmost drawn cell there, by its top or its cliff face (ADR 0007), kept
 * within the map (it wraps on a wrapping map). Null outside a bounded map.
 */
export function cellAt(point: ScreenPoint, world: WorldPlacement, heights: Heights, heightOf: HeightOf, grid: Grid): TilePoint | null {
  const drawn = pickOnHeights(heights, { x: (point.x - world.origin.x) / world.zoom, y: (point.y - world.origin.y) / world.zoom }, heightOf);
  return normalize(grid, drawn);
}

/** Where a tile's ground diamond is on the canvas, at its height: its left and right corners, and its middle height. */
export function tileOnCanvas(cell: TilePoint, height: number, world: WorldPlacement, heights: Heights, tileWidth: number) {
  const center = groundCenterAt(heights, cell, height);
  const x = world.origin.x + center.x * world.zoom;
  const halfWidth = (tileWidth / 2) * world.zoom;
  return { left: x - halfWidth, right: x + halfWidth, y: world.origin.y + center.y * world.zoom };
}

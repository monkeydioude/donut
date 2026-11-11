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

/** Finds the map cell of an object drawn at a world point, by its pixels; null when there's none. */
export type ObjectAt = (point: ScreenPoint) => TilePoint | null;

/**
 * The map cell under a point of the canvas: a dome drawn there, by its pixels (ADR 0012), else the topmost drawn cell,
 * by its top or its cliff face (ADR 0007), kept within the map on a wrapping map. Past a bounded map's edges, the cell
 * there: a click on it leaves a city (ADR 0012).
 */
export function cellAt(point: ScreenPoint, world: WorldPlacement, heights: Heights, heightOf: HeightOf, grid: Grid, objectAt: ObjectAt = () => null): TilePoint {
  const at = { x: (point.x - world.origin.x) / world.zoom, y: (point.y - world.origin.y) / world.zoom };
  const drawn = objectAt(at) ?? pickOnHeights(heights, at, heightOf);
  return normalize(grid, drawn) ?? drawn;
}

/** Where a tile's ground diamond is on the canvas, at its height: its left and right corners, and its middle height. */
export function tileOnCanvas(cell: TilePoint, height: number, world: WorldPlacement, heights: Heights, tileWidth: number) {
  const center = groundCenterAt(heights, cell, height);
  const x = world.origin.x + center.x * world.zoom;
  const halfWidth = (tileWidth / 2) * world.zoom;
  return { left: x - halfWidth, right: x + halfWidth, y: world.origin.y + center.y * world.zoom };
}

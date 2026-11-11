import type { ScreenPoint, TilePoint } from '../iso/projection';
import { CHUNK_SIZE, type Grid } from '../map/grid';
import { gridOf, type MapDescription } from '../map/worldMap';

/** The ring the donut mark moves on, in the donut sprite's pixels: its center, outer and inner radii. */
export interface DonutRing {
  center: ScreenPoint;
  outer: number;
  inner: number;
}

/** A square near an equator; ▲ on the top face; ▼ on the bottom face, hidden from above (ADR 0005). */
export type DonutShape = 'square' | 'up' | 'down';

export interface Size {
  width: number;
  height: number;
}

/** Your marks blink: shown, then hidden, this long each. */
export const BLINK_MS = 500;

/** How far from an equator, in rows, the mark is a square: a chunk. */
const EQUATOR_BAND = CHUNK_SIZE;

/** The scales the map can be shown at, biggest first, and the smallest, used when none fits: up to ×8 for a city's
 * small map (ADR 0012). */
const MAP_SCALES = [8, 4, 2, 1, 0.5] as const;
const SMALLEST_SCALE = 0.5;

/** Where a cell's mark goes on the rectangle: the middle of its pixel, at the map's scale. */
export function rectangleMark(cell: TilePoint, scale: number): ScreenPoint {
  return { x: (cell.col + 0.5) * scale, y: (cell.row + 0.5) * scale };
}

/**
 * Where your mark goes on the donut seen from above. Your column is the angle around the ring, column 0 at the top,
 * going clockwise. Your row is the distance from the center: the outer edge at row 0, the inner edge at row
 * height / 2, back to the outer edge at the last row.
 */
export function donutMark(cell: TilePoint, grid: Grid, ring: DonutRing): { at: ScreenPoint; shape: DonutShape } {
  const around = (2 * Math.PI * cell.col) / grid.width;
  const tube = (2 * Math.PI * cell.row) / grid.height;
  const middle = (ring.outer + ring.inner) / 2;
  const radius = middle + ((ring.outer - ring.inner) / 2) * Math.cos(tube);
  const at = { x: ring.center.x + radius * Math.sin(around), y: ring.center.y - radius * Math.cos(around) };
  return { at, shape: donutShape(cell.row, grid.height) };
}

/** A square within a chunk of an equator (row 0 or height / 2, across the seam); ▲ before height / 2; ▼ after. */
export function donutShape(row: number, height: number): DonutShape {
  const half = height / 2;
  const fromOuter = Math.min(row, height - row);
  if (fromOuter <= EQUATOR_BAND || Math.abs(row - half) <= EQUATOR_BAND) return 'square';
  return row < half ? 'up' : 'down';
}

/** Whether your marks show at that time: 0.5 s shown, 0.5 s hidden. */
export function blinkOn(now: number): boolean {
  return Math.floor(now / BLINK_MS) % 2 === 0;
}

/** The biggest scale at which the map fits the room, from ×8 down to ×½, and ×½ when nothing fits. */
export function mapScale(map: Size, room: Size): number {
  const fits = (scale: number) => map.width * scale <= room.width && map.height * scale <= room.height;
  return MAP_SCALES.find(fits) ?? SMALLEST_SCALE;
}

/** The cell under a point of the map's rectangle, at the map's scale: the inverse of `rectangleMark`; null off it. */
export function mapCellAt(point: ScreenPoint, scale: number, map: Size): TilePoint | null {
  const col = Math.floor(point.x / scale);
  const row = Math.floor(point.y / scale);
  const inside = col >= 0 && row >= 0 && col < map.width && row < map.height;
  return inside ? { col, row } : null;
}

/**
 * Where you are on the donut (ADR 0012): your cell, on a map that wraps; in a city, the cell its dome stands on, on the
 * world; nowhere on a map that doesn't wrap and stands nowhere.
 */
export function donutPlace(map: MapDescription, you: TilePoint | null): { cell: TilePoint; grid: Grid } | null {
  if (map.wraps) return you && { cell: you, grid: gridOf(map) };
  const place = map.standsOn;
  return place && { cell: { col: place.col, row: place.row }, grid: { width: place.width, height: place.height, wraps: true } };
}

/** The line under the map's name: your region's, left out when it's the map's name, as in a city (ADR 0012). */
export function regionLine(mapName: string, region: string | null): string {
  return region === null || region === mapName ? '' : region;
}

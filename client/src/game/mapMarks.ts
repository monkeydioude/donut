import type { ScreenPoint, TilePoint } from '../iso/projection';
import { CHUNK_SIZE, type Grid } from '../map/grid';

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

/** The scales the map can be shown at, biggest first, and the smallest, used when none fits. */
const MAP_SCALES = [2, 1, 0.5] as const;
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

/** The biggest scale at which the map fits the room: ×2, ×1 or ×½, and ×½ when nothing fits. */
export function mapScale(map: Size, room: Size): number {
  const fits = (scale: number) => map.width * scale <= room.width && map.height * scale <= room.height;
  return MAP_SCALES.find(fits) ?? SMALLEST_SCALE;
}

import { MAX_HEIGHT, MIN_HEIGHT } from '../map/worldMap';
import type { IsoProjection, ScreenPoint, TilePoint } from './projection';

/** A cell's height in levels, for a drawn cell. 0 where it isn't known. */
export type HeightOf = (cell: TilePoint) => number;

/** How a height is drawn: each level up draws a cell `levelPx` art pixels higher (ADR 0007). */
export interface Heights {
  projection: IsoProjection;
  levelPx: number;
}

/** Where a cell's ground center is drawn, at its height. */
export function groundCenterAt({ projection, levelPx }: Heights, cell: TilePoint, height: number): ScreenPoint {
  const flat = projection.toScreen(cell);
  return { x: flat.x, y: flat.y - height * levelPx };
}

/** How finely picking goes down the levels: a quarter of an art pixel at a time. */
const PICK_STEPS_PER_PX = 4;

/**
 * The topmost cell drawn at a world point. A cell shows as a column, from its top down to the lowest level: its faces,
 * and under them what the cells in front hide. Going down from the highest level, finely, it takes the cell whose
 * ground at that level would hold the point: the first one standing at least that high is drawn there, by its top
 * or its face. Cells further down the screen are in front, so that first one is the frontmost.
 */
export function pickOnHeights({ projection, levelPx }: Heights, point: ScreenPoint, heightOf: HeightOf): TilePoint {
  const stepsPerLevel = levelPx * PICK_STEPS_PER_PX;
  for (let step = 0; step < (MAX_HEIGHT - MIN_HEIGHT) * stepsPerLevel; step++) {
    const level = MAX_HEIGHT - step / stepsPerLevel;
    const cell = projection.toTile({ x: point.x, y: point.y + level * levelPx });
    if (heightOf(cell) >= level) return cell;
  }
  // every cell stands at least at the lowest level
  return projection.toTile({ x: point.x, y: point.y + MIN_HEIGHT * levelPx });
}

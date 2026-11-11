import type { TilePoint } from '../iso/projection';
import { domeAt, groundAt, type HeldChunks } from '../map/chunks';
import type { Grid } from '../map/grid';
import type { CellAction } from '../net/protocol';

/**
 * What a click on a cell does (ADR 0012), by the server's own rule: a dome's cell is Enter; in a city, the void, a
 * wall, or past the map's edge is Exit; anything else is Move. A cell in a chunk not held is Move, which the server
 * answers with no path.
 */
export function actionAt(cell: TilePoint, held: HeldChunks, grid: Grid): CellAction {
  if (domeAt(held, cell)) return 'enter';
  if (!grid.wraps && isOutside(cell, held, grid)) return 'exit';
  return 'move';
}

/** The words of an action in the menu. */
export function actionLabel(action: CellAction): string {
  return { move: 'Move', enter: 'Enter', exit: 'Exit' }[action];
}

function isOutside(cell: TilePoint, held: HeldChunks, grid: Grid): boolean {
  const inside = cell.col >= 0 && cell.row >= 0 && cell.col < grid.width && cell.row < grid.height;
  if (!inside) return true;
  const ground = groundAt(held, cell);
  return ground === 'void' || ground === 'wall';
}

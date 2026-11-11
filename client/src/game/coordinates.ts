import type { TilePoint } from '../iso/projection';

/** A cell's coordinates as shown on screen, column then row like `/teleport` takes them; nothing for no cell. */
export function coordinatesText(cell: TilePoint | null): string {
  return cell ? `${cell.col}, ${cell.row}` : '';
}

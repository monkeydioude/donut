import { silent, type Log } from '../debug/log';
import type { TilePoint } from '../iso/projection';

export interface PointerHandlers {
  /** The tile under the pointer, null outside the map or off the canvas. Called only when it changes. */
  hover(cell: TilePoint | null): void;
  /** A click on a tile. */
  select(cell: TilePoint): void;
  /** A right click anywhere, or a click outside the map. */
  cancel(): void;
}

/** Canvas point (CSS pixels from its top-left) to the tile there, or null. */
export type PickCell = (point: { x: number; y: number }) => TilePoint | null;

/** Turns the mouse on the canvas into tile hovers, selections and cancels. Returns a function that stops. */
export function listenToPointer(canvas: HTMLElement, pick: PickCell, handlers: PointerHandlers, log: Log = silent): () => void {
  let hovered: TilePoint | null = null;
  const pointAt = (event: MouseEvent) => {
    const rect = canvas.getBoundingClientRect();
    return { x: event.clientX - rect.left, y: event.clientY - rect.top };
  };
  const hover = (cell: TilePoint | null) => {
    if (cell?.col === hovered?.col && cell?.row === hovered?.row) return;
    hovered = cell;
    log('hover', cell);
    handlers.hover(cell);
  };
  const onMove = (event: PointerEvent) => hover(pick(pointAt(event)));
  const onLeave = () => hover(null);
  const onClick = (event: MouseEvent) => {
    if (event.button !== 0) return;
    const point = pointAt(event);
    const cell = pick(point);
    log('click', { point, cell });
    if (cell) handlers.select(cell);
    else handlers.cancel();
  };
  const onContextMenu = (event: MouseEvent) => {
    event.preventDefault();
    log('right click: cancel');
    handlers.cancel();
  };

  canvas.addEventListener('pointermove', onMove);
  canvas.addEventListener('pointerleave', onLeave);
  canvas.addEventListener('click', onClick);
  window.addEventListener('contextmenu', onContextMenu);
  return () => {
    canvas.removeEventListener('pointermove', onMove);
    canvas.removeEventListener('pointerleave', onLeave);
    canvas.removeEventListener('click', onClick);
    window.removeEventListener('contextmenu', onContextMenu);
  };
}

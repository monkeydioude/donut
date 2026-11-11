import { coordinatesText } from '../game/coordinates';
import type { TilePoint } from '../iso/projection';

export interface CoordinatesLabel {
  /** Shows a cell's coordinates, or hides the label for no cell. Touches the DOM only when the text changes. */
  show(cell: TilePoint | null): void;
}

/** A label holding a cell's coordinates, hidden until it has one. Text only, never HTML. */
export function createCoordinatesLabel(host: HTMLElement, className: string): CoordinatesLabel {
  const label = document.createElement('div');
  label.className = className;
  label.hidden = true;
  host.appendChild(label);
  return {
    show(cell) {
      const text = coordinatesText(cell);
      if (text === label.textContent) return;
      label.textContent = text;
      label.hidden = text === '';
    },
  };
}

import { silent, type Log } from '../debug/log';

/** An item of the action menu. */
export interface MenuAction {
  label: string;
  enabled: boolean;
  choose: () => void;
}

export interface ActionMenu {
  /** Opens or updates the menu beside a tile. Buttons are kept while the labels stay, so a click is never lost. */
  show(tile: TileAnchor, actions: readonly MenuAction[]): void;
  hide(): void;
}

/** A tile on the page: the left and right corners of its diamond, and its middle height. */
export interface TileAnchor {
  left: number;
  right: number;
  y: number;
}

export interface Size {
  width: number;
  height: number;
}

const GAP = 8;
const MARGIN = 8;

/**
 * Where the menu goes: to the right of the tile, vertically centered on it. Too far right, it goes to the left
 * of the tile; then it is kept inside the viewport.
 */
export function menuPosition(tile: TileAnchor, menu: Size, viewport: Size): { left: number; top: number } {
  const fitsRight = tile.right + GAP + menu.width <= viewport.width - MARGIN;
  const left = fitsRight ? tile.right + GAP : tile.left - GAP - menu.width;
  return {
    left: clamp(left, MARGIN, viewport.width - MARGIN - menu.width),
    top: clamp(tile.y - menu.height / 2, MARGIN, viewport.height - MARGIN - menu.height),
  };
}

/** The menu: a blue window of buttons over the canvas. */
export function createActionMenu(host: HTMLElement, viewport: () => Size, log: Log = silent): ActionMenu {
  const root = document.createElement('div');
  root.className = 'menu';
  root.hidden = true;
  host.appendChild(root);
  let current: readonly MenuAction[] = [];
  let shownState = '';

  const buttonFor = (index: number) => {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'menu-item';
    button.addEventListener('click', (event) => {
      event.stopPropagation();
      log('chosen', current[index]?.label);
      current[index]?.choose();
    });
    return button;
  };

  return {
    show(tile, actions) {
      const state = actions.map((a) => `${a.label}${a.enabled ? '' : ' (off)'}`).join(', ');
      if (root.hidden || state !== shownState) log(root.hidden ? 'open' : 'update', { items: state, tile });
      shownState = state;
      if (!sameLabels(current, actions)) root.replaceChildren(...actions.map((_, i) => buttonFor(i)));
      current = actions;
      actions.forEach((action, i) => {
        const button = root.children[i] as HTMLButtonElement;
        button.textContent = action.label;
        button.disabled = !action.enabled;
      });
      root.hidden = false;
      const { left, top } = menuPosition(tile, { width: root.offsetWidth, height: root.offsetHeight }, viewport());
      root.style.left = `${left}px`;
      root.style.top = `${top}px`;
    },
    hide() {
      if (!root.hidden) log('close');
      root.hidden = true;
    },
  };
}

function sameLabels(a: readonly MenuAction[], b: readonly MenuAction[]): boolean {
  return a.length === b.length && a.every((action, i) => action.label === b[i]?.label);
}

function clamp(value: number, low: number, high: number): number {
  return Math.min(Math.max(value, low), Math.max(low, high));
}

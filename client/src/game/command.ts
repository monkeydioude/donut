import type { Cell, CellAction, ServerMessage } from '../net/protocol';

/** A tile you clicked, what a click there does, and the path to it once the server answered (ADR 0012). */
export interface Selection {
  cell: Cell;
  action: CellAction;
  /** Null while the server answers; empty when there is no path, or the server's action isn't this one. */
  path: readonly Cell[] | null;
}

/** What you point at, what you selected, and where the server walks you. */
export interface CommandState {
  hover: Cell | null;
  selection: Selection | null;
  /** The cells the server is still to walk you along. */
  walking: readonly Cell[];
}

export type CommandEvent =
  | { type: 'hover'; cell: Cell | null }
  | { type: 'select'; cell: Cell; action: CellAction }
  | { type: 'cancel' }
  /** The menu's action was chosen: the walk (or the door) is asked, the menu closes. */
  | { type: 'act' }
  | { type: 'server'; message: ServerMessage; you: string | null };

export function initialCommand(): CommandState {
  return { hover: null, selection: null, walking: [] };
}

export function applyCommand(state: CommandState, event: CommandEvent): CommandState {
  switch (event.type) {
    case 'hover':
      return { ...state, hover: event.cell };
    case 'select':
      return { ...state, selection: { cell: event.cell, action: event.action, path: null } };
    case 'cancel':
    case 'act':
      return { ...state, selection: null };
    case 'server':
      return applyServer(state, event.message, event.you);
  }
}

/** The menu's action is on once the server gave a path to the selected tile, for that action. */
export function canAct(state: CommandState): boolean {
  return (state.selection?.path?.length ?? 0) > 0;
}

/** The path to draw: the walk under way, else the selected tile's planned path. */
export function shownPath(state: CommandState): readonly Cell[] {
  return state.walking.length > 0 ? state.walking : (state.selection?.path ?? []);
}

export function sameCell(a: Cell, b: Cell): boolean {
  return a.col === b.col && a.row === b.row;
}

function applyServer(state: CommandState, message: ServerMessage, you: string | null): CommandState {
  switch (message.type) {
    case 'planned':
      // an answer to an older click is dropped; one for another action than the menu's leaves it off
      return state.selection && sameCell(state.selection.cell, message)
        ? { ...state, selection: { ...state.selection, path: message.action === state.selection.action ? message.path : [] } }
        : state;
    case 'walking':
      return { ...state, walking: message.path };
    case 'moved':
      return message.id === you ? { ...state, walking: walkedTo(state.walking, message) } : state;
    case 'stopped':
      return { ...state, walking: [] };
    case 'teleported':
      // you're elsewhere: the selected tile and its path are stale, and the walk was dropped (ADR 0010)
      return { ...state, selection: null, walking: [] };
    default:
      return state;
  }
}

/** The rest of the walk once you stand on `cell`. */
function walkedTo(walking: readonly Cell[], cell: Cell): readonly Cell[] {
  const at = walking.findIndex((step) => sameCell(step, cell));
  return at >= 0 ? walking.slice(at + 1) : walking;
}

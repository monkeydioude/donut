import type { Cell, ServerMessage } from '../net/protocol';

/** A tile you clicked, and the path to it once the server answered. */
export interface Selection {
  cell: Cell;
  /** Null while the server answers; empty when there is no path. */
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
  | { type: 'select'; cell: Cell }
  | { type: 'cancel' }
  /** Move was chosen in the menu: the walk is asked, the menu closes. */
  | { type: 'move' }
  | { type: 'server'; message: ServerMessage; you: string | null };

export function initialCommand(): CommandState {
  return { hover: null, selection: null, walking: [] };
}

export function applyCommand(state: CommandState, event: CommandEvent): CommandState {
  switch (event.type) {
    case 'hover':
      return { ...state, hover: event.cell };
    case 'select':
      return { ...state, selection: { cell: event.cell, path: null } };
    case 'cancel':
    case 'move':
      return { ...state, selection: null };
    case 'server':
      return applyServer(state, event.message, event.you);
  }
}

/** The menu's Move is on once the server gave a path to the selected tile. */
export function canMove(state: CommandState): boolean {
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
      // an answer to an older click is dropped
      return state.selection && sameCell(state.selection.cell, message)
        ? { ...state, selection: { ...state.selection, path: message.path } }
        : state;
    case 'walking':
      return { ...state, walking: message.path };
    case 'moved':
      return message.id === you ? { ...state, walking: walkedTo(state.walking, message) } : state;
    case 'stopped':
      return { ...state, walking: [] };
    default:
      return state;
  }
}

/** The rest of the walk once you stand on `cell`. */
function walkedTo(walking: readonly Cell[], cell: Cell): readonly Cell[] {
  const at = walking.findIndex((step) => sameCell(step, cell));
  return at >= 0 ? walking.slice(at + 1) : walking;
}

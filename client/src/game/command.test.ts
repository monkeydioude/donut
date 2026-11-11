import { describe, expect, it } from 'vitest';
import type { ServerMessage } from '../net/protocol';
import { applyCommand, canAct, initialCommand, shownPath, type CommandEvent, type CommandState } from './command';

const tile = { col: 4, row: 2 };
const path = [{ col: 3, row: 2 }, { col: 4, row: 2 }];
const server = (message: ServerMessage): CommandEvent => ({ type: 'server', message, you: 'me' });
const run = (...events: CommandEvent[]): CommandState => events.reduce(applyCommand, initialCommand());

// ADR 0002 test 19
describe('applyCommand', () => {
  it('hover follows the pointer', () => {
    expect(run({ type: 'hover', cell: tile }).hover).toEqual(tile);
    expect(run({ type: 'hover', cell: tile }, { type: 'hover', cell: null }).hover).toBeNull();
  });

  it('a click selects the tile and waits for its path, Move off', () => {
    const state = run({ type: 'select', cell: tile, action: 'move' });
    expect(state.selection).toEqual({ cell: tile, action: 'move', path: null });
    expect(canAct(state)).toBe(false);
    expect(shownPath(state)).toEqual([]);
  });

  it('the path for the selected tile shows and turns Move on', () => {
    const state = run({ type: 'select', cell: tile, action: 'move' }, server({ type: 'planned', ...tile, path, action: 'move' }));
    expect(state.selection?.path).toEqual(path);
    expect(canAct(state)).toBe(true);
    expect(shownPath(state)).toEqual(path);
  });

  it('an answer for an older click is ignored', () => {
    const older = { col: 1, row: 1 };
    const state = run({ type: 'select', cell: older, action: 'move' }, { type: 'select', cell: tile, action: 'move' }, server({ type: 'planned', ...older, path, action: 'move' }));
    expect(state.selection).toEqual({ cell: tile, action: 'move', path: null });
  });

  it('no path keeps Move off', () => {
    expect(canAct(run({ type: 'select', cell: tile, action: 'move' }, server({ type: 'planned', ...tile, path: [], action: 'move' })))).toBe(false);
  });

  it('Move closes the menu; the walk shows, shortens with your steps, and clears on arrival', () => {
    let state = run({ type: 'select', cell: tile, action: 'move' }, server({ type: 'planned', ...tile, path, action: 'move' }), { type: 'act' });
    expect(state.selection).toBeNull();

    state = applyCommand(state, server({ type: 'walking', path }));
    expect(shownPath(state)).toEqual(path);

    state = applyCommand(state, server({ type: 'moved', id: 'someone else', col: 3, row: 2 }));
    expect(shownPath(state), 'other players walking change nothing').toEqual(path);

    state = applyCommand(state, server({ type: 'moved', id: 'me', col: 3, row: 2 }));
    expect(shownPath(state)).toEqual([{ col: 4, row: 2 }]);

    state = applyCommand(state, server({ type: 'moved', id: 'me', col: 4, row: 2 }));
    expect(shownPath(state)).toEqual([]);
  });

  it('stopped ends the walk', () => {
    expect(run(server({ type: 'walking', path }), server({ type: 'stopped', col: 2, row: 2 })).walking).toEqual([]);
  });

  it('Cancel (or a right click) clears the selection', () => {
    expect(run({ type: 'select', cell: tile, action: 'move' }, { type: 'cancel' }).selection).toBeNull();
  });

  it('a click on another tile selects it instead', () => {
    const other = { col: 0, row: 3 };
    const state = run({ type: 'select', cell: tile, action: 'move' }, server({ type: 'planned', ...tile, path, action: 'move' }), { type: 'select', cell: other, action: 'move' });
    expect(state.selection).toEqual({ cell: other, action: 'move', path: null });
  });

  it('while walking, the walk is the path shown, even with a selection', () => {
    const state = run(server({ type: 'walking', path }), { type: 'select', cell: { col: 0, row: 0 }, action: 'move' });
    expect(shownPath(state)).toEqual(path);
  });
});

// ADR 0010 test 13
describe('a teleport', () => {
  it('clears the selection, its path and the walk', () => {
    const state = run({ type: 'select', cell: tile, action: 'move' }, server({ type: 'planned', ...tile, path, action: 'move' }), server({ type: 'walking', path }), server({ type: 'teleported', col: 9, row: 9, walkable: true }));
    expect(state.selection).toBeNull();
    expect(state.walking).toEqual([]);
    expect(shownPath(state)).toEqual([]);
  });

  // ADR 0012 test 26
  it("a planned path for another action than the menu's leaves it off", () => {
    const state = run({ type: 'select', cell: tile, action: 'enter' }, server({ type: 'planned', ...tile, path, action: 'move' }));
    expect(canAct(state)).toBe(false);
    expect(state.selection?.action).toBe('enter');
  });

  it("a planned path for the menu's Enter or Exit turns it on", () => {
    for (const action of ['enter', 'exit'] as const) {
      expect(canAct(run({ type: 'select', cell: tile, action }, server({ type: 'planned', ...tile, path, action })))).toBe(true);
    }
  });
});

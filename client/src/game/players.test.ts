import { describe, expect, it } from 'vitest';
import type { PlayerInfo } from '../net/protocol';
import { applyMessage, emptyPlayers, ownPlayer, type PlayersState } from './players';

const ann: PlayerInfo = { id: 'a', name: 'Ann', robot: 'dome-head', col: 2, row: 2 };
const bob: PlayerInfo = { id: 'b', name: 'Bob', robot: 'rusty-toy', col: 3, row: 3 };

const welcomed = (): PlayersState =>
  applyMessage(emptyPlayers(), { type: 'welcome', you: 'a', players: [ann, bob], stepMs: 250 });
const list = (state: PlayersState) => [...state.players.values()];

// ADR 0001 test 23
describe('applyMessage', () => {
  it('welcome: you, everyone on the map, and the time of a step', () => {
    const state = welcomed();
    expect(state.you).toBe('a');
    expect(list(state)).toEqual([ann, bob]);
    expect(ownPlayer(state)).toEqual(ann);
    expect(state.stepMs).toBe(250);
  });

  it('joined, moved and left follow the others', () => {
    const carl: PlayerInfo = { id: 'c', name: 'Carl', robot: '', col: 0, row: 0 };
    let state = applyMessage(welcomed(), { type: 'joined', player: carl });
    expect(list(state)).toContainEqual(carl);

    state = applyMessage(state, { type: 'moved', id: 'c', col: 1, row: 0 });
    expect(state.players.get('c')).toEqual({ ...carl, col: 1 });

    state = applyMessage(state, { type: 'left', id: 'c' });
    expect(state.players.has('c')).toBe(false);
  });

  it('stopped puts your player where the server says', () => {
    const state = applyMessage(welcomed(), { type: 'stopped', col: 4, row: 1 });
    expect(ownPlayer(state)).toEqual({ ...ann, col: 4, row: 1 });
  });

  it('paths move nobody', () => {
    const before = welcomed();
    expect(applyMessage(before, { type: 'planned', col: 4, row: 4, path: [{ col: 3, row: 3 }] })).toBe(before);
    expect(applyMessage(before, { type: 'walking', path: [{ col: 3, row: 3 }] })).toBe(before);
  });

  it('ignores moves of unknown players and leaves the previous state untouched', () => {
    const before = welcomed();
    expect(applyMessage(before, { type: 'moved', id: 'ghost', col: 0, row: 0 })).toEqual(before);
    applyMessage(before, { type: 'left', id: 'b' });
    expect(before.players.has('b')).toBe(true);
  });

  it('has no own player before the welcome', () => {
    expect(ownPlayer(emptyPlayers())).toBeNull();
    expect(applyMessage(emptyPlayers(), { type: 'stopped', col: 1, row: 1 })).toEqual(emptyPlayers());
  });
});

// ADR 0010 test 12
describe('teleported', () => {
  it('moves your player there, and a refusal changes nothing', () => {
    const state = applyMessage(welcomed(), { type: 'teleported', col: 100, row: 100, walkable: true });
    expect(ownPlayer(state)).toEqual({ ...ann, col: 100, row: 100 });
    expect(applyMessage(state, { type: 'refused', request: 'teleport', reason: 'no' })).toBe(state);
  });
});

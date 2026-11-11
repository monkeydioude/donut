import { describe, expect, it } from 'vitest';
import type { MapDescription } from '../map/worldMap';
import type { ServerMessage } from '../net/protocol';
import { cityDescription, worldDescription } from '../testing/maps';
import { arrive, initialTravel, receive, type TravelState } from './travel';

const welcome = (map: MapDescription): ServerMessage => ({ type: 'welcome', you: 'a', players: [], stepMs: 200, map });
const moved: ServerMessage = { type: 'moved', id: 'b', col: 1, row: 1 };
const forget: ServerMessage = { type: 'forget', cx: 0, cy: 0 };

function after(...messages: ServerMessage[]): TravelState {
  return messages.reduce((state, message) => receive(state, message).state, initialTravel());
}

// ADR 0012 test 27
describe('travelling between maps', () => {
  it('the first welcome builds its map', () => {
    const step = receive(initialTravel(), welcome(worldDescription()));
    expect(step.step).toEqual({ kind: 'first', map: worldDescription() });
    expect(step.state.map?.slug).toBe('world');
  });

  it('nothing comes before the first welcome', () => {
    expect(receive(initialTravel(), moved).step).toEqual({ kind: 'wait' });
  });

  it('a welcome for the map shown, and the other messages, are applied to it', () => {
    const state = after(welcome(worldDescription()));
    expect(receive(state, moved).step).toEqual({ kind: 'apply' });
    expect(receive(state, welcome(worldDescription())).step).toEqual({ kind: 'apply' });
  });

  it('a welcome for another map starts the fade; the messages meanwhile wait, and come after it, in order', () => {
    let state = after(welcome(worldDescription()));
    const leave = receive(state, welcome(cityDescription()));
    expect(leave.step).toEqual({ kind: 'leave', from: worldDescription(), to: cityDescription() });
    state = leave.state;
    for (const message of [moved, forget]) {
      const next = receive(state, message);
      expect(next.step).toEqual({ kind: 'wait' });
      state = next.state;
    }
    const arrived = arrive(state);
    expect(arrived?.map.slug).toBe('doma-1');
    expect(arrived?.messages).toEqual([welcome(cityDescription()), moved, forget]);
    expect(arrived?.state).toEqual({ map: cityDescription(), waiting: null });
  });

  it('two maps in a row while fading: the last one wins', () => {
    const state = after(welcome(worldDescription()), welcome(cityDescription()), moved, welcome(worldDescription()), forget);
    const arrived = arrive(state);
    expect(arrived?.map.slug).toBe('world');
    expect(arrived?.messages).toEqual([welcome(worldDescription()), forget]);
  });

  it('nothing to arrive at when not fading', () => {
    expect(arrive(after(welcome(worldDescription())))).toBeNull();
  });
});

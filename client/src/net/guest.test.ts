import { describe, expect, it } from 'vitest';
import type { KeyValueStore } from '../core/store';
import { brokenStore, memoryStore } from '../testing/store';
import { loadGuest } from './guest';

const ids = ['11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-222222222222'];
const robots = ['dome-head', 'rusty-toy', 'round-bot'];
/** 234 for the name, so Guest-1234; the second robot. */
const randomInt = (max: number) => (max === 9000 ? 234 : 1);
const deps = (store: KeyValueStore | null) => {
  const queue = [...ids];
  return { store, newId: () => queue.shift() ?? 'none', randomInt, robots };
};
const saved = (store: ReturnType<typeof memoryStore>) => JSON.parse(store.items.get('donut.guest') ?? 'null') as unknown;

// ADR 0001 test 25
describe('loadGuest', () => {
  it('creates a guest once, then reads it back', () => {
    const store = memoryStore();
    const first = loadGuest(deps(store));
    expect(first).toEqual({ id: ids[0], name: 'Guest-1234', robot: 'rusty-toy' });

    expect(loadGuest({ ...deps(store), newId: () => 'never used' })).toEqual(first);
  });

  it('replaces a saved guest that is not valid', () => {
    const store = memoryStore();
    store.items.set('donut.guest', JSON.stringify({ id: 'not-a-uuid', name: 'Ann', robot: 'dome-head' }));
    expect(loadGuest(deps(store)).id).toBe(ids[0]);
  });

  it('gives a guest for the session when storage fails or is missing', () => {
    expect(loadGuest(deps(brokenStore))).toEqual({ id: ids[0], name: 'Guest-1234', robot: 'rusty-toy' });
    expect(loadGuest(deps(null))).toEqual({ id: ids[0], name: 'Guest-1234', robot: 'rusty-toy' });
  });
});

// ADR 0006 tests 20 to 24
describe('the guest\'s robot', () => {
  const ann = { id: ids[1], name: 'Ann' };

  it('is picked from the list for a new guest, with randomInt, and saved', () => {
    const store = memoryStore();
    const asked: number[] = [];
    const guest = loadGuest({ ...deps(store), randomInt: (max) => (asked.push(max), max - 1) });
    expect(asked).toContain(robots.length);
    expect(guest.robot).toBe('round-bot');
    expect(saved(store)).toEqual(guest);
  });

  it('is read back unchanged when it is in the list', () => {
    const store = memoryStore();
    store.items.set('donut.guest', JSON.stringify({ ...ann, robot: 'dome-head' }));
    expect(loadGuest(deps(store))).toEqual({ ...ann, robot: 'dome-head' });
  });

  it('is given to a saved guest without one, who keeps their id and name', () => {
    const store = memoryStore();
    store.items.set('donut.guest', JSON.stringify(ann));
    expect(loadGuest(deps(store))).toEqual({ ...ann, robot: 'rusty-toy' });
    expect(saved(store)).toEqual({ ...ann, robot: 'rusty-toy' });
  });

  it('is replaced when it is not in the list anymore, and the guest keeps their id and name', () => {
    const store = memoryStore();
    store.items.set('donut.guest', JSON.stringify({ ...ann, robot: 'gone-bot' }));
    expect(loadGuest(deps(store))).toEqual({ ...ann, robot: 'rusty-toy' });
    expect(saved(store)).toEqual({ ...ann, robot: 'rusty-toy' });
  });

  it('is given without storage too', () => {
    expect(loadGuest(deps(null)).robot).toBe('rusty-toy');
  });
});

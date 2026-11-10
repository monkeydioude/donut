import { describe, expect, it } from 'vitest';
import { loadGuest, type KeyValueStore } from './guest';

function memoryStore(): KeyValueStore & { items: Map<string, string> } {
  const items = new Map<string, string>();
  return { items, getItem: (key) => items.get(key) ?? null, setItem: (key, value) => void items.set(key, value) };
}

const brokenStore: KeyValueStore = {
  getItem: () => {
    throw new Error('blocked');
  },
  setItem: () => {
    throw new Error('blocked');
  },
};

const ids = ['11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-222222222222'];
const deps = (store: KeyValueStore | null) => {
  const queue = [...ids];
  return { store, newId: () => queue.shift() ?? 'none', randomInt: () => 234 };
};

// ADR 0001 test 25
describe('loadGuest', () => {
  it('creates a guest once, then reads it back', () => {
    const store = memoryStore();
    const first = loadGuest(deps(store));
    expect(first).toEqual({ id: ids[0], name: 'Guest-1234' });

    expect(loadGuest({ ...deps(store), newId: () => 'never used' })).toEqual(first);
  });

  it('replaces a saved guest that is not valid', () => {
    const store = memoryStore();
    store.items.set('donut.guest', JSON.stringify({ id: 'not-a-uuid', name: 'Ann' }));
    expect(loadGuest(deps(store)).id).toBe(ids[0]);
  });

  it('gives a guest for the session when storage fails or is missing', () => {
    expect(loadGuest(deps(brokenStore))).toEqual({ id: ids[0], name: 'Guest-1234' });
    expect(loadGuest(deps(null))).toEqual({ id: ids[0], name: 'Guest-1234' });
  });
});

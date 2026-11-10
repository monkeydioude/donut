/** Returns a float in [0, 1). Injected wherever randomness is needed, so tests can control it. */
export type Rng = () => number;

export interface Weighted<T> {
  value: T;
  weight: number;
}

/** Seeded generator (mulberry32): the same seed always gives the same sequence. */
export function createSeededRng(seed: number): Rng {
  let state = seed >>> 0;
  return () => {
    state = (state + 0x6d2b79f5) >>> 0;
    let t = state;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/** Integer in [0, maxExclusive). */
export function randomInt(rng: Rng, maxExclusive: number): number {
  return Math.floor(rng() * maxExclusive);
}

/** Picks one item, all with the same chance. */
export function pickOne<T>(rng: Rng, items: readonly T[]): T {
  const item = items[randomInt(rng, items.length)];
  if (item === undefined) throw new Error('pickOne: no items');
  return item;
}

/** Picks one value, each with a chance proportional to its weight. */
export function pickWeighted<T>(rng: Rng, items: readonly Weighted<T>[]): T {
  const total = items.reduce((sum, item) => sum + item.weight, 0);
  let roll = rng() * total;
  for (const item of items) {
    roll -= item.weight;
    if (roll < 0) return item.value;
  }
  const last = items[items.length - 1];
  if (!last) throw new Error('pickWeighted: no items');
  return last.value;
}

/**
 * Picks one item for a cell, always the same for the same cell: a chunk drawn again looks the same.
 * `salt` keeps the picks of different layers (ground, objects) apart.
 */
export function pickForCell<T>(items: readonly T[], col: number, row: number, salt = 0): T {
  const item = items[hashCell(col, row, salt) % items.length];
  if (item === undefined) throw new Error('pickForCell: no items');
  return item;
}

/** A well-mixed unsigned 32-bit number from a cell. */
export function hashCell(col: number, row: number, salt = 0): number {
  let h = Math.imul(col, 0x27d4eb2d) ^ Math.imul(row, 0x165667b1) ^ Math.imul(salt, 0x9e3779b9);
  h = Math.imul(h ^ (h >>> 15), 0x85ebca6b);
  h = Math.imul(h ^ (h >>> 13), 0xc2b2ae35);
  return (h ^ (h >>> 16)) >>> 0;
}

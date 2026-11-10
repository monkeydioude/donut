import { describe, expect, it } from 'vitest';
import { sequenceRng } from '../testing/fakeRng';
import { createSeededRng, pickForCell, pickOne, pickWeighted, randomInt } from './rng';

const draw = (rng: () => number, count: number) => Array.from({ length: count }, rng);

describe('createSeededRng', () => {
  it('gives the same sequence for the same seed', () => {
    expect(draw(createSeededRng(7), 10)).toEqual(draw(createSeededRng(7), 10));
  });

  it('gives different sequences for different seeds', () => {
    expect(draw(createSeededRng(7), 10)).not.toEqual(draw(createSeededRng(8), 10));
  });

  it('stays in [0, 1)', () => {
    const values = draw(createSeededRng(1), 10_000);
    expect(values.every((v) => v >= 0 && v < 1)).toBe(true);
  });
});

describe('randomInt', () => {
  it('maps [0, 1) onto [0, max)', () => {
    expect(randomInt(sequenceRng(0), 4)).toBe(0);
    expect(randomInt(sequenceRng(0.999), 4)).toBe(3);
  });
});

describe('pickOne', () => {
  it('picks items evenly', () => {
    expect(pickOne(sequenceRng(0), ['a', 'b', 'c'])).toBe('a');
    expect(pickOne(sequenceRng(0.5), ['a', 'b', 'c'])).toBe('b');
    expect(pickOne(sequenceRng(0.99), ['a', 'b', 'c'])).toBe('c');
  });

  it('throws without items', () => {
    expect(() => pickOne(sequenceRng(0.5), [])).toThrow();
  });
});

describe('pickWeighted', () => {
  const items = [{ value: 'a', weight: 1 }, { value: 'b', weight: 3 }];

  it('picks each value in proportion to its weight', () => {
    expect(pickWeighted(sequenceRng(0.2), items)).toBe('a');
    expect(pickWeighted(sequenceRng(0.25), items)).toBe('b');
    expect(pickWeighted(sequenceRng(0.99), items)).toBe('b');
  });

  it('throws without items', () => {
    expect(() => pickWeighted(sequenceRng(0.5), [])).toThrow();
  });
});

describe('pickForCell', () => {
  const items = Array.from({ length: 16 }, (_, i) => i);

  it('always picks the same item for the same cell', () => {
    expect(pickForCell(items, 40, 7)).toBe(pickForCell(items, 40, 7));
  });

  it('spreads the picks over the items', () => {
    const picked = new Set<number>();
    for (let col = 0; col < 32; col++) picked.add(pickForCell(items, col, 3));
    expect(picked.size).toBeGreaterThan(10);
  });

  it('picks differently for another salt, and throws without items', () => {
    const cells = Array.from({ length: 32 }, (_, col) => col);
    expect(cells.map((col) => pickForCell(items, col, 0, 1))).not.toEqual(cells.map((col) => pickForCell(items, col, 0, 2)));
    expect(() => pickForCell([], 0, 0)).toThrow();
  });
});

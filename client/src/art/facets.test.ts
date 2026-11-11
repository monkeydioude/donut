import { describe, expect, it } from 'vitest';
import { facetAt, PATTERN_CELLS, patternPlace, patternPoint, scatteredSeeds } from './facets';

const seeds = scatteredSeeds(4, 2, 5);

// ADR 0011 test 12
describe('the rock and ice pattern', () => {
  it('repeats every 8 cells each way, negative cells included', () => {
    expect(PATTERN_CELLS).toBe(8);
    expect(patternPlace({ col: 3, row: 5 })).toEqual({ col: 3, row: 5 });
    expect(patternPlace({ col: 11, row: 13 })).toEqual({ col: 3, row: 5 });
    expect(patternPlace({ col: -5, row: -3 })).toEqual({ col: 3, row: 5 });
  });

  it('is the same on both sides of an edge between two cells, so neighbors join', () => {
    for (const row of [-0.4, 0, 0.3]) {
      const left = facetAt(seeds, patternPoint({ col: 2, row: 6 }, { col: 0.5, row }));
      const right = facetAt(seeds, patternPoint({ col: 3, row: 6 }, { col: -0.5, row }));
      expect(right).toEqual(left);
    }
  });

  it('has no seam where it repeats', () => {
    const before = facetAt(seeds, patternPoint({ col: 7, row: 1 }, { col: 0.5, row: 0.2 }));
    const after = facetAt(seeds, patternPoint({ col: 0, row: 1 }, { col: -0.5, row: 0.2 }));
    expect(after.seed).toBe(before.seed);
    expect(after.border).toBeCloseTo(before.border);
  });

  it('spreads its seeds one per square of a grid, the same for the same rng seed', () => {
    expect(seeds).toHaveLength(16);
    seeds.forEach((seed, i) => {
      expect(Math.floor(seed.at.col / 2)).toBe(i % 4);
      expect(Math.floor(seed.at.row / 2)).toBe(Math.floor(i / 4));
    });
    expect(scatteredSeeds(4, 2, 5)).toEqual(seeds);
  });
});

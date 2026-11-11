import { describe, expect, it } from 'vitest';
import { footprint, gridOf, objectPlacement, regionName, type MapDescription } from './worldMap';

const world: MapDescription = {
  slug: 'world',
  name: 'World',
  width: 1024,
  height: 512,
  wraps: true,
  chunkSize: 32,
  regions: [{ id: 1, name: 'Sprinkle Meadow', biome: 'meadow' }],
  standsOn: null,
};

describe('a map description', () => {
  it('gives its grid', () => {
    expect(gridOf(world)).toEqual({ width: 1024, height: 512, wraps: true });
  });

  it("gives a region's name, none for region 0 or an unknown one", () => {
    expect(regionName(world, 1)).toBe('Sprinkle Meadow');
    expect(regionName(world, 0)).toBeNull();
    expect(regionName(world, 9)).toBeNull();
  });
});

// ADR 0012 test 28
describe('objects on the map', () => {
  it('a rock covers its cell, a dome the 2 × 2 from its back cell', () => {
    expect(footprint({ kind: 'rock', col: 4, row: 6 })).toEqual([{ col: 4, row: 6 }]);
    expect(footprint({ kind: 'dome', col: 4, row: 6 })).toEqual([
      { col: 4, row: 6 },
      { col: 5, row: 6 },
      { col: 4, row: 7 },
      { col: 5, row: 7 },
    ]);
  });

  it('a dome is drawn on the middle of its footprint and sorted by its front cell', () => {
    expect(objectPlacement('dome', { col: 4, row: 6 })).toEqual({ at: { col: 4.5, row: 6.5 }, sortedBy: { col: 5, row: 7 } });
    expect(objectPlacement('rock', { col: 4, row: 6 })).toEqual({ at: { col: 4, row: 6 }, sortedBy: { col: 4, row: 6 } });
  });
});

import { describe, expect, it } from 'vitest';
import { gridOf, regionName, type MapDescription } from './worldMap';

const world: MapDescription = {
  slug: 'world',
  name: 'World',
  width: 1024,
  height: 512,
  wraps: true,
  chunkSize: 32,
  regions: [{ id: 1, name: 'Sprinkle Meadow', biome: 'meadow' }],
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

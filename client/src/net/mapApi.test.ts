import { describe, expect, it } from 'vitest';
import { parseDescription } from './mapApi';

const serverDescription = () => ({
  id: 'm1',
  slug: 'world',
  name: 'World',
  width: 1024,
  height: 512,
  wraps: true,
  chunkSize: 32,
  regions: [{ id: 1, name: 'Sprinkle Meadow', biome: 'meadow' }],
});

// ADR 0003 test 27
describe('parseDescription', () => {
  it("reads the map's size, wrapping, chunk size and regions", () => {
    expect(parseDescription(serverDescription())).toEqual({
      slug: 'world',
      name: 'World',
      width: 1024,
      height: 512,
      wraps: true,
      chunkSize: 32,
      regions: [{ id: 1, name: 'Sprinkle Meadow', biome: 'meadow' }],
      standsOn: null,
    });
  });

  // ADR 0012 test 24
  it("reads a city's size, and where it stands", () => {
    const standsOn = { map: 'world', col: 1022, row: 510, width: 2048, height: 1024 };
    const city = parseDescription({ ...serverDescription(), slug: 'doma-1', width: 32, height: 48, wraps: false, standsOn });
    expect(city.width).toBe(32);
    expect(city.height).toBe(48);
    expect(city.standsOn).toEqual(standsOn);
  });

  const broken: Record<string, (json: ReturnType<typeof serverDescription>) => unknown> = {
    'not an object': () => 'world',
    'no slug': (json) => ({ ...json, slug: '' }),
    'a size of 0': (json) => ({ ...json, width: 0 }),
    'a size not a multiple of 32': (json) => ({ ...json, height: 100 }),
    'a wrapping size of 48 (ADR 0012)': (json) => ({ ...json, height: 48 }),
    'a bounded size of 24 (ADR 0012)': (json) => ({ ...json, wraps: false, width: 24 }),
    'a bounded size of 0 (ADR 0012)': (json) => ({ ...json, wraps: false, width: 0 }),
    'a standsOn without a map (ADR 0012)': (json) => ({ ...json, standsOn: { col: 1, row: 1, width: 2048, height: 1024 } }),
    'a standsOn outside its map (ADR 0012)': (json) => ({ ...json, standsOn: { map: 'world', col: 2048, row: 1, width: 2048, height: 1024 } }),
    'a size over 4096': (json) => ({ ...json, width: 4128 }),
    'no wraps': (json) => ({ ...json, wraps: undefined }),
    'another chunk size': (json) => ({ ...json, chunkSize: 16 }),
    'no regions': (json) => ({ ...json, regions: undefined }),
    'a region with id 0': (json) => ({ ...json, regions: [{ id: 0, name: 'Nowhere', biome: 'meadow' }] }),
    'a region without a name': (json) => ({ ...json, regions: [{ id: 1, name: '', biome: 'meadow' }] }),
  };
  for (const [name, breakIt] of Object.entries(broken)) {
    it(`refuses ${name}`, () => {
      expect(() => parseDescription(breakIt(serverDescription()))).toThrow(/invalid map description/);
    });
  }
});

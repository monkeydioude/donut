import { describe, expect, it } from 'vitest';
import { fetchDescription, parseDescription } from './mapApi';

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
    });
  });

  const broken: Record<string, (json: ReturnType<typeof serverDescription>) => unknown> = {
    'not an object': () => 'world',
    'no slug': (json) => ({ ...json, slug: '' }),
    'a size of 0': (json) => ({ ...json, width: 0 }),
    'a size not a multiple of 32': (json) => ({ ...json, height: 100 }),
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

describe('fetchDescription', () => {
  it('reads the description of a slug', async () => {
    const urls: string[] = [];
    const map = await fetchDescription('world', async (url) => {
      urls.push(url);
      return new Response(JSON.stringify(serverDescription()));
    });
    expect(urls).toEqual(['/api/maps/world']);
    expect(map.regions).toHaveLength(1);
  });

  it('fails when the server does', async () => {
    await expect(fetchDescription('nowhere', async () => new Response('not found', { status: 404 }))).rejects.toThrow(/404/);
  });
});

import type { MapDescription } from '../map/worldMap';

/** The world as `welcome` describes it: it wraps, and stands nowhere. */
export function worldDescription(): MapDescription {
  return {
    slug: 'world',
    name: 'World',
    width: 1024,
    height: 512,
    wraps: true,
    chunkSize: 32,
    regions: [{ id: 1, name: 'Sprinkle Meadow', biome: 'meadow' }],
    standsOn: null,
  };
}

/** Doma-1 as `welcome` describes it (ADR 0012): 32 × 32, not wrapping, standing on the world's dome. */
export function cityDescription(): MapDescription {
  return {
    slug: 'doma-1',
    name: 'Doma-1',
    width: 32,
    height: 32,
    wraps: false,
    chunkSize: 32,
    regions: [{ id: 1, name: 'Doma-1', biome: 'dome' }],
    standsOn: { map: 'world', col: 1022, row: 510, width: 1024, height: 512 },
  };
}

import { describe, expect, it } from 'vitest';
import { createIsoProjection } from './projection';

describe('createIsoProjection (64x32 tiles)', () => {
  const projection = createIsoProjection(64, 32);

  it('puts the origin tile at (0, 0)', () => {
    expect(projection.toScreen({ col: 0, row: 0 })).toEqual({ x: 0, y: 0 });
  });

  it('moves down-right along columns and down-left along rows', () => {
    expect(projection.toScreen({ col: 1, row: 0 })).toEqual({ x: 32, y: 16 });
    expect(projection.toScreen({ col: 0, row: 1 })).toEqual({ x: -32, y: 16 });
    expect(projection.toScreen({ col: 2, row: 3 })).toEqual({ x: -32, y: 80 });
  });

  it('draws tiles further down the screen on top', () => {
    expect(projection.depth({ col: 2, row: 3 })).toBe(5);
    expect(projection.depth({ col: 1, row: 1 })).toBeGreaterThan(projection.depth({ col: 1, row: 0 }));
  });
});

// ADR 0002 test 17
describe('toTile', () => {
  const projection = createIsoProjection(64, 32);

  it('finds the tile of its center', () => {
    for (const tile of [{ col: 0, row: 0 }, { col: 3, row: 1 }, { col: 1, row: 4 }, { col: 7, row: 7 }]) {
      expect(projection.toTile(projection.toScreen(tile))).toEqual(tile);
    }
  });

  it('finds the tile near the edges of its diamond', () => {
    // tile (2, 2) is centered on (0, 64); its diamond reaches 32 px left and right, 16 px up and down
    expect(projection.toTile({ x: 0, y: 64 - 15 })).toEqual({ col: 2, row: 2 });
    expect(projection.toTile({ x: 0, y: 64 + 15 })).toEqual({ col: 2, row: 2 });
    expect(projection.toTile({ x: -31, y: 64 })).toEqual({ col: 2, row: 2 });
    expect(projection.toTile({ x: 31, y: 64 })).toEqual({ col: 2, row: 2 });
  });

  it('gives the neighbor across each edge', () => {
    // just past the four corners of tile (2, 2)'s diamond
    expect(projection.toTile({ x: 0, y: 64 - 17 })).toEqual({ col: 1, row: 1 });
    expect(projection.toTile({ x: 0, y: 64 + 17 })).toEqual({ col: 3, row: 3 });
    expect(projection.toTile({ x: -33, y: 64 })).toEqual({ col: 1, row: 3 });
    expect(projection.toTile({ x: 33, y: 64 })).toEqual({ col: 3, row: 1 });
  });

  it('gives tiles outside a map too: the caller checks the map', () => {
    expect(projection.toTile({ x: 0, y: -40 })).toEqual({ col: -1, row: -1 });
  });
});

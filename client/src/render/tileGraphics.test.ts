import type { Texture } from 'pixi.js';
import { describe, expect, it } from 'vitest';
import { FLAT_BLOCK } from '../art/block';
import { createGeneratedArt } from '../art/tileArtSet';
import { createSeededRng } from '../core/rng';
import { createWorldGraphics } from './tileGraphics';

// ADR 0011 test 13
describe('terrain sprites', () => {
  it('paint a mountain or glacier once per place in the pattern, edges, shape and icy top', () => {
    let painted = 0;
    const graphics = createWorldGraphics({
      artSet: createGeneratedArt({ tileWidth: 32, tileHeight: 16, rockHeadroom: 6, levelPx: 8 }),
      variantsPerKind: 1,
      rng: createSeededRng(1),
      toTexture: () => {
        painted++;
        return {} as Texture;
      },
    });
    const start = painted;
    const look = { place: { col: 2, row: 3 }, outline: ['se' as const], shape: { ...FLAT_BLOCK, n: 2 }, iced: false };
    graphics.terrain.mountain(look);
    graphics.terrain.mountain({ ...look, outline: ['se'], shape: { ...FLAT_BLOCK, n: 2 } });
    expect(painted - start, 'the same look twice').toBe(1);
    graphics.terrain.mountain({ ...look, iced: true });
    graphics.terrain.mountain({ ...look, place: { col: 3, row: 3 } });
    graphics.terrain.glacier({ place: look.place, edges: [], shape: FLAT_BLOCK });
    graphics.terrain.glacier({ place: look.place, edges: [], shape: FLAT_BLOCK });
    expect(painted - start).toBe(4);
  });

  it("a block's sprite sits on its cell's ground center, under the room its shape needs", () => {
    const graphics = createWorldGraphics({
      artSet: createGeneratedArt({ tileWidth: 32, tileHeight: 16, rockHeadroom: 6, levelPx: 8 }),
      variantsPerKind: 1,
      rng: createSeededRng(1),
      toTexture: () => ({}) as Texture,
    });
    const sprite = graphics.terrain.mountain({ place: { col: 0, row: 0 }, outline: [], shape: { ...FLAT_BLOCK, n: 3 }, iced: false });
    expect(sprite.pivot).toEqual({ x: 16, y: 24 + 8 });
  });
});

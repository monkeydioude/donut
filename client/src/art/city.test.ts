import { describe, expect, it } from 'vitest';
import { distinctColors, isOpaque, pixelsWhere } from '../testing/pixels';
import { floorLookOf, glassLookOf, paintFloor, paintGlassFace, paintWallTop } from './city';
import { readPixel } from './pixelBuffer';
import { isOnGround, type TileFrame } from './tileFrame';

const frame: TileFrame = { tileWidth: 32, tileHeight: 16, headroom: 0 };
const INSIDE = 0x2f2838;

// ADR 0012: Doma-1's art, approved on its preview image
describe('the floor', () => {
  it('covers the ground diamond only, in a few pale colors', () => {
    const floor = paintFloor(frame, { wear: 'none', slab: 0 });
    expect(pixelsWhere(floor, (x, y) => isOpaque(floor, x, y) !== isOnGround(frame, x, y))).toEqual([]);
    expect(distinctColors(floor).size).toBe(2);
  });

  it('a missing slab is a darker quarter of the cell', () => {
    const plain = paintFloor(frame, { wear: 'none', slab: 0 });
    const missing = paintFloor(frame, { wear: 'missing', slab: 3 });
    const changed = pixelsWhere(missing, (x, y) => readPixel(missing, x, y).rgb !== readPixel(plain, x, y).rgb);
    expect(changed.length).toBeGreaterThan(40);
    expect(changed.length).toBeLessThan(140);
  });

  it("is picked from the cell's place: the same cell, the same look; most cells plain", () => {
    expect(floorLookOf({ col: 7, row: 9 })).toEqual(floorLookOf({ col: 7, row: 9 }));
    let plain = 0;
    for (let i = 0; i < 1000; i++) if (floorLookOf({ col: i % 40, row: Math.floor(i / 40) }).wear === 'none') plain++;
    expect(plain).toBeGreaterThan(750);
  });
});

describe('the wall', () => {
  it("its top is the metal ring, on the ground diamond", () => {
    const top = paintWallTop(frame);
    expect(pixelsWhere(top, (x, y) => isOpaque(top, x, y) !== isOnGround(frame, x, y))).toEqual([]);
  });

  it('its glass shows the dark outside through a missing panel only', () => {
    const whole = paintGlassFace(frame, 8, 'left', { offset: 0, missing: -1, cracked: -1 });
    const broken = paintGlassFace(frame, 8, 'left', { offset: 0, missing: 1, cracked: -1 });
    const dark = (buffer: typeof whole) => pixelsWhere(buffer, (x, y) => isOpaque(buffer, x, y) && readPixel(buffer, x, y).rgb === INSIDE);
    expect(dark(whole)).toEqual([]);
    expect(dark(broken).length).toBeGreaterThan(10);
  });

  it("its honeycomb's rows shift half a panel every other level drawn, so they meet across cells", () => {
    const a = glassLookOf({ col: 3, row: 4 }, 3, 0);
    expect(glassLookOf({ col: 3, row: 4 }, 3, 1).offset).not.toBe(a.offset);
    expect(glassLookOf({ col: 4, row: 4 }, 3, 0).offset, 'the next cell is drawn a level lower').not.toBe(a.offset);
    expect(glassLookOf({ col: 4, row: 4 }, 3, 1).offset).toBe(a.offset);
  });
});

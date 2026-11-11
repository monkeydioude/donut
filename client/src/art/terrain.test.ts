import { describe, expect, it } from 'vitest';
import { BRIDGE_COLORS, paintBridge, RAIL_PX } from './bridge';
import { FACE_COLORS, paintFace, paintRim } from './cliff';
import { depthFromEdge, edgesKey } from './edges';
import { FLAT_BLOCK, blockFrame, type BlockShape } from './block';
import { GLACIER_COLORS, paintGlacier } from './glacier';
import { HOLE_COLORS, paintHole } from './hole';
import { MOUNTAIN_COLORS, paintMountain, type MountainLook } from './mountain';
import { OCEAN_COLORS, paintOcean } from './ocean';
import { readPixel, type PixelBuffer } from './pixelBuffer';
import { isOnGround, type TileFrame } from './tileFrame';
import { paintWater, WATER_COLORS, WATER_FRAMES } from './water';

const frame: TileFrame = { tileWidth: 32, tileHeight: 16, headroom: 0 };

function colorsOf(buffer: PixelBuffer): Set<number> {
  const colors = new Set<number>();
  for (let y = 0; y < buffer.height; y++) {
    for (let x = 0; x < buffer.width; x++) if (readPixel(buffer, x, y).alpha > 0) colors.add(readPixel(buffer, x, y).rgb);
  }
  return colors;
}

describe('edges', () => {
  it('measure how far inside the diamond a pixel is from an edge', () => {
    expect(depthFromEdge(frame, 'se', 30, 8)).toBeLessThan(1);
    expect(depthFromEdge(frame, 'se', 16, 8)).toBeGreaterThan(6);
    expect(depthFromEdge(frame, 'nw', 30, 8), 'the other quarter').toBe(Infinity);
    expect(depthFromEdge(frame, 'se', 31, 15), 'off the ground').toBe(Infinity);
  });

  it('key a set of edges the same in any order', () => {
    expect(edgesKey(['sw', 'ne'])).toBe(edgesKey(['ne', 'sw']));
  });
});

describe('paintFace', () => {
  it('paints the level under one side of the diamond, with a band at the bottom', () => {
    const left = paintFace(frame, 8, 'earth', 'left');
    expect(left.height).toBe(16 + 8);
    // the diamond's lowest pixel at column 4 is row 9
    expect(readPixel(left, 4, 10).rgb, 'just under the lower left edge').toBe(FACE_COLORS.earth.left.face);
    expect(readPixel(left, 4, 12).rgb).toBe(FACE_COLORS.earth.left.face);
    expect(readPixel(left, 4, 17).rgb).toBe(FACE_COLORS.earth.left.band);
    expect(readPixel(left, 4, 18).alpha, 'one level only').toBe(0);
    expect(readPixel(left, 20, 18).alpha, 'nothing on the other side').toBe(0);
    expect(readPixel(left, 4, 8).alpha, 'nothing on the ground').toBe(0);
  });

  it('the right side is darker', () => {
    const right = paintFace(frame, 8, 'earth', 'right');
    expect(readPixel(right, 27, 12).rgb).toBe(FACE_COLORS.earth.right.face);
    expect(FACE_COLORS.earth.right.face).not.toBe(FACE_COLORS.earth.left.face);
  });
});

describe('paintRim', () => {
  it('lines only the edges given, one pixel inside the diamond, in the color of the earth faces', () => {
    const rim = paintRim(frame, ['nw']);
    expect(readPixel(rim, 1, 7).rgb, 'on the nw edge').toBe(FACE_COLORS.earth.left.face);
    expect(readPixel(rim, 2, 7).rgb, 'an iso line steps 2 px across per pixel down').toBe(FACE_COLORS.earth.left.face);
    expect(readPixel(rim, 4, 7).alpha, 'further in').toBe(0);
    expect(readPixel(rim, 30, 8).alpha, 'the se edge').toBe(0);
    expect(colorsOf(paintRim(frame, []))).toEqual(new Set());
  });

  it("paints a corner's 2 pixels, where the rims of the cells around meet", () => {
    const corners: Record<string, [number, number][]> = {
      n: [[15, 0], [16, 0]],
      e: [[30, 7], [30, 8]],
      s: [[15, 15], [16, 15]],
      w: [[1, 7], [1, 8]],
    };
    for (const [corner, pixels] of Object.entries(corners)) {
      const rim = paintRim(frame, [], [corner as 'n' | 'e' | 's' | 'w']);
      const painted: [number, number][] = [];
      for (let y = 0; y < 16; y++) for (let x = 0; x < 32; x++) if (readPixel(rim, x, y).alpha > 0) painted.push([x, y]);
      expect(painted, corner).toEqual(pixels);
    }
  });

  it('takes the lit brown on the left edges and the darker one on the right edges, like the faces', () => {
    const rim = paintRim(frame, ['sw', 'se', 'ne']);
    expect(readPixel(rim, 1, 8).rgb, 'sw').toBe(FACE_COLORS.earth.left.face);
    expect(readPixel(rim, 30, 8).rgb, 'se').toBe(FACE_COLORS.earth.right.face);
    expect(readPixel(rim, 30, 7).rgb, 'ne').toBe(FACE_COLORS.earth.right.face);
  });
});

describe('paintWater', () => {
  it('flows: each frame differs, and the loop is seamless', () => {
    const frames = Array.from({ length: WATER_FRAMES + 1 }, (_, i) => paintWater(frame, i, []).data.join());
    expect(new Set(frames.slice(0, WATER_FRAMES)).size).toBe(WATER_FRAMES);
    expect(frames[WATER_FRAMES]).toBe(frames[0]);
  });

  it('has streaks, and foam only on the edges given', () => {
    expect(colorsOf(paintWater(frame, 0, []))).toEqual(new Set([WATER_COLORS.base, WATER_COLORS.streak]));
    const foamed = paintWater(frame, 0, ['se']);
    expect(readPixel(foamed, 30, 8).rgb).toBe(WATER_COLORS.foam);
    expect(readPixel(foamed, 1, 7).rgb, 'not on the nw edge').not.toBe(WATER_COLORS.foam);
  });

  it('stays on the ground diamond', () => {
    const water = paintWater(frame, 2, ['ne', 'nw', 'se', 'sw']);
    for (let y = 0; y < 16; y++) for (let x = 0; x < 32; x++) expect(readPixel(water, x, y).alpha > 0).toBe(isOnGround(frame, x, y));
  });
});

describe('paintHole', () => {
  it('is a dark pit, with a rim and an inner wall under its back edges toward other kinds', () => {
    expect(colorsOf(paintHole(frame, []))).toEqual(new Set([HOLE_COLORS.pit]));
    const lone = paintHole(frame, ['ne', 'nw', 'se', 'sw']);
    expect(readPixel(lone, 30, 8).rgb, 'the rim').toBe(HOLE_COLORS.rim);
    expect(readPixel(lone, 8, 5).rgb, 'under the nw edge: the wall').toBe(HOLE_COLORS.walls[0]);
    expect(readPixel(lone, 16, 8).rgb, 'its middle: the pit').toBe(HOLE_COLORS.pit);
  });
});

describe('paintBridge', () => {
  const bridgeFrame: TileFrame = { ...frame, headroom: RAIL_PX + 1 };

  it('lays planks with seams, and rails on its sides toward the river only', () => {
    expect(colorsOf(paintBridge(bridgeFrame, ['ne', 'sw']))).toEqual(new Set([BRIDGE_COLORS.plank, BRIDGE_COLORS.seam]));
    const railed = colorsOf(paintBridge(bridgeFrame, ['nw', 'se']));
    expect(railed.has(BRIDGE_COLORS.rail) && railed.has(BRIDGE_COLORS.railTop)).toBe(true);
  });

  it("raises the back rail into the frame's headroom", () => {
    const railed = paintBridge(bridgeFrame, ['nw']);
    const top = Array.from({ length: 32 }, (_, x) => readPixel(railed, x, 2).alpha).some((alpha) => alpha > 0);
    expect(top).toBe(true);
  });
});

// ADR 0008 test 20
describe('paintOcean', () => {
  it('is still: a deeper blue with ripples, the same every time', () => {
    expect(colorsOf(paintOcean(frame, []))).toEqual(new Set([OCEAN_COLORS.base, OCEAN_COLORS.ripple]));
    expect(paintOcean(frame, []).data.join()).toBe(paintOcean(frame, []).data.join());
  });

  it('has foam on the edges given only', () => {
    const foamed = paintOcean(frame, ['se']);
    expect(readPixel(foamed, 30, 8).rgb, 'on the se edge').toBe(OCEAN_COLORS.foam);
    expect(readPixel(foamed, 1, 8).rgb, 'the sw edge has none').not.toBe(OCEAN_COLORS.foam);
    expect(colorsOf(paintOcean(frame, [])).has(OCEAN_COLORS.foam)).toBe(false);
  });

  it('covers the ground exactly', () => {
    const ocean = paintOcean(frame, ['ne', 'nw', 'se', 'sw']);
    for (let y = 0; y < 16; y++) for (let x = 0; x < 32; x++) expect(readPixel(ocean, x, y).alpha > 0).toBe(isOnGround(frame, x, y));
  });
});

// ADR 0011 test 12
describe('paintMountain', () => {
  const look = (more: Partial<MountainLook> = {}): MountainLook => ({ place: { col: 0, row: 0 }, outline: [], shape: FLAT_BLOCK, iced: false, ...more });
  const paint = (l: MountainLook) => paintMountain(blockFrame(32, 16, l.shape, 8), l, 8);
  const rock = new Set<number>([...MOUNTAIN_COLORS.top, MOUNTAIN_COLORS.crack, MOUNTAIN_COLORS.outline]);
  const stone = new Set<number>([...MOUNTAIN_COLORS.side.left, ...MOUNTAIN_COLORS.side.right, MOUNTAIN_COLORS.crack]);

  it('is rock on the ground diamond only, without a peak: nothing rises above its top', () => {
    const flat = paint(look());
    for (let y = 0; y < 16; y++) for (let x = 0; x < 32; x++) expect(readPixel(flat, x, y).alpha > 0, `${x},${y}`).toBe(isOnGround(frame, x, y));
    for (const color of colorsOf(flat)) expect(rock.has(color), color.toString(16)).toBe(true);
  });

  it('a peak rises to a point over its middle, lit on its left', () => {
    const shape: BlockShape = { ...FLAT_BLOCK, peak: true };
    const peak = paint(look({ shape }));
    const groundCenterY = blockFrame(32, 16, shape, 8).headroom + 8;
    expect(groundCenterY - 12, 'its point is 12 px over its middle, at the top of the frame').toBe(0);
    expect(readPixel(peak, 15, 0).alpha + readPixel(peak, 16, 0).alpha, 'the point, in the middle').toBeGreaterThan(0);
    expect(readPixel(peak, 4, 0).alpha, 'nothing beside it up there').toBe(0);
    const brightness = (rgb: number) => ((rgb >> 16) & 0xff) + ((rgb >> 8) & 0xff) + (rgb & 0xff);
    expect(brightness(readPixel(peak, 10, groundCenterY + 2).rgb), 'left of the middle is lighter').toBeGreaterThan(brightness(readPixel(peak, 21, groundCenterY + 2).rgb));
  });

  it('has an outline on the edges given only', () => {
    expect(readPixel(paint(look({ outline: ['se'] })), 30, 8).rgb, 'on the se edge').toBe(MOUNTAIN_COLORS.outline);
    expect(colorsOf(paint(look())).has(MOUNTAIN_COLORS.outline), 'none given').toBe(false);
  });

  it('an icy top is in the ice colors, its sides in stone', () => {
    const shape: BlockShape = { ...FLAT_BLOCK, w: 2 };
    const iced = paint(look({ shape, iced: true }));
    const ice = new Set<number>([...GLACIER_COLORS.top, GLACIER_COLORS.crack, GLACIER_COLORS.gloss, GLACIER_COLORS.edge]);
    const colors = colorsOf(iced);
    expect([...colors].some((c) => ice.has(c)), 'an icy top').toBe(true);
    expect([...colors].some((c) => stone.has(c)), 'stone sides').toBe(true);
    for (const color of colors) expect(ice.has(color) || stone.has(color), color.toString(16)).toBe(true);
  });

  it('is the same for the same place in the pattern, 8 cells apart', () => {
    expect(paint(look({ place: { col: 3, row: 5 } })).data.join()).toBe(paint(look({ place: { col: 3, row: 5 } })).data.join());
  });
});

// ADR 0011 test 12
describe('paintGlacier', () => {
  const paint = (edges: readonly ('ne' | 'nw' | 'se' | 'sw')[], shape: BlockShape = FLAT_BLOCK) =>
    paintGlacier(blockFrame(32, 16, shape, 8), { place: { col: 0, row: 0 }, edges, shape }, 8);

  it('is ice on the ground diamond only when flat: no raised slab', () => {
    const flat = paint(['ne', 'nw', 'se', 'sw']);
    for (let y = 0; y < 16; y++) for (let x = 0; x < 32; x++) expect(readPixel(flat, x, y).alpha > 0, `${x},${y}`).toBe(isOnGround(frame, x, y));
  });

  it('has its bright line on the edges given only', () => {
    expect(readPixel(paint(['se']), 30, 8).rgb, 'on the se edge').toBe(GLACIER_COLORS.edge);
    expect(colorsOf(paint([])).has(GLACIER_COLORS.edge), 'none given').toBe(false);
  });

  it('leaning up, its sides are the ice faces, streaked', () => {
    const colors = colorsOf(paint([], { ...FLAT_BLOCK, w: 2 }));
    expect(colors.has(FACE_COLORS.ice.left.face)).toBe(true);
    expect(colors.has(0x8ad0ea), 'a lighter streak').toBe(true);
  });
});

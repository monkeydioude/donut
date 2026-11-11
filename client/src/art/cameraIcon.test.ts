import { describe, expect, it } from 'vitest';
import { distinctColors, pixelsWhere } from '../testing/pixels';
import { PADLOCK_BOX, paintCameraIcon } from './cameraIcon';
import { readPixel } from './pixelBuffer';

const locked = paintCameraIcon(true);
const free = paintCameraIcon(false);
const GOLD = 0xffd84a;

const inPadlock = (x: number, y: number) =>
  x >= PADLOCK_BOX.x && x < PADLOCK_BOX.x + PADLOCK_BOX.width && y >= PADLOCK_BOX.y && y < PADLOCK_BOX.y + PADLOCK_BOX.height;
const differs = (x: number, y: number) => {
  const [a, b] = [readPixel(locked, x, y), readPixel(free, x, y)];
  return a.alpha !== b.alpha || (a.alpha > 0 && a.rgb !== b.rgb);
};
const isGold = (icon: typeof locked) => (x: number, y: number) => readPixel(icon, x, y).alpha === 255 && readPixel(icon, x, y).rgb === GOLD;

// ADR 0004 test 26
describe('paintCameraIcon', () => {
  it('has the same size locked and free, on a transparent background', () => {
    expect([locked.width, locked.height]).toEqual([free.width, free.height]);
    expect(readPixel(free, 0, 0).alpha).toBe(0);
  });

  it('draws the same camera: the icons differ only where the padlock goes', () => {
    expect(pixelsWhere(locked, (x, y) => differs(x, y) && !inPadlock(x, y))).toEqual([]);
    expect(pixelsWhere(locked, differs).length).toBeGreaterThan(0);
  });

  it('has the gold padlock only when locked', () => {
    expect(pixelsWhere(locked, isGold(locked)).length).toBeGreaterThan(0);
    expect(pixelsWhere(free, isGold(free))).toEqual([]);
  });

  it('uses a small palette', () => {
    expect(distinctColors(locked).size).toBeLessThanOrEqual(8);
  });
});

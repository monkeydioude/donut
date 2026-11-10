import { describe, expect, it } from 'vitest';
import { distinctColors } from '../testing/pixels';
import { FINGER_HOTSPOT, paintFinger } from './finger';
import { readPixel } from './pixelBuffer';

// ADR 0002 test 21
describe('paintFinger', () => {
  const finger = paintFinger();

  it('is a 20 x 23 sprite on a transparent background', () => {
    expect([finger.width, finger.height]).toEqual([20, 23]);
    for (const [x, y] of [[19, 0], [0, 22], [19, 22]] as const) expect(readPixel(finger, x, y).alpha).toBe(0);
  });

  it('uses a small palette', () => {
    expect(distinctColors(finger).size).toBeLessThanOrEqual(4);
  });

  it('points with an opaque fingertip', () => {
    expect(readPixel(finger, FINGER_HOTSPOT.x, FINGER_HOTSPOT.y).alpha).toBe(255);
  });
});

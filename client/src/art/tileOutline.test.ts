import { describe, expect, it } from 'vitest';
import { pixelsWhere } from '../testing/pixels';
import { readPixel } from './pixelBuffer';
import { isInnerGround, isOnGround, type TileFrame } from './tileFrame';
import { paintTileOutline } from './tileOutline';

// ADR 0002 test 22
describe('paintTileOutline', () => {
  const frame: TileFrame = { tileWidth: 32, tileHeight: 16, headroom: 0 };

  it('paints only the rim of the ground diamond', () => {
    const outline = paintTileOutline(frame, 0xffffff);
    const isRim = (x: number, y: number) => isOnGround(frame, x, y) && !isInnerGround(frame, x, y);
    expect(pixelsWhere(outline, (x, y) => (readPixel(outline, x, y).alpha > 0) !== isRim(x, y))).toEqual([]);
    expect(pixelsWhere(outline, isRim).length).toBeGreaterThan(40);
  });

  it('takes its color and transparency', () => {
    const outline = paintTileOutline(frame, 0xffd84a, 200);
    expect(readPixel(outline, 16, 0)).toEqual({ rgb: 0xffd84a, alpha: 200 });
  });
});

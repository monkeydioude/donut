import { describe, expect, it } from 'vitest';
import { distinctColors, isOpaque, pixelsWhere } from '../testing/pixels';
import { readPixel } from './pixelBuffer';
import { paintPlayer, PLAYER_LOOKS } from './player';
import { isDrawable, textureSize, type TileFrame } from './tileFrame';

const frame: TileFrame = { tileWidth: 32, tileHeight: 16, headroom: 10 };

// ADR 0001 test 27
describe('paintPlayer', () => {
  it('stands in the frame, transparent around it, never over the tiles in front', () => {
    const buffer = paintPlayer(frame, 'you');
    const { width, height } = textureSize(frame);
    for (const [x, y] of [[0, 0], [width - 1, 0], [0, height - 1], [width - 1, height - 1]] as const) {
      expect(readPixel(buffer, x, y).alpha).toBe(0);
    }
    expect(pixelsWhere(buffer, (x, y) => isOpaque(buffer, x, y)).length).toBeGreaterThan(40);
    expect(pixelsWhere(buffer, (x, y) => readPixel(buffer, x, y).alpha > 0 && !isDrawable(frame, x, y))).toEqual([]);
  });

  it('uses a small palette', () => {
    expect(distinctColors(paintPlayer(frame, 'you')).size).toBeLessThanOrEqual(8);
  });

  it('dresses you and the others differently', () => {
    expect(distinctColors(paintPlayer(frame, 'you'))).toContain(PLAYER_LOOKS.you.shirt);
    expect(distinctColors(paintPlayer(frame, 'other'))).toContain(PLAYER_LOOKS.other.shirt);
    expect(distinctColors(paintPlayer(frame, 'other'))).not.toContain(PLAYER_LOOKS.you.shirt);
  });
});

import { describe, expect, it } from 'vitest';
import { pixelsWhere } from '../testing/pixels';
import { GROUND_SHADOW } from './drawing';
import { NEIGHBORS, readPixel, type PixelBuffer } from './pixelBuffer';
import { paintPlayerShadow, paintYourRing, YOUR_RING_RGB } from './playerMarks';
import type { TileFrame } from './tileFrame';

const frame: TileFrame = { tileWidth: 32, tileHeight: 16, headroom: 0 };
/** The four pixels around the ground center, (16, 8). */
const center = [[15, 7], [16, 7], [15, 8], [16, 8]] as const;

const painted = (buffer: PixelBuffer) => pixelsWhere(buffer, (x, y) => readPixel(buffer, x, y).alpha > 0);
const xsOf = (pixels: string[]) => pixels.map((p) => Number(p.split(',')[0]));

/** Whether a fill from the ground center through unpainted pixels reaches the buffer's border. */
function leaksOut(buffer: PixelBuffer): boolean {
  const seen = new Set<string>();
  const queue: [number, number][] = [[16, 8]];
  while (queue.length > 0) {
    const [x, y] = queue.pop()!;
    const key = `${x},${y}`;
    if (seen.has(key) || readPixel(buffer, x, y).alpha > 0) continue;
    if (x === 0 || y === 0 || x === buffer.width - 1 || y === buffer.height - 1) return true;
    seen.add(key);
    for (const [dx, dy] of NEIGHBORS) queue.push([x + dx, y + dy]);
  }
  return false;
}

// ADR 0006 test 27
describe('paintYourRing', () => {
  const ring = paintYourRing(frame);

  it('is yellow and opaque', () => {
    const notYellow = pixelsWhere(ring, (x, y) => {
      const pixel = readPixel(ring, x, y);
      return pixel.alpha > 0 && (pixel.rgb !== YOUR_RING_RGB || pixel.alpha !== 255);
    });
    expect(painted(ring).length).toBeGreaterThan(20);
    expect(notYellow).toEqual([]);
  });

  it('is a closed ring, empty inside', () => {
    for (const [x, y] of center) expect(readPixel(ring, x, y).alpha).toBe(0);
    expect(leaksOut(ring)).toBe(false);
  });

  it('is wider than the shadow, and centered on the ground center', () => {
    const ringXs = xsOf(painted(ring));
    const shadowXs = xsOf(painted(paintPlayerShadow(frame)));
    expect(Math.min(...ringXs)).toBeLessThan(Math.min(...shadowXs));
    expect(Math.max(...ringXs)).toBeGreaterThan(Math.max(...shadowXs));
    expect(Math.min(...ringXs) + Math.max(...ringXs)).toBe(31);
  });
});

// ADR 0006 test 28
describe('paintPlayerShadow', () => {
  const shadow = paintPlayerShadow(frame);

  it('is the usual semi-transparent ground shadow', () => {
    const notShadow = pixelsWhere(shadow, (x, y) => {
      const pixel = readPixel(shadow, x, y);
      return pixel.alpha > 0 && (pixel.rgb !== GROUND_SHADOW.rgb || pixel.alpha !== GROUND_SHADOW.alpha);
    });
    expect(notShadow).toEqual([]);
  });

  it('covers the ground center, symmetric around it', () => {
    for (const [x, y] of center) expect(readPixel(shadow, x, y).alpha).toBe(GROUND_SHADOW.alpha);
    const xs = xsOf(painted(shadow));
    expect(Math.min(...xs) + Math.max(...xs)).toBe(31);
  });
});

import type { Rng } from '../core/rng';
import { isInEllipse, isShapeEdge, paintGroundShadow, type Ellipse } from './drawing';
import { createPixelBuffer, forEachPixel, setPixel, type PixelBuffer } from './pixelBuffer';
import { groundCenter, isDrawable, textureSize, type TileFrame } from './tileFrame';

const ROCK = {
  outline: 0x4f4a57,
  /** From lit to dark: a pixel takes the first shade its light reaches, else the darkest. */
  shades: [
    { minLight: 0.85, rgb: 0xf2efe7 },
    { minLight: 0.5, rgb: 0xd2cec5 },
    { minLight: 0.1, rgb: 0xaaa59d },
  ],
  darkest: 0x847f79,
  pebbleChance: 0.5,
  /** How far below its center a boulder goes, in radii: below that it is cut flat on the ground. */
  bottomCut: 0.7,
};

/** Unit vector toward the light: from the top-left, facing the viewer. */
const LIGHT = { x: -0.45, y: -0.65, z: 0.61 };

/** A rock object: a boulder, sometimes with a pebble beside it, and their shadows. Transparent elsewhere. */
export function paintRock(frame: TileFrame, rng: Rng): PixelBuffer {
  const { width, height } = textureSize(frame);
  const buffer = createPixelBuffer(width, height);
  for (const boulder of layoutBoulders(frame, rng)) {
    paintGroundShadow(buffer, frame, shadowOf(boulder));
    paintBoulder(buffer, frame, boulder);
  }
  return buffer;
}

/** Sizes are given for a 32px wide tile and scaled to the frame. Sorted back to front. */
function layoutBoulders(frame: TileFrame, rng: Rng): Ellipse[] {
  const center = groundCenter(frame);
  const scale = frame.tileWidth / 32;
  const rx = (7 + rng() * 1.5) * scale;
  const ry = rx * 0.85;
  const boulders = [{ cx: center.x + (rng() - 0.5) * 2 * scale, cy: center.y - ry * 0.5, rx, ry }];
  if (rng() < ROCK.pebbleChance) {
    const side = rng() < 0.5 ? -1 : 1;
    boulders.push({ cx: center.x + side * 8 * scale, cy: center.y - 0.5 * scale, rx: 3.5 * scale, ry: 3 * scale });
  }
  return boulders.sort((a, b) => a.cy - b.cy);
}

function shadowOf(boulder: Ellipse): Ellipse {
  return { cx: boulder.cx + 1, cy: boulder.cy + boulder.ry * ROCK.bottomCut, rx: boulder.rx * 1.1, ry: boulder.ry * 0.45 };
}

function paintBoulder(buffer: PixelBuffer, frame: TileFrame, boulder: Ellipse): void {
  const inside = (x: number, y: number) => isInBoulder(boulder, x, y);
  forEachPixel(buffer.width, buffer.height, (x, y) => {
    if (!inside(x, y) || !isDrawable(frame, x, y)) return;
    setPixel(buffer, x, y, isShapeEdge(inside, x, y) ? ROCK.outline : shadeAt(boulder, x, y));
  });
}

function isInBoulder(boulder: Ellipse, x: number, y: number): boolean {
  return isInEllipse(boulder, x, y) && y + 0.5 <= boulder.cy + boulder.ry * ROCK.bottomCut;
}

/** Lights the ellipse as if it were the front half of a sphere, then snaps to the palette. */
function shadeAt(boulder: Ellipse, x: number, y: number): number {
  const u = (x + 0.5 - boulder.cx) / boulder.rx;
  const v = (y + 0.5 - boulder.cy) / boulder.ry;
  const z = Math.sqrt(Math.max(0, 1 - u * u - v * v));
  const light = u * LIGHT.x + v * LIGHT.y + z * LIGHT.z;
  return ROCK.shades.find((shade) => light >= shade.minLight)?.rgb ?? ROCK.darkest;
}

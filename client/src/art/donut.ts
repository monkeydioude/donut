import type { DonutRing } from '../game/mapMarks';
import { createPixelBuffer, forEachPixel, setPixel, type PixelBuffer } from './pixelBuffer';

/**
 * The donut planet seen from above (ADR 0005): a green ring in the world's colors, with darker patches, lit from the
 * top-left, with a colored dark outline and a transparent hole.
 */
export const DONUT_SIZE = 40;
const CENTER = DONUT_SIZE / 2;
const OUTER = CENTER - 1;
const INNER = DONUT_SIZE * 0.19;
const OUTLINE = 1.5;
/** How deep the light and shade go into the ring from its outer edge, in pixels. */
const RIM = 3;

const PALETTE = {
  outline: 0x2f6b55,
  base: 0x74d3a5,
  light: 0x9be3c0,
  shade: 0x5fc396,
  patch: 0x4fb386,
};

/**
 * The darker patches: where they are (angle from the top, clockwise, and distance from the center, as a share of the
 * outer radius) and their size, as a share of the sprite.
 */
const PATCHES = [
  { angle: 0.3, distance: 0.67, size: 0.065 },
  { angle: 1.4, distance: 0.8, size: 0.05 },
  { angle: 2.2, distance: 0.57, size: 0.055 },
  { angle: 3.3, distance: 0.73, size: 0.07 },
  { angle: 4.3, distance: 0.6, size: 0.05 },
  { angle: 5.2, distance: 0.8, size: 0.055 },
].map(({ angle, distance, size }) => ({
  x: CENTER + distance * OUTER * Math.sin(angle),
  y: CENTER - distance * OUTER * Math.cos(angle),
  radius: size * DONUT_SIZE,
}));

/** The ring your mark moves on: inside the outline, so it always shows on the land. */
export const DONUT_MARK_RING: DonutRing = { center: { x: CENTER, y: CENTER }, outer: OUTER - 2.5, inner: INNER + 2.5 };

export function paintDonut(): PixelBuffer {
  const donut = createPixelBuffer(DONUT_SIZE, DONUT_SIZE);
  forEachPixel(DONUT_SIZE, DONUT_SIZE, (x, y) => {
    const rgb = donutPixel(x + 0.5, y + 0.5);
    if (rgb !== null) setPixel(donut, x, y, rgb);
  });
  return donut;
}

/** The color at a point of the sprite, null outside the ring. */
function donutPixel(x: number, y: number): number | null {
  const dx = x - CENTER;
  const dy = y - CENTER;
  const distance = Math.hypot(dx, dy);
  if (distance > OUTER || distance < INNER) return null;
  if (distance > OUTER - OUTLINE || distance < INNER + OUTLINE) return PALETTE.outline;
  if (PATCHES.some((p) => Math.hypot(x - p.x, y - p.y) <= p.radius)) return PALETTE.patch;
  if (distance > OUTER - RIM) return rimColor(dx, dy, distance);
  return PALETTE.base;
}

/** The outer rim: lit toward the top-left, shaded toward the bottom-right. */
function rimColor(dx: number, dy: number, distance: number): number {
  const towardLight = (-dx - dy) / (distance * Math.SQRT2);
  if (towardLight > 0.5) return PALETTE.light;
  if (towardLight < -0.5) return PALETTE.shade;
  return PALETTE.base;
}

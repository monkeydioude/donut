import { isInnerGround, isOnGround, type TileFrame } from './tileFrame';
import { createPixelBuffer, forEachPixel, NEIGHBORS, setPixel, type PixelBuffer } from './pixelBuffer';

export interface Point {
  x: number;
  y: number;
}

/** A pixel of a small pattern, relative to where the pattern is placed. */
export interface PatternPixel {
  dx: number;
  dy: number;
  rgb: number;
}

export interface Ellipse {
  cx: number;
  cy: number;
  rx: number;
  ry: number;
}

/** Draws the pattern at `origin` only if it fits entirely on the inner ground. Returns whether it was drawn. */
export function stampOnGround(buffer: PixelBuffer, frame: TileFrame, origin: Point, pattern: readonly PatternPixel[]): boolean {
  const fits = pattern.every(({ dx, dy }) => isInnerGround(frame, origin.x + dx, origin.y + dy));
  if (fits) pattern.forEach(({ dx, dy, rgb }) => setPixel(buffer, origin.x + dx, origin.y + dy, rgb));
  return fits;
}

/** Every pixel of the inner ground, where details can be placed. */
export function innerGroundPoints(buffer: PixelBuffer, frame: TileFrame): Point[] {
  const points: Point[] = [];
  forEachPixel(buffer.width, buffer.height, (x, y) => {
    if (isInnerGround(frame, x, y)) points.push({ x, y });
  });
  return points;
}

export type PixelTest = (x: number, y: number) => boolean;

/**
 * A sprite drawn by hand: one string per row, one character per pixel, each character a palette color.
 * Characters missing from the palette ('.') are transparent.
 */
export function bufferFromRows(rows: readonly string[], palette: Readonly<Record<string, number>>): PixelBuffer {
  const width = Math.max(...rows.map((row) => row.length));
  const buffer = createPixelBuffer(width, rows.length);
  rows.forEach((row, y) => {
    [...row].forEach((char, x) => {
      const rgb = palette[char];
      if (rgb !== undefined) setPixel(buffer, x, y, rgb);
    });
  });
  return buffer;
}

/** True for a pixel of a shape touching a pixel outside it: where its outline goes. */
export function isShapeEdge(inside: PixelTest, x: number, y: number): boolean {
  return inside(x, y) && NEIGHBORS.some(([dx, dy]) => !inside(x + dx, y + dy));
}

/** A shadow on the ground: a dark teal, see-through, so it darkens whatever ground is under it. */
export const GROUND_SHADOW = { rgb: 0x1d6b4f, alpha: 64 } as const;

/** Paints the ground shadow on every pixel of the ellipse that is on the frame's ground. */
export function paintGroundShadow(buffer: PixelBuffer, frame: TileFrame, shadow: Ellipse): void {
  forEachPixel(buffer.width, buffer.height, (x, y) => {
    if (isOnGround(frame, x, y) && isInEllipse(shadow, x, y)) setPixel(buffer, x, y, GROUND_SHADOW.rgb, GROUND_SHADOW.alpha);
  });
}

/** True when the center of pixel (x, y) is inside the ellipse. */
export function isInEllipse(ellipse: Ellipse, x: number, y: number): boolean {
  const u = (x + 0.5 - ellipse.cx) / ellipse.rx;
  const v = (y + 0.5 - ellipse.cy) / ellipse.ry;
  return u * u + v * v <= 1;
}

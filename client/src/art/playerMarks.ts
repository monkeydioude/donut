import { isInEllipse, isShapeEdge, paintGroundShadow, type Ellipse } from './drawing';
import { createPixelBuffer, forEachPixel, setPixel, type PixelBuffer } from './pixelBuffer';
import { groundCenter, textureSize, type TileFrame } from './tileFrame';

/** Your ring's color: the selection border's yellow (ADR 0006). */
export const YOUR_RING_RGB = 0xffd84a;

/** The shadow under a player's robot, up to 16 px wide. Sizes for a 32px wide tile, scaled to the frame. */
export function playerShadow(frame: TileFrame): Ellipse {
  return ellipseAtGround(frame, 6, 2.5);
}

/** Your ring: wider than the shadow, so it shows around it. */
export function yourRing(frame: TileFrame): Ellipse {
  return ellipseAtGround(frame, 8, 3.5);
}

/** The usual semi-transparent ground shadow, at the frame's ground center. */
export function paintPlayerShadow(frame: TileFrame): PixelBuffer {
  const buffer = emptyBuffer(frame);
  paintGroundShadow(buffer, frame, playerShadow(frame));
  return buffer;
}

/** Your ring: a one-pixel yellow outline of an ellipse around the ground center, empty inside. */
export function paintYourRing(frame: TileFrame): PixelBuffer {
  const buffer = emptyBuffer(frame);
  const ring = yourRing(frame);
  const inside = (x: number, y: number) => isInEllipse(ring, x, y);
  forEachPixel(buffer.width, buffer.height, (x, y) => {
    if (isShapeEdge(inside, x, y)) setPixel(buffer, x, y, YOUR_RING_RGB);
  });
  return buffer;
}

function ellipseAtGround(frame: TileFrame, rx: number, ry: number): Ellipse {
  const ground = groundCenter(frame);
  const scale = frame.tileWidth / 32;
  return { cx: ground.x, cy: ground.y, rx: rx * scale, ry: ry * scale };
}

function emptyBuffer(frame: TileFrame): PixelBuffer {
  const { width, height } = textureSize(frame);
  return createPixelBuffer(width, height);
}

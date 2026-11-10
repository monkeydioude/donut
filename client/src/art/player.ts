import { isInEllipse, isShapeEdge, paintGroundShadow, type Ellipse } from './drawing';
import { createPixelBuffer, forEachPixel, setPixel, type PixelBuffer } from './pixelBuffer';
import { groundCenter, isDrawable, textureSize, type TileFrame } from './tileFrame';

/** Your player and the others only differ by their clothes. */
export type PlayerLook = 'you' | 'other';

export const PLAYER_LOOKS: Readonly<Record<PlayerLook, { shirt: number; shirtShade: number }>> = {
  you: { shirt: 0xf08a6c, shirtShade: 0xc96a52 },
  other: { shirt: 0x6c9bf0, shirtShade: 0x5276c9 },
};

const BODY = { skin: 0xf6d2b0, hair: 0x5a3d35, outline: 0x3f3550, eye: 0x3f3550 };

/** Placeholder character standing on the frame's ground center, facing the viewer. Sizes for a 32px wide tile. */
export function paintPlayer(frame: TileFrame, look: PlayerLook): PixelBuffer {
  const { width, height } = textureSize(frame);
  const buffer = createPixelBuffer(width, height);
  const parts = layoutParts(frame);
  paintGroundShadow(buffer, frame, parts.shadow);

  const inBody = (x: number, y: number) => isInEllipse(parts.body, x, y) && y + 0.5 <= parts.feetY;
  const inHead = (x: number, y: number) => isInEllipse(parts.head, x, y);
  const inside = (x: number, y: number) => isDrawable(frame, x, y) && (inBody(x, y) || inHead(x, y));
  forEachPixel(width, height, (x, y) => {
    if (!inside(x, y)) return;
    const rgb = isShapeEdge(inside, x, y)
      ? BODY.outline
      : inHead(x, y)
        ? headColor(parts.head, y)
        : shirtColor(parts.body, x, look);
    setPixel(buffer, x, y, rgb);
  });
  for (const eye of parts.eyes) setPixel(buffer, eye.x, eye.y, BODY.eye);
  return buffer;
}

function layoutParts(frame: TileFrame) {
  const ground = groundCenter(frame);
  const scale = frame.tileWidth / 32;
  const head = { cx: ground.x, cy: ground.y - 11 * scale, rx: 4.5 * scale, ry: 4.5 * scale };
  const eyeY = Math.floor(head.cy + 0.5 * scale);
  return {
    shadow: { cx: ground.x, cy: ground.y, rx: 5 * scale, ry: 2 * scale },
    body: { cx: ground.x, cy: ground.y - 4 * scale, rx: 4 * scale, ry: 4.5 * scale },
    feetY: ground.y,
    head,
    eyes: [
      { x: Math.floor(ground.x - 2 * scale), y: eyeY },
      { x: Math.floor(ground.x + 1 * scale), y: eyeY },
    ],
  };
}

/** Hair on the top of the head, skin below. */
function headColor(head: Ellipse, y: number): number {
  return y + 0.5 < head.cy - head.ry * 0.15 ? BODY.hair : BODY.skin;
}

/** The shirt is lit from the left. */
function shirtColor(body: Ellipse, x: number, look: PlayerLook): number {
  const colors = PLAYER_LOOKS[look];
  return x + 0.5 > body.cx + body.rx * 0.35 ? colors.shirtShade : colors.shirt;
}

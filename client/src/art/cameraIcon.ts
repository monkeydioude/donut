import { bufferFromRows, stampRows } from './drawing';
import type { PixelBuffer } from './pixelBuffer';

/**
 * The camera lock's icon (ADR 0004): a cream camera seen from the front, its lens top center, and a gold padlock
 * in front of its bottom-right corner when locked, cut out from it by a transparent halo.
 * o: outline, b: body, s: body shade, d: lens, h: lens highlight, r: record dot, y: gold, g: gold shade,
 * x: erased (the padlock's halo and hole).
 */
const CAMERA_ROWS = [
  '......oooooo...........',
  '.oooooobbbbooooooo.....',
  '.obbbbbbbbbbbrrbbo.....',
  '.obbbbbooooobbbbbo.....',
  '.obbbbodddddobbbbo.....',
  '.obbbodhhddddobbbo.....',
  '.obbbodhdddddobbbo.....',
  '.obbbodddddddobbbo.....',
  '.osssodddddddossso.....',
  '.ossssodddddosssso.....',
  '.osssssooooossssso.....',
  '.ossssssssssssssso.....',
  '.ooooooooooooooooo.....',
  '.......................',
  '.......................',
  '.......................',
];

const PADLOCK_ROWS = [
  '..xxxxx..',
  '.xxoooxx.',
  'xxoyyyoxx',
  'xoyoxoyox',
  'xooooooox',
  'xoyyyyyox',
  'xoyyoyyox',
  'xoggoggox',
  'xooooooox',
  'xxxxxxxxx',
];

/** Where the padlock and its halo go on the camera. */
export const PADLOCK_BOX = { x: 14, y: 6, width: 9, height: 10 } as const;

const PALETTE = {
  o: 0x3a2e4f,
  b: 0xfff4dc,
  s: 0xe6d3b3,
  d: 0x4a3f63,
  h: 0xbfe6ff,
  r: 0xff7a93,
  y: 0xffd84a,
  g: 0xe0a93a,
};

export function paintCameraIcon(locked: boolean): PixelBuffer {
  const icon = bufferFromRows(CAMERA_ROWS, PALETTE);
  if (locked) stampRows(icon, PADLOCK_ROWS, { ...PALETTE, x: null }, PADLOCK_BOX);
  return icon;
}

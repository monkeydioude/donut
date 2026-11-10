import { bufferFromRows } from './drawing';
import type { PixelBuffer } from './pixelBuffer';

/**
 * The mouse pointer: a white glove pointing up-left with its index.
 * Taken from an image the user supplied (a JRPG-style cursor), its pixels recovered from the upscaled image and
 * cleaned: the blur around the outline and the pixels sticking out of the cuff removed. Check its license before
 * any public release.
 * o: outline, w: white, g: light shade, s: shade.
 */
const FINGER_ROWS = [
  '..ooooo.............',
  '.oogggoo............',
  'oosgwwgoo...........',
  'oosgwwwgoo..........',
  '.ossgwwwgoo.ooo.....',
  '.oossgwwwgoooooo....',
  '..oossgwwwgooggoo...',
  '...oossgwwwgogwgoo..',
  '...ooosgwwwwgswwgoo.',
  '.oogggggwwwwwgwwgoo.',
  '.oogwwwwwwwwwwwwgoo.',
  '.oogwwwwwwwwwwwggoo.',
  '.osggwwwwwwwwwwwgoo.',
  '.osggwwwwwwggwwwgooo',
  '.osggwwwwwwwwwwggggo',
  '.osggwwwwwwwwwgwwggo',
  '.ossggwwwwwwwggwwgo.',
  '.osssggwwwwwggwwgo..',
  '.oosssgggggggwwgo...',
  '..oossssssssgggo....',
  '...oossssosgggo.....',
  '....oooooossso......',
  '.....oooooooo.......',
];

const FINGER_PALETTE = { o: 0x18181f, w: 0xffffff, g: 0xd8d8e8, s: 0xa8a8c0 };

/** The fingertip: the pixel that points. */
export const FINGER_HOTSPOT = { x: 1, y: 1 } as const;

export function paintFinger(): PixelBuffer {
  return bufferFromRows(FINGER_ROWS, FINGER_PALETTE);
}

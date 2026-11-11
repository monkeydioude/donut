import type { DonutShape } from '../game/mapMarks';
import { bufferFromRows } from './drawing';
import type { PixelBuffer } from './pixelBuffer';

/** The map's marks (ADR 0005): yours in gold, the other players' in cream, all with a dark outline. */
const PALETTE = { o: 0x3a2e4f, y: 0xffd84a, c: 0xfff4dc };

const YOU: Record<DonutShape, readonly string[]> = {
  square: ['ooooo', 'oyyyo', 'oyyyo', 'oyyyo', 'ooooo'],
  up: ['...o...', '..oyo..', '.oyyyo.', 'oyyyyyo', 'ooooooo'],
  down: ['ooooooo', 'oyyyyyo', '.oyyyo.', '..oyo..', '...o...'],
};

const OTHER = ['.oo.', 'occo', 'occo', '.oo.'];

/** Your mark: a square on the rectangle; on the donut, a square, ▲ or ▼. */
export function paintYouMark(shape: DonutShape): PixelBuffer {
  return bufferFromRows(YOU[shape], PALETTE);
}

/** Another player you see, on the rectangle. */
export function paintOtherMark(): PixelBuffer {
  return bufferFromRows(OTHER, PALETTE);
}

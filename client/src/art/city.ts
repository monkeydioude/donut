import { hashCell } from '../core/rng';
import type { FaceSide } from './cliff';
import { createPixelBuffer, forEachPixel, setPixel, type PixelBuffer } from './pixelBuffer';
import { groundCenter, isOnGround, textureSize, type TileFrame } from './tileFrame';

/**
 * A dome city's art (ADR 0012): worn pale paving, the dome's metal ring on top of its wall, and its old hexagonal glass
 * panels on the wall's faces, some missing, some cracked. Light from the top left, as everywhere.
 */

/** How a floor cell is worn, picked from its place so everyone sees the same: most are plain. */
export type FloorWear = 'none' | 'crack' | 'stain' | 'missing';

export interface FloorLook {
  wear: FloorWear;
  /** Which of the cell's 4 slabs is worn: 0 back, 1 right, 2 left, 3 front. */
  slab: number;
}

/** How a level of glass face looks: its honeycomb's offset, and which of its panels is missing or cracked (−1: none). */
export interface GlassLook {
  offset: 0 | 4;
  missing: number;
  cracked: number;
}

const FLOOR = { stone: 0xdcd5c6, seam: 0xbfb6a4, crack: 0x8f8678, stain: 0xc6bdab, sunken: 0xa39b8c, sunkenEdge: 0x7f7769 } as const;
const WALL = { top: 0x9a96a3, lit: 0xbdb9c5, dark: 0x77737f, rivet: 0x625e6c } as const;
const GLASS = {
  left: { pane: 0x9fd0c9, light: 0xd2ece6, frame: 0x6d6876 },
  right: { pane: 0x7fb4b1, light: 0xaad5cf, frame: 0x57525f },
  inside: 0x2f2838,
  crack: 0xe6f5f0,
} as const;
/** The honeycomb's cells are this many pixels across: a face's 16 pixels hold two, and a level one row of them. */
const HEX_PX = 8;

/** A floor cell: 4 pale slabs with seams between them, and its wear. */
export function paintFloor(frame: TileFrame, look: FloorLook): PixelBuffer {
  const { width, height } = textureSize(frame);
  const buffer = createPixelBuffer(width, height);
  const slab = (x: number, y: number) => (isOnGround(frame, x, y) ? slabAt(frame, x, y) : -1);
  forEachPixel(width, height, (x, y) => {
    const here = slab(x, y);
    if (here < 0) return;
    const onSeam = [slab(x + 1, y), slab(x, y + 1)].some((next) => next >= 0 && next !== here);
    setPixel(buffer, x, y, onSeam ? FLOOR.seam : floorColor(here, look, slab(x, y - 1) !== here));
  });
  if (look.wear === 'crack') paintCrack(buffer, frame, look.slab, slab);
  if (look.wear === 'stain') paintStain(buffer, frame, look.slab, slab);
  return buffer;
}

/** Which slab a ground pixel is on: the cell cut in 4 along its middle, as cells are cut. */
function slabAt(frame: TileFrame, x: number, y: number): number {
  const center = groundCenter(frame);
  const across = (x + 0.5 - center.x) / (frame.tileWidth / 2);
  const down = (y + 0.5 - center.y) / (frame.tileHeight / 2);
  const col = (across + down) / 2;
  const row = (down - across) / 2;
  return (col < 0 ? 0 : 1) + (row < 0 ? 0 : 2);
}

/** A missing slab is sunken and darker, with a shadow along its back edge. */
function floorColor(slab: number, look: FloorLook, backEdge: boolean): number {
  if (look.wear !== 'missing' || slab !== look.slab) return FLOOR.stone;
  return backEdge ? FLOOR.sunkenEdge : FLOOR.sunken;
}

/** A short crack across a slab, from near its middle. */
function paintCrack(buffer: PixelBuffer, frame: TileFrame, slab: number, slabOf: (x: number, y: number) => number): void {
  const start = slabMiddle(frame, slab);
  const steps = [
    [0, 0],
    [1, 0],
    [2, 1],
    [3, 1],
    [4, 2],
    [-1, 0],
    [-2, -1],
  ] as const;
  for (const [dx, dy] of steps) {
    const x = start.x + dx;
    const y = start.y + dy;
    if (slabOf(x, y) === slab && slabOf(x + 1, y) === slab && slabOf(x, y + 1) === slab) setPixel(buffer, x, y, FLOOR.crack);
  }
}

/** A soft darker patch on a slab. */
function paintStain(buffer: PixelBuffer, frame: TileFrame, slab: number, slabOf: (x: number, y: number) => number): void {
  const middle = slabMiddle(frame, slab);
  forEachPixel(buffer.width, buffer.height, (x, y) => {
    const u = (x + 0.5 - middle.x - 0.5) / 3;
    const v = (y + 0.5 - middle.y - 0.5) / 1.5;
    if (u * u + v * v <= 1 && slabOf(x, y) === slab && slabOf(x + 1, y) === slab && slabOf(x, y + 1) === slab) setPixel(buffer, x, y, FLOOR.stain);
  });
}

/** The middle of a slab, in texture pixels. */
function slabMiddle(frame: TileFrame, slab: number): { x: number; y: number } {
  const center = groundCenter(frame);
  const quarterCol = slab % 2 === 0 ? -0.25 : 0.25;
  const quarterRow = slab < 2 ? -0.25 : 0.25;
  return {
    x: Math.floor(center.x + (quarterCol - quarterRow) * (frame.tileWidth / 2)),
    y: Math.floor(center.y + (quarterCol + quarterRow) * (frame.tileHeight / 2)),
  };
}

/** The top of the dome's wall: its metal ring, lit on its back edges, with a rivet in the middle. */
export function paintWallTop(frame: TileFrame): PixelBuffer {
  const { width, height } = textureSize(frame);
  const buffer = createPixelBuffer(width, height);
  const center = groundCenter(frame);
  forEachPixel(width, height, (x, y) => {
    if (!isOnGround(frame, x, y)) return;
    const onEdge = !isOnGround(frame, x, y - 1) || !isOnGround(frame, x, y + 1);
    const back = y + 0.5 < center.y;
    setPixel(buffer, x, y, onEdge ? (back ? WALL.lit : WALL.dark) : WALL.top);
  });
  setPixel(buffer, Math.floor(center.x), Math.floor(center.y), WALL.rivet);
  setPixel(buffer, Math.floor(center.x) - 1, Math.floor(center.y), WALL.rivet);
  return buffer;
}

/**
 * One level of the dome's wall face, as `paintFace` lays a level out: hexagonal glass panels in metal frames, a row of
 * them per level, shifted half a panel every other level so the rows make a honeycomb. A missing panel shows the dark
 * outside; a cracked one a bright line.
 */
export function paintGlassFace(frame: TileFrame, levelPx: number, side: FaceSide, look: GlassLook): PixelBuffer {
  const buffer = createPixelBuffer(frame.tileWidth, frame.tileHeight + frame.headroom + levelPx);
  const half = frame.tileWidth / 2;
  for (let x = side === 'left' ? 0 : half; x < (side === 'left' ? half : frame.tileWidth); x++) {
    const bottom = lowestGroundPixel(frame, x);
    if (bottom === null) continue;
    for (let i = 0; i < levelPx; i++) setPixel(buffer, x, bottom + 1 + i, glassPixel(x, (i * HEX_PX) / levelPx, side, look));
  }
  return buffer;
}

/** A glass face's pixel, `y` from 0 at the level's top to HEX_PX at its bottom. */
function glassPixel(x: number, y: number, side: FaceSide, look: GlassLook): number {
  const colors = GLASS[side];
  const u = (x + look.offset) % HEX_PX;
  const panel = Math.floor((x + look.offset) / HEX_PX);
  const top = Math.floor(Math.abs(u - HEX_PX / 2) / 2);
  const row = Math.floor(y);
  if (row === top || row === HEX_PX - 1 - top || (u === 0 && row > top && row < HEX_PX - 1 - top)) return colors.frame;
  // above its top edge or below its bottom edge, a pixel is a corner of the next row's panel: always whole glass
  const inPanel = row > top && row < HEX_PX - 1 - top;
  if (inPanel && panel === look.missing) return GLASS.inside;
  if (inPanel && panel === look.cracked && u - 1 === row - top) return GLASS.crack;
  if (inPanel && row === top + 1 && (u === 2 || u === 3)) return colors.light;
  return colors.pane;
}

/** The lowest pixel row of the ground diamond in a texture column, null when none. */
function lowestGroundPixel(frame: TileFrame, x: number): number | null {
  for (let y = frame.headroom + frame.tileHeight - 1; y >= 0; y--) {
    if (isOnGround(frame, x, y)) return y;
  }
  return null;
}

/** Salts keep the floor's and the glass's picks apart from each other and from the other layers. */
const FLOOR_SALT = 11;
const GLASS_SALT = 12;

/** A floor cell's look, from its place: about 1 in 12 cracked, 1 in 16 stained, 1 in 25 with a slab missing. */
export function floorLookOf(cell: { col: number; row: number }): FloorLook {
  const hash = hashCell(cell.col, cell.row, FLOOR_SALT);
  const roll = hash % 100;
  const wear: FloorWear = roll < 8 ? 'crack' : roll < 14 ? 'stain' : roll < 18 ? 'missing' : 'none';
  return { wear, slab: (hash >>> 8) % 4 };
}

/**
 * A level of a wall cell's face, from its place: its honeycomb's offset follows where the level is drawn, so rows meet
 * across cells and levels; about 1 in 6 levels has a panel missing, and 1 in 6 one cracked.
 */
export function glassLookOf(cell: { col: number; row: number }, height: number, level: number): GlassLook {
  const drawnRow = cell.col + cell.row - height + level;
  const hash = hashCell(cell.col, cell.row * 16 + level, GLASS_SALT);
  const roll = hash % 100;
  const panel = (hash >>> 8) % 3;
  return { offset: drawnRow % 2 === 0 ? 0 : 4, missing: roll < 16 ? panel : -1, cracked: roll >= 16 && roll < 32 ? panel : -1 };
}

/** A key for a look, to paint it once. */
export function glassKey(side: FaceSide, look: GlassLook): string {
  return `glass ${side} ${look.offset} ${look.missing} ${look.cracked}`;
}

export function floorKey(look: FloorLook): string {
  return `floor ${look.wear} ${look.wear === 'none' ? 0 : look.slab}`;
}

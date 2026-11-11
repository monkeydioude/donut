import type { TilePoint } from '../iso/projection';
import { forEachBlockPixel, isOnEdge, shadeIndex, type BlockPixel, type BlockShape } from './block';
import { FACE_COLORS, ICE_STREAKS, isIceStreak, type FaceSide } from './cliff';
import type { Edge } from './edges';
import { facetAt, patternPoint, scatteredSeeds, type FacetHit } from './facets';
import { createPixelBuffer, setPixel, type PixelBuffer } from './pixelBuffer';
import { textureSize, type TileFrame } from './tileFrame';

/** The ice's top from its darkest shade to its lightest, lit by how its slope faces the light, like the rock's. */
export const GLACIER_COLORS = {
  top: [0x5fb4d8, 0x7dcbe5, 0x9fe4f1, 0xc2f2f9],
  crack: 0x74c3df,
  gloss: 0xeafcff,
  edge: 0xd4f7fd,
} as const;

/** The ice's plates: 4 × 4 over the 8 × 8 cells of its pattern, bigger than the rock's facets, 2 shades. */
const ICE_SEEDS = scatteredSeeds(4, 2, 23);
/** Cracks run only between plates of different shades: the others melt into one, so it reads as ice, not tiles. */
const CRACK = 0.04;
/** Each plate's glint: two short "/" streaks up-left of its seed, in screen pixels from the seed. */
const GLINTS = [
  { x: -6, y: -3, halfLength: 3 },
  { x: -2, y: -1, halfLength: 1.5 },
] as const;

/** How a glacier cell looks (ADR 0011): its place in the pattern, its edges with a bright line, its shape. */
export interface GlacierLook {
  place: TilePoint;
  edges: readonly Edge[];
  shape: BlockShape;
}

/**
 * A glacier tile (ADR 0011): one block of a merged ice mass. Its top leans up to the higher ice behind, cyan-teal lit
 * by how it faces the light, in large plates with soft cracks and a glint, laid on the map so neighbors join, a thin
 * bright line on the edges given. Its sides are saturated blue with vertical streaks.
 */
export function paintGlacier(frame: TileFrame, look: GlacierLook, levelPx: number): PixelBuffer {
  const { width, height } = textureSize(frame);
  const buffer = createPixelBuffer(width, height);
  forEachBlockPixel(frame, look.shape, levelPx, (x, y, pixel) => setPixel(buffer, x, y, glacierColor(look, x, pixel)));
  return buffer;
}

function glacierColor(look: GlacierLook, x: number, pixel: BlockPixel): number {
  if (pixel.part === 'side') return iceSideColor(x, pixel.side);
  if (isOnEdge(pixel.at, look.edges)) return GLACIER_COLORS.edge;
  return iceTopColor(patternPoint(look.place, pixel.at), pixel.light);
}

/**
 * The ice's top at a point of the pattern: one shade per slope, from its light, so the steps stand out; soft cracks
 * between plates of different shades, a glint on the others. Also the icy tops of the tallest mountains.
 */
export function iceTopColor(point: TilePoint, light: number): number {
  const plate = facetAt(ICE_SEEDS, point);
  if (plate.border < CRACK && plate.nextShade !== plate.seed.shade) return GLACIER_COLORS.crack;
  if (plate.seed.shade === 0 && isGlint(plate)) return GLACIER_COLORS.gloss;
  return GLACIER_COLORS.top[shadeIndex(light, GLACIER_COLORS.top.length)] ?? GLACIER_COLORS.top[0];
}

/** A side: the ice faces' blue, a lighter streak every few pixels, like the faces under it. */
function iceSideColor(x: number, side: FaceSide): number {
  return isIceStreak(x) ? ICE_STREAKS[side] : FACE_COLORS.ice[side].face;
}

/** On a "/" streak near the plate's seed, in screen pixels (at a 32 × 16 tile): as x goes right, y goes up. */
function isGlint(plate: FacetHit): boolean {
  const x = (plate.offset.col - plate.offset.row) * 16;
  const y = (plate.offset.col + plate.offset.row) * 8;
  return GLINTS.some((glint) => Math.abs(x - glint.x + (y - glint.y)) < 0.8 && Math.abs(x - glint.x) < glint.halfLength);
}

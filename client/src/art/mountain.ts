import type { TilePoint } from '../iso/projection';
import { forEachBlockPixel, isOnEdge, shadeIndex, type BlockPixel, type BlockShape } from './block';
import type { FaceSide } from './cliff';
import type { Edge } from './edges';
import { facetAt, patternPoint, scatteredSeeds } from './facets';
import { GLACIER_COLORS, iceTopColor } from './glacier';
import { createPixelBuffer, setPixel, type PixelBuffer } from './pixelBuffer';
import { textureSize, type TileFrame } from './tileFrame';

/** The rock's top from its darkest shade to its lightest, lit by how its slope faces the light. */
export const MOUNTAIN_COLORS = {
  top: [0x6c6879, 0x7f7b8c, 0x938fa0, 0xa8a4b4],
  side: { left: [0x8a8694, 0x807c8b], right: [0x736f7d, 0x6a6674] },
  crack: 0x5f5a6b,
  outline: 0x4f4a57,
} as const;

/** The rock's facets: 6 × 6 over the 8 × 8 cells of its pattern, a darker one for half of them. */
const ROCK_SEEDS = scatteredSeeds(6, 2, 11);
/** How close to a facet's border, in cells, a crack is. */
const CRACK = 0.06;

/** How a mountain cell looks (ADR 0011): its place in the pattern, its outlined edges, its shape, an icy top. */
export interface MountainLook {
  place: TilePoint;
  outline: readonly Edge[];
  shape: BlockShape;
  /** The tops at +6 are drawn in ice: cosmetic, still a mountain. */
  iced: boolean;
}

/**
 * A mountain tile (ADR 0011): one block of a merged massif. Its top leans up to the higher rock behind, or rises to a
 * point on a peak, gray-purple rock lit by how it faces the light, in large facets with cracks between them laid on
 * the map so neighbors join, a dark outline on the edges given. Its sides are stone, the cracks running down them.
 */
export function paintMountain(frame: TileFrame, look: MountainLook, levelPx: number): PixelBuffer {
  const { width, height } = textureSize(frame);
  const buffer = createPixelBuffer(width, height);
  forEachBlockPixel(frame, look.shape, levelPx, (x, y, pixel) => setPixel(buffer, x, y, mountainColor(look, pixel)));
  return buffer;
}

function mountainColor(look: MountainLook, pixel: BlockPixel): number {
  const point = patternPoint(look.place, pixel.at);
  if (pixel.part === 'side') return sideColor(point, pixel.side);
  if (isOnEdge(pixel.at, look.outline)) return look.iced ? GLACIER_COLORS.edge : MOUNTAIN_COLORS.outline;
  return look.iced ? iceTopColor(point, pixel.light) : rockTopColor(point, pixel.light);
}

/** The rock at a point: its light's shade, one darker on half the facets, cracks on their borders. */
function rockTopColor(point: TilePoint, light: number): number {
  const facet = facetAt(ROCK_SEEDS, point);
  if (facet.border < CRACK) return MOUNTAIN_COLORS.crack;
  const shade = Math.max(0, shadeIndex(light, MOUNTAIN_COLORS.top.length) - facet.seed.shade);
  return MOUNTAIN_COLORS.top[shade] ?? MOUNTAIN_COLORS.top[0];
}

/** A side under the top's front edge: its facet's stone, darker for half of them, the cracks running straight down. */
function sideColor(point: TilePoint, side: FaceSide): number {
  const facet = facetAt(ROCK_SEEDS, point);
  if (facet.border < CRACK) return MOUNTAIN_COLORS.crack;
  return MOUNTAIN_COLORS.side[side][facet.seed.shade] ?? MOUNTAIN_COLORS.side[side][0];
}

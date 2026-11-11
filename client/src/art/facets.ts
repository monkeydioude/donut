import { createSeededRng } from '../core/rng';
import type { TilePoint } from '../iso/projection';

/** How many cells the rock's and the ice's patterns span before they repeat, each way (ADR 0011). */
export const PATTERN_CELLS = 8;

/** A cell's place in the repeating pattern: its column and row, modulo PATTERN_CELLS. */
export function patternPlace(cell: TilePoint): TilePoint {
  return { col: mod(cell.col, PATTERN_CELLS), row: mod(cell.row, PATTERN_CELLS) };
}

/** A point of the pattern: a cell's place plus a point within the cell (−0.5 to 0.5 each way, 0 at its center). */
export function patternPoint(place: TilePoint, within: TilePoint): TilePoint {
  return { col: place.col + within.col, row: place.row + within.row };
}

/** A facet's seed: where it is in the pattern, in cells, and its shade, an index into the caller's colors. */
export interface Seed {
  at: TilePoint;
  shade: number;
}

/**
 * Seeds spread over the pattern without clumps: one per square of a grid `perSide` squares a side, at a random place in
 * its square, with a random shade out of `shades`. The same rng seed gives the same seeds.
 */
export function scatteredSeeds(perSide: number, shades: number, rngSeed: number): Seed[] {
  const rng = createSeededRng(rngSeed);
  const square = PATTERN_CELLS / perSide;
  const seeds: Seed[] = [];
  for (let j = 0; j < perSide; j++) {
    for (let i = 0; i < perSide; i++) {
      const at = { col: (i + 0.15 + 0.7 * rng()) * square, row: (j + 0.15 + 0.7 * rng()) * square };
      seeds.push({ at, shade: Math.floor(rng() * shades) });
    }
  }
  return seeds;
}

/** Which facet a point is in, how far it is past the border to the next one (in cells), and where its seed is. */
export interface FacetHit {
  seed: Seed;
  /** From the facet's seed, across the seams. */
  offset: TilePoint;
  /** How much closer the nearest seed is than the second nearest: small on a border, where cracks go. */
  border: number;
  /** The shade of the facet across the nearest border. */
  nextShade: number;
}

/**
 * The facet holding a point of the pattern: the nearest seed, across the pattern's seams, so the facets tile the map
 * without a seam. Distances are measured on the map, in cells.
 */
export function facetAt(seeds: readonly Seed[], point: TilePoint): FacetHit {
  let nearest = { seed: seeds[0] as Seed, offset: { col: 0, row: 0 }, distance: Infinity };
  let second = { shade: 0, distance: Infinity };
  for (const seed of seeds) {
    const offset = { col: wrapped(point.col - seed.at.col), row: wrapped(point.row - seed.at.row) };
    const distance = Math.hypot(offset.col, offset.row);
    if (distance < nearest.distance) {
      second = { shade: nearest.seed.shade, distance: nearest.distance };
      nearest = { seed, offset, distance };
    } else if (distance < second.distance) {
      second = { shade: seed.shade, distance };
    }
  }
  return { seed: nearest.seed, offset: nearest.offset, border: second.distance - nearest.distance, nextShade: second.shade };
}

/** A difference along the pattern, brought to the shortest way round: −PATTERN_CELLS / 2 to PATTERN_CELLS / 2. */
function wrapped(delta: number): number {
  return delta - PATTERN_CELLS * Math.round(delta / PATTERN_CELLS);
}

function mod(value: number, by: number): number {
  return ((value % by) + by) % by;
}

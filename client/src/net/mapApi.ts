import { CHUNK_SIZE } from '../map/grid';
import type { MapDescription, Place, Region } from '../map/worldMap';
import { isInt, isNonEmptyString, isRecord, type Json } from './validate';

const MAX_SIZE = 4096;
/** A world cell stands for a local map of this side (ADR 0012). */
const LOCAL_SIDE = 16;

/** Where the map's picture is (ADR 0005): the browser loads it, and keeps it by its ETag. */
export function mapImageUrl(slug: string): string {
  return `/api/maps/${encodeURIComponent(slug)}/image.png`;
}

/**
 * Checks a map's description, as `welcome` carries it (ADR 0012). Throws on anything invalid, or a chunk size the
 * client doesn't draw.
 */
export function parseDescription(json: unknown): MapDescription {
  if (!isRecord(json) || !isNonEmptyString(json.slug) || typeof json.name !== 'string') throw invalid('missing slug or name');
  const { width, height, wraps, chunkSize, regions, standsOn } = json;
  if (typeof wraps !== 'boolean' || !isSize(width, wraps) || !isSize(height, wraps)) throw invalid('bad size');
  if (chunkSize !== CHUNK_SIZE) throw invalid(`chunks of ${String(chunkSize)} cells, expected ${CHUNK_SIZE}`);
  if (!Array.isArray(regions) || !regions.every(isRegion)) throw invalid('bad regions');
  if (standsOn !== null && standsOn !== undefined && !isPlace(standsOn)) throw invalid('bad standsOn');
  return {
    slug: json.slug,
    name: json.name,
    width,
    height,
    wraps,
    chunkSize,
    regions: regions.map(({ id, name, biome }) => ({ id, name, biome })),
    standsOn: isPlace(standsOn) ? { map: standsOn.map, col: standsOn.col, row: standsOn.row, width: standsOn.width, height: standsOn.height } : null,
  };
}

/** A side of a map: a multiple of a chunk when it wraps, of a world cell's local map (16) when it doesn't (ADR 0012). */
function isSize(value: unknown, wraps: boolean): value is number {
  const step = wraps ? CHUNK_SIZE : LOCAL_SIDE;
  return isInt(value) && value >= step && value <= MAX_SIZE && value % step === 0;
}

/** Where a city stands: a cell of another map, which wraps and is of a valid size. */
function isPlace(value: unknown): value is Place & Json {
  return (
    isRecord(value) &&
    isNonEmptyString(value.map) &&
    isSize(value.width, true) &&
    isSize(value.height, true) &&
    isInt(value.col) &&
    isInt(value.row) &&
    value.col >= 0 &&
    value.row >= 0 &&
    value.col < value.width &&
    value.row < value.height
  );
}

function isRegion(value: unknown): value is Region & Json {
  return isRecord(value) && isInt(value.id) && value.id >= 1 && value.id <= 255 && isNonEmptyString(value.name) && typeof value.biome === 'string';
}

function invalid(reason: string): Error {
  return new Error(`invalid map description: ${reason}`);
}

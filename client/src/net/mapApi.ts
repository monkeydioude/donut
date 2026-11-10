import { silent, type Log } from '../debug/log';
import { CHUNK_SIZE } from '../map/grid';
import type { MapDescription, Region } from '../map/worldMap';
import { isInt, isNonEmptyString, isRecord, type Json } from './validate';

export type FetchFn = (url: string) => Promise<Response>;

const MAX_SIZE = 4096;

/** Loads a map's description from the back (GET /api/maps/{slug}): its size and regions, without its cells. */
export async function fetchDescription(slug: string, fetchFn: FetchFn, log: Log = silent): Promise<MapDescription> {
  const url = `/api/maps/${encodeURIComponent(slug)}`;
  const started = performance.now();
  log('GET', url);
  const response = await fetchFn(url);
  log('answered', { status: response.status, ms: Math.round(performance.now() - started) });
  if (!response.ok) throw new Error(`fetchDescription: ${slug} answered ${response.status}`);
  const map = parseDescription(await response.json());
  log('description loaded', { slug: map.slug, size: `${map.width}x${map.height}`, wraps: map.wraps, regions: map.regions.length });
  return map;
}

/** Checks the server's description JSON. Throws on anything invalid, or a chunk size the client doesn't draw. */
export function parseDescription(json: unknown): MapDescription {
  if (!isRecord(json) || !isNonEmptyString(json.slug) || typeof json.name !== 'string') throw invalid('missing slug or name');
  const { width, height, wraps, chunkSize, regions } = json;
  if (!isSize(width) || !isSize(height) || typeof wraps !== 'boolean') throw invalid('bad size');
  if (chunkSize !== CHUNK_SIZE) throw invalid(`chunks of ${String(chunkSize)} cells, expected ${CHUNK_SIZE}`);
  if (!Array.isArray(regions) || !regions.every(isRegion)) throw invalid('bad regions');
  return { slug: json.slug, name: json.name, width, height, wraps, chunkSize, regions: regions.map(({ id, name, biome }) => ({ id, name, biome })) };
}

function isSize(value: unknown): value is number {
  return isInt(value) && value >= CHUNK_SIZE && value <= MAX_SIZE && value % CHUNK_SIZE === 0;
}

function isRegion(value: unknown): value is Region & Json {
  return isRecord(value) && isInt(value.id) && value.id >= 1 && value.id <= 255 && isNonEmptyString(value.name) && typeof value.biome === 'string';
}

function invalid(reason: string): Error {
  return new Error(`invalid map description: ${reason}`);
}

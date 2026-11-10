import type { Chunk } from '../map/chunks';
import { CELLS_PER_CHUNK, CHUNK_SIZE, chunkOrigin, type ChunkKey } from '../map/grid';
import { GROUND_KINDS, OBJECT_KINDS, type GroundKind, type MapObject } from '../map/worldMap';
import { isInt, isNonEmptyString, isOneOf, isRecord, type Json } from './validate';

export interface Cell {
  col: number;
  row: number;
}

/** A player on the map, as the server tells it. */
export interface PlayerInfo extends Cell {
  id: string;
  name: string;
}

/** What the server sends over the websocket (ADR 0001, ADR 0002, ADR 0003). */
export type ServerMessage =
  | { type: 'chunk'; chunk: Chunk }
  | { type: 'forget'; cx: number; cy: number }
  | { type: 'welcome'; you: string; players: PlayerInfo[]; stepMs: number }
  | { type: 'joined'; player: PlayerInfo }
  | { type: 'moved'; id: string; col: number; row: number }
  | { type: 'left'; id: string }
  | { type: 'planned'; col: number; row: number; path: Cell[] }
  | { type: 'walking'; path: Cell[] }
  | { type: 'stopped'; col: number; row: number };

export interface Guest {
  id: string;
  name: string;
}

/** Decodes a server message; anything unknown or malformed gives null. */
export function decodeServerMessage(text: string): ServerMessage | null {
  let json: unknown;
  try {
    json = JSON.parse(text);
  } catch {
    return null;
  }
  return isRecord(json) ? decodeByType(json) : null;
}

/** Asks the path from where your player stands to a cell. */
export function encodePlan(to: Cell): string {
  return JSON.stringify({ type: 'plan', col: to.col, row: to.row });
}

/** Asks the server to walk your player to a cell. */
export function encodeWalk(to: Cell): string {
  return JSON.stringify({ type: 'walk', col: to.col, row: to.row });
}

/** The websocket address of a map, on the page's own host (Vite proxies it to the back in dev). */
export function socketUrl(page: { protocol: string; host: string }, slug: string, guest: Guest): string {
  const scheme = page.protocol === 'https:' ? 'wss' : 'ws';
  const query = new URLSearchParams({ player: guest.id, name: guest.name });
  return `${scheme}://${page.host}/api/maps/${encodeURIComponent(slug)}/ws?${query}`;
}

function decodeByType(json: Record<string, unknown>): ServerMessage | null {
  switch (json.type) {
    case 'welcome':
      return decodeWelcome(json);
    case 'joined':
      return isPlayerInfo(json.player) ? { type: 'joined', player: json.player } : null;
    case 'moved': {
      const { id } = json;
      return isNonEmptyString(id) && isCell(json) ? { type: 'moved', id, col: json.col, row: json.row } : null;
    }
    case 'left':
      return isNonEmptyString(json.id) ? { type: 'left', id: json.id } : null;
    case 'planned': {
      const { path } = json;
      return isCell(json) && isPath(path) ? { type: 'planned', col: json.col, row: json.row, path } : null;
    }
    case 'walking':
      return isPath(json.path) ? { type: 'walking', path: json.path } : null;
    case 'stopped':
      return isCell(json) ? { type: 'stopped', col: json.col, row: json.row } : null;
    case 'chunk':
      return decodeChunk(json);
    case 'forget':
      return isChunkKey(json) ? { type: 'forget', cx: json.cx, cy: json.cy } : null;
    default:
      return null;
  }
}

/** A chunk: its ground as indices into its kinds, its heights and regions, 1,024 each, and the objects on it. */
function decodeChunk(json: Json): ServerMessage | null {
  const { version, kinds, ground, height, region, objects } = json;
  if (!isChunkKey(json) || !isInt(version) || version < 1) return null;
  if (!Array.isArray(kinds) || kinds.length === 0 || !kinds.every((kind) => isOneOf(kind, GROUND_KINDS))) return null;
  const key = { cx: json.cx, cy: json.cy };
  if (!isCellList(ground, 0, kinds.length - 1) || !isCellList(height, -128, 127) || !isCellList(region, 0, 255)) return null;
  if (!Array.isArray(objects) || !objects.every((object) => isObjectIn(object, key))) return null;
  const chunk: Chunk = {
    key,
    version,
    ground: ground.map((index) => kinds[index] as GroundKind),
    height,
    region,
    objects: objects.map(({ id, kind, col, row }) => ({ id, kind, col, row })),
  };
  return { type: 'chunk', chunk };
}

function isChunkKey(value: Json): value is Json & ChunkKey {
  return isInt(value.cx) && isInt(value.cy) && value.cx >= 0 && value.cy >= 0;
}

/** One integer from min to max per cell of a chunk. */
function isCellList(value: unknown, min: number, max: number): value is number[] {
  return Array.isArray(value) && value.length === CELLS_PER_CHUNK && value.every((n) => isInt(n) && n >= min && n <= max);
}

function isObjectIn(value: unknown, key: ChunkKey): value is MapObject {
  if (!isRecord(value) || !isNonEmptyString(value.id) || !isOneOf(value.kind, OBJECT_KINDS) || !isCell(value)) return false;
  const origin = chunkOrigin(key);
  const { col, row } = value;
  return col >= origin.col && row >= origin.row && col < origin.col + CHUNK_SIZE && row < origin.row + CHUNK_SIZE;
}

function decodeWelcome(json: Record<string, unknown>): ServerMessage | null {
  const { you, players, stepMs } = json;
  if (!isNonEmptyString(you) || !Array.isArray(players) || !players.every(isPlayerInfo)) return null;
  if (!isInt(stepMs) || stepMs <= 0) return null;
  return { type: 'welcome', you, players, stepMs };
}

function isPlayerInfo(value: unknown): value is PlayerInfo {
  return isRecord(value) && isNonEmptyString(value.id) && typeof value.name === 'string' && isCell(value);
}

function isPath(value: unknown): value is Cell[] {
  return Array.isArray(value) && value.every((cell) => isRecord(cell) && isCell(cell));
}

function isCell(value: Record<string, unknown>): value is { col: number; row: number } {
  return isInt(value.col) && isInt(value.row);
}

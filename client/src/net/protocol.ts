import type { Chunk } from '../map/chunks';
import { CELLS_PER_CHUNK, CHUNK_SIZE, chunkOrigin, type ChunkKey } from '../map/grid';
import { footprint, GROUND_KINDS, MAX_HEIGHT, MIN_HEIGHT, OBJECT_KINDS, type GroundKind, type MapDescription, type MapObject } from '../map/worldMap';
import { parseDescription } from './mapApi';
import { isInt, isNonEmptyString, isOneOf, isRecord, type Json } from './validate';

export interface Cell {
  col: number;
  row: number;
}

/** A player on the map, as the server tells it. */
export interface PlayerInfo extends Cell {
  id: string;
  name: string;
  /** The slug of the robot they're drawn as; "" when the server has none (ADR 0006). */
  robot: string;
}

/** What the server sends over the websocket (ADR 0001, ADR 0002, ADR 0003). */
export type ServerMessage =
  | { type: 'chunk'; chunk: Chunk }
  | { type: 'forget'; cx: number; cy: number }
  /** You're on a map: its description (ADR 0012), who you are, who you see. */
  | { type: 'welcome'; you: string; players: PlayerInfo[]; stepMs: number; map: MapDescription }
  | { type: 'joined'; player: PlayerInfo }
  | { type: 'moved'; id: string; col: number; row: number }
  | { type: 'left'; id: string }
  /** The path to a cell, and what going there does (ADR 0012). */
  | { type: 'planned'; col: number; row: number; path: Cell[]; action: CellAction }
  | { type: 'walking'; path: Cell[] }
  | { type: 'stopped'; col: number; row: number }
  /** You were teleported there, walkable or not (ADR 0010). */
  | { type: 'teleported'; col: number; row: number; walkable: boolean }
  /** A request wasn't done, and why, in words to show (ADR 0010). */
  | { type: 'refused'; request: string; reason: string };

/** What a click on a cell does: move there, enter the dome there, or leave the city (ADR 0012). */
export type CellAction = 'move' | 'enter' | 'exit';
export const CELL_ACTIONS: readonly CellAction[] = ['move', 'enter', 'exit'];

export interface Guest {
  id: string;
  name: string;
  robot: string;
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

/** Asks the server to put your player on a cell of a map (ADR 0010). */
export function encodeTeleport(map: string, to: Cell): string {
  return JSON.stringify({ type: 'teleport', map, col: to.col, row: to.row });
}

/** Asks the server to walk your player beside the dome on a cell, then into its city (ADR 0012). */
export function encodeEnter(dome: Cell): string {
  return JSON.stringify({ type: 'enter', col: dome.col, row: dome.row });
}

/** Asks the server to walk your player to their city's edge closest to a clicked cell, then out (ADR 0012). */
export function encodeExit(clicked: Cell): string {
  return JSON.stringify({ type: 'exit', col: clicked.col, row: clicked.row });
}

/** The game's websocket, on the page's own host (Vite proxies it to the back in dev): the server puts you on your map. */
export function socketUrl(page: { protocol: string; host: string }, guest: Guest): string {
  const scheme = page.protocol === 'https:' ? 'wss' : 'ws';
  const query = new URLSearchParams({ player: guest.id, name: guest.name, robot: guest.robot });
  return `${scheme}://${page.host}/api/ws?${query}`;
}

function decodeByType(json: Record<string, unknown>): ServerMessage | null {
  switch (json.type) {
    case 'welcome':
      return decodeWelcome(json);
    case 'joined':
      return isPlayerInfo(json.player) ? { type: 'joined', player: toPlayerInfo(json.player) } : null;
    case 'moved': {
      const { id } = json;
      return isNonEmptyString(id) && isCell(json) ? { type: 'moved', id, col: json.col, row: json.row } : null;
    }
    case 'left':
      return isNonEmptyString(json.id) ? { type: 'left', id: json.id } : null;
    case 'planned': {
      const { path, action } = json;
      return isCell(json) && isPath(path) && isOneOf(action, CELL_ACTIONS) ? { type: 'planned', col: json.col, row: json.row, path, action } : null;
    }
    case 'walking':
      return isPath(json.path) ? { type: 'walking', path: json.path } : null;
    case 'stopped':
      return isCell(json) ? { type: 'stopped', col: json.col, row: json.row } : null;
    case 'teleported': {
      const { walkable } = json;
      return isCell(json) && typeof walkable === 'boolean' ? { type: 'teleported', col: json.col, row: json.row, walkable } : null;
    }
    case 'refused': {
      const { request, reason } = json;
      return typeof request === 'string' && typeof reason === 'string' ? { type: 'refused', request, reason } : null;
    }
    case 'chunk':
      return decodeChunk(json);
    case 'forget':
      return isChunkKey(json) ? { type: 'forget', cx: json.cx, cy: json.cy } : null;
    default:
      return null;
  }
}

/** A chunk: its ground as indices into its kinds, its heights (−3 to 6) and regions, 1,024 each, and the objects on it. */
function decodeChunk(json: Json): ServerMessage | null {
  const { version, kinds, ground, height, region, objects } = json;
  if (!isChunkKey(json) || !isInt(version) || version < 1) return null;
  if (!Array.isArray(kinds) || kinds.length === 0 || !kinds.every((kind) => isOneOf(kind, GROUND_KINDS))) return null;
  const key = { cx: json.cx, cy: json.cy };
  if (!isCellList(ground, 0, kinds.length - 1) || !isCellList(height, MIN_HEIGHT, MAX_HEIGHT) || !isCellList(region, 0, 255)) return null;
  if (!Array.isArray(objects) || !objects.every((object) => isObjectIn(object, key))) return null;
  const chunk: Chunk = {
    key,
    version,
    ground: ground.map((index) => kinds[index] as GroundKind),
    height,
    region,
    objects: objects.map(toMapObject),
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

/** An object whose whole footprint is in its chunk. A dome leads to a map (ADR 0012). */
function isObjectIn(value: unknown, key: ChunkKey): value is MapObject {
  if (!isRecord(value)) return false;
  const { id, kind, leadsTo } = value;
  if (!isNonEmptyString(id) || !isOneOf(kind, OBJECT_KINDS) || !isCell(value)) return false;
  if (kind === 'dome' && !isNonEmptyString(leadsTo)) return false;
  const origin = chunkOrigin(key);
  const inChunk = ({ col, row }: Cell) => col >= origin.col && row >= origin.row && col < origin.col + CHUNK_SIZE && row < origin.row + CHUNK_SIZE;
  return footprint({ kind, col: value.col, row: value.row }).every(inChunk);
}

function toMapObject({ id, kind, col, row, leadsTo }: MapObject): MapObject {
  return kind === 'dome' ? { id, kind, col, row, leadsTo } : { id, kind, col, row };
}

function decodeWelcome(json: Record<string, unknown>): ServerMessage | null {
  const { you, players, stepMs } = json;
  if (!isNonEmptyString(you) || !Array.isArray(players) || !players.every(isPlayerInfo)) return null;
  if (!isInt(stepMs) || stepMs <= 0) return null;
  const map = descriptionOf(json.map);
  return map && { type: 'welcome', you, players: players.map(toPlayerInfo), stepMs, map };
}

/** A welcome's map, null when it isn't a valid description. */
function descriptionOf(json: unknown): MapDescription | null {
  try {
    return parseDescription(json);
  } catch {
    return null;
  }
}

/** A player's id, name and cell are required; its robot is not. */
function isPlayerInfo(value: unknown): value is Json & Omit<PlayerInfo, 'robot'> {
  return isRecord(value) && isNonEmptyString(value.id) && typeof value.name === 'string' && isCell(value);
}

/** A missing or malformed robot is "": the default robot is drawn. */
function toPlayerInfo(player: Json & Omit<PlayerInfo, 'robot'>): PlayerInfo {
  const { id, name, col, row, robot } = player;
  return { id, name, col, row, robot: typeof robot === 'string' ? robot : '' };
}

function isPath(value: unknown): value is Cell[] {
  return Array.isArray(value) && value.every((cell) => isRecord(cell) && isCell(cell));
}

function isCell(value: Record<string, unknown>): value is { col: number; row: number } {
  return isInt(value.col) && isInt(value.row);
}

import { silent, type Log } from '../debug/log';
import type { KeyValueStore } from '../core/store';
import type { Guest } from './protocol';

export interface GuestDeps {
  /** null when the browser gives no storage (private mode, blocked site data). */
  store: KeyValueStore | null;
  newId: () => string;
  /** Integer in [0, max). */
  randomInt: (max: number) => number;
  /** The robots a guest may get, from the sheet: never empty. */
  robots: readonly string[];
  log?: Log;
}

type Identity = Omit<Guest, 'robot'>;

const STORAGE_KEY = 'donut.guest';
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/**
 * The player's guest identity: read back from storage, or created once and saved. A saved guest without a robot,
 * or whose robot left the sheet, keeps their id and name and gets a new random robot (ADR 0006). Never logs the
 * secret id.
 */
export function loadGuest({ store, newId, randomInt, robots, log = silent }: GuestDeps): Guest {
  if (!store) log('no storage: the guest lasts for this session only');
  const saved = readGuest(store);
  if (saved && robots.includes(saved.robot)) {
    log('guest read back from storage', { name: saved.name, robot: saved.robot });
    return { id: saved.id, name: saved.name, robot: saved.robot };
  }
  const robot = pickRobot(robots, randomInt);
  if (saved) {
    const guest = { id: saved.id, name: saved.name, robot };
    const kept = writeGuest(store, guest);
    log('saved guest given a robot', { name: guest.name, robot, why: saved.robot ? `${saved.robot} is not in the sheet` : 'none yet', kept_in_storage: kept });
    return guest;
  }
  const guest = { id: newId(), name: `Guest-${1000 + randomInt(9000)}`, robot };
  const kept = writeGuest(store, guest);
  log('new guest created', { name: guest.name, robot, kept_in_storage: kept });
  return guest;
}

/** One of the robots, each equally likely. */
function pickRobot(robots: readonly string[], randomInt: (max: number) => number): string {
  return robots[randomInt(robots.length)] ?? '';
}

/** The saved guest with its robot as saved: "" when there is none. */
function readGuest(store: KeyValueStore | null): (Identity & { robot: string }) | null {
  try {
    const json: unknown = JSON.parse(store?.getItem(STORAGE_KEY) ?? 'null');
    if (!isIdentity(json)) return null;
    const { robot } = json as { robot?: unknown };
    return { id: json.id, name: json.name, robot: typeof robot === 'string' ? robot : '' };
  } catch {
    return null;
  }
}

/** Saves the guest; false when there's no storage, or it's full or blocked: the identity lasts for this session. */
function writeGuest(store: KeyValueStore | null, guest: Guest): boolean {
  try {
    store?.setItem(STORAGE_KEY, JSON.stringify(guest));
    return store !== null;
  } catch {
    return false;
  }
}

function isIdentity(value: unknown): value is Identity {
  if (typeof value !== 'object' || value === null) return false;
  const { id, name } = value as Record<string, unknown>;
  return typeof id === 'string' && UUID.test(id) && typeof name === 'string' && name.trim().length > 0 && name.length <= 24;
}

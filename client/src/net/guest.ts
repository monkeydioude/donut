import { silent, type Log } from '../debug/log';
import type { Guest } from './protocol';

/** The part of localStorage the guest identity needs. */
export interface KeyValueStore {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
}

export interface GuestDeps {
  /** null when the browser gives no storage (private mode, blocked site data). */
  store: KeyValueStore | null;
  newId: () => string;
  /** Integer in [0, max). */
  randomInt: (max: number) => number;
  log?: Log;
}

const STORAGE_KEY = 'donut.guest';
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/** The player's guest identity: read back from storage, or created once and saved. Never logs the secret id. */
export function loadGuest({ store, newId, randomInt, log = silent }: GuestDeps): Guest {
  if (!store) log('no storage: the guest lasts for this session only');
  const saved = readGuest(store);
  if (saved) {
    log('guest read back from storage', { name: saved.name });
    return saved;
  }
  const guest = { id: newId(), name: `Guest-${1000 + randomInt(9000)}` };
  const kept = writeGuest(store, guest);
  log('new guest created', { name: guest.name, kept_in_storage: kept });
  return guest;
}

/** localStorage, or null when touching it throws. */
export function browserStore(): KeyValueStore | null {
  try {
    return window.localStorage;
  } catch {
    return null;
  }
}

function readGuest(store: KeyValueStore | null): Guest | null {
  try {
    const json: unknown = JSON.parse(store?.getItem(STORAGE_KEY) ?? 'null');
    return isGuest(json) ? json : null;
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

function isGuest(value: unknown): value is Guest {
  if (typeof value !== 'object' || value === null) return false;
  const { id, name } = value as Record<string, unknown>;
  return typeof id === 'string' && UUID.test(id) && typeof name === 'string' && name.trim().length > 0 && name.length <= 24;
}

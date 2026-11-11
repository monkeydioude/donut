import type { KeyValueStore } from '../core/store';

const STORAGE_KEY = 'donut.camera';

/** Whether the camera starts locked: the saved lock, or locked the first time and when it can't be read. */
export function loadCameraLock(store: KeyValueStore | null): boolean {
  try {
    const saved: unknown = JSON.parse(store?.getItem(STORAGE_KEY) ?? 'null');
    return isSettings(saved) ? saved.locked : true;
  } catch {
    return true;
  }
}

/** Saves the lock, for every map. False when there's no storage, or it's full or blocked. */
export function saveCameraLock(store: KeyValueStore | null, locked: boolean): boolean {
  try {
    store?.setItem(STORAGE_KEY, JSON.stringify({ locked }));
    return store !== null;
  } catch {
    return false;
  }
}

function isSettings(value: unknown): value is { locked: boolean } {
  return typeof value === 'object' && value !== null && typeof (value as Record<string, unknown>).locked === 'boolean';
}

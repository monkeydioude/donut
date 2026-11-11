import { describe, expect, it } from 'vitest';
import { brokenStore, memoryStore } from '../testing/store';
import { loadCameraLock, saveCameraLock } from './cameraSettings';

// ADR 0004 test 23
describe('loadCameraLock', () => {
  it('is locked when nothing is saved', () => {
    expect(loadCameraLock(memoryStore())).toBe(true);
  });

  it('is locked when the saved value is broken', () => {
    for (const value of ['not json', 'null', '{"locked":"no"}', '[false]', 'false']) {
      const store = memoryStore();
      store.items.set('donut.camera', value);
      expect(loadCameraLock(store)).toBe(true);
    }
  });

  it('is locked without storage, or with a storage that throws', () => {
    expect(loadCameraLock(null)).toBe(true);
    expect(loadCameraLock(brokenStore)).toBe(true);
  });
});

// ADR 0004 test 24
describe('saveCameraLock', () => {
  it('a saved free camera is read back free', () => {
    const store = memoryStore();
    expect(saveCameraLock(store, false)).toBe(true);
    expect(store.items.get('donut.camera')).toBe('{"locked":false}');
    expect(loadCameraLock(store)).toBe(false);
  });

  it("tells when it couldn't save", () => {
    expect(saveCameraLock(null, false)).toBe(false);
    expect(saveCameraLock(brokenStore, false)).toBe(false);
  });
});

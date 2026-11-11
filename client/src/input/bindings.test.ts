import { describe, expect, it } from 'vitest';
import { createDefaultConfig } from '../config';
import { keyOutcome, platformOf, pointerAction, type KeyPress, type Platform } from './bindings';

const worldmap = createDefaultConfig().controls.worldmap;

function key(code: string, printed: string, more: Partial<KeyPress> = {}): KeyPress {
  return { code, key: printed, meta: false, ctrl: false, repeat: false, ...more };
}

const actionOf = (press: KeyPress, platform: Platform = 'mac') => keyOutcome(worldmap, press, platform).action;
const cmd = { meta: true };
const ctrl = { ctrl: true };

// ADR 0004 test 1
describe('keys by position', () => {
  it('KeyZ zooms in whatever it prints, and the key printed z elsewhere does nothing', () => {
    expect(actionOf(key('KeyZ', 'w'))).toBe('zoomIn');
    expect(actionOf(key('KeyW', 'z'))).toBeNull();
  });

  it('Y, X and Space are read by position too', () => {
    expect(actionOf(key('KeyY', 'y'))).toBe('toggleLock');
    expect(actionOf(key('KeyX', 'x'))).toBe('zoomOut');
    expect(actionOf(key('Space', ' '))).toBe('center');
  });
});

// ADR 0004 test 2
describe('keys by printed character', () => {
  it('mod + "+", mod + "=" and the numpad\'s + zoom in', () => {
    expect(actionOf(key('Equal', '+', cmd))).toBe('zoomIn');
    expect(actionOf(key('Equal', '=', cmd))).toBe('zoomIn');
    expect(actionOf(key('NumpadAdd', '+', cmd))).toBe('zoomIn');
  });

  it('mod + "-", wherever it is, zooms out', () => {
    expect(actionOf(key('Minus', '-', cmd))).toBe('zoomOut');
    expect(actionOf(key('Digit6', '-', cmd))).toBe('zoomOut');
    expect(actionOf(key('NumpadSubtract', '-', cmd))).toBe('zoomOut');
  });
});

// ADR 0004 test 3
describe('the modifier', () => {
  it('is Cmd on macOS', () => {
    expect(actionOf(key('Equal', '+', cmd), 'mac')).toBe('zoomIn');
    expect(actionOf(key('Equal', '+', ctrl), 'mac')).toBeNull();
  });

  it('is Ctrl elsewhere', () => {
    expect(actionOf(key('Equal', '+', ctrl), 'other')).toBe('zoomIn');
    expect(actionOf(key('Equal', '+', cmd), 'other')).toBeNull();
  });

  it('is told from the browser platform', () => {
    expect(['MacIntel', 'macOS', 'iPhone'].map(platformOf)).toEqual(['mac', 'mac', 'mac']);
    expect(['Win32', 'Windows', 'Linux x86_64'].map(platformOf)).toEqual(['other', 'other', 'other']);
  });
});

// ADR 0004 test 4
describe('keys without the modifier', () => {
  it("don't match while Cmd or Ctrl is held, and leave the page's shortcut alone", () => {
    expect(keyOutcome(worldmap, key('KeyZ', 'z', cmd), 'mac')).toEqual({ block: false, action: null });
    expect(keyOutcome(worldmap, key('KeyZ', 'z', ctrl), 'mac')).toEqual({ block: false, action: null });
    expect(keyOutcome(worldmap, key('KeyZ', 'z', ctrl), 'other')).toEqual({ block: false, action: null });
  });

  it('ignore Shift', () => {
    expect(actionOf(key('KeyY', 'Y'))).toBe('toggleLock');
  });
});

// ADR 0004 test 5
describe('unbound keys', () => {
  it('do nothing and leave the page alone', () => {
    expect(keyOutcome(worldmap, key('KeyQ', 'q'), 'mac')).toEqual({ block: false, action: null });
  });

  it("a key bound only in another state does nothing in this one", () => {
    const otherState = { open: [{ code: 'KeyQ' }] };
    expect(keyOutcome(otherState, key('KeyY', 'y'), 'mac')).toEqual({ block: false, action: null });
    expect(keyOutcome(otherState, key('KeyQ', 'q'), 'mac').action).toBe('open');
  });
});

// ADR 0004 test 6
describe('held keys', () => {
  it('act once: repeats are blocked but do nothing', () => {
    expect(keyOutcome(worldmap, key('KeyZ', 'z'), 'mac')).toEqual({ block: true, action: 'zoomIn' });
    expect(keyOutcome(worldmap, key('KeyZ', 'z', { repeat: true }), 'mac')).toEqual({ block: true, action: null });
  });
});

// ADR 0004 tests 7 and 8
describe('pointer inputs', () => {
  it('the wheel up zooms in and down zooms out', () => {
    expect(pointerAction(worldmap, { wheel: 'up' })).toBe('zoomIn');
    expect(pointerAction(worldmap, { wheel: 'down' })).toBe('zoomOut');
  });

  it('the right button cancels on a click; the left button pans on a drag (ADR 0007)', () => {
    expect(pointerAction(worldmap, { button: 'right', gesture: 'click' })).toBe('cancel');
    expect(pointerAction(worldmap, { button: 'left', gesture: 'drag' })).toBe('pan');
    expect(pointerAction(worldmap, { button: 'right', gesture: 'drag' }), 'a right drag does nothing').toBeNull();
    expect(pointerAction(worldmap, { button: 'left', gesture: 'click' }), 'a left click selects a tile, on the pointer').toBeNull();
  });
});

const mapState = createDefaultConfig().controls.map;

// ADR 0005 test 9
describe('opening the map', () => {
  it('m and M open it on the world map, whatever key position prints them', () => {
    expect(actionOf(key('KeyM', 'm'))).toBe('openMap');
    expect(actionOf(key('Semicolon', 'm'))).toBe('openMap');
    expect(actionOf(key('Semicolon', 'M'))).toBe('openMap');
    expect(actionOf(key('KeyM', ','))).toBeNull();
  });
});

// ADR 0005 test 10
describe('the map state', () => {
  const onMap = (press: KeyPress) => keyOutcome(mapState, press, 'mac').action;

  it('m, M and Escape close the map', () => {
    expect(onMap(key('Semicolon', 'm'))).toBe('close');
    expect(onMap(key('KeyM', 'M'))).toBe('close');
    expect(onMap(key('Escape', 'Escape'))).toBe('close');
  });

  it("the world map's keys do nothing on it", () => {
    for (const press of [key('KeyZ', 'z'), key('KeyY', 'y'), key('Space', ' ')]) {
      expect(keyOutcome(mapState, press, 'mac')).toEqual({ block: false, action: null });
    }
    expect(pointerAction(mapState, { wheel: 'up' })).toBeNull();
    expect(pointerAction(mapState, { button: 'right', gesture: 'drag' })).toBeNull();
  });
});

import { describe, expect, it } from 'vitest';
import type { ScreenPoint, TilePoint } from '../iso/projection';
import { createClickGuard, createHoverTracker } from './pointer';

/** Tiles 10 px wide along x, under a camera that can slide by `shift` px. */
function setup() {
  const camera = { shift: 0 };
  const hovers: (TilePoint | null)[] = [];
  const pick = (point: ScreenPoint) => ({ col: Math.floor((point.x + camera.shift) / 10), row: 0 });
  const tracker = createHoverTracker(pick, (cell) => hovers.push(cell));
  return { camera, hovers, tracker };
}

// ADR 0004 test 25
describe('createHoverTracker', () => {
  it('gives the new tile under a still pointer when the camera moves', () => {
    const { camera, hovers, tracker } = setup();
    tracker.moveTo({ x: 5, y: 0 });
    camera.shift = 10;
    tracker.repick();
    expect(hovers).toEqual([{ col: 0, row: 0 }, { col: 1, row: 0 }]);
  });

  it('gives nothing when it is the same tile', () => {
    const { camera, hovers, tracker } = setup();
    tracker.moveTo({ x: 5, y: 0 });
    camera.shift = 3;
    tracker.repick();
    expect(hovers).toEqual([{ col: 0, row: 0 }]);
  });

  it('picks nothing once the pointer left', () => {
    const { camera, hovers, tracker } = setup();
    tracker.moveTo({ x: 5, y: 0 });
    tracker.leave();
    camera.shift = 10;
    tracker.repick();
    expect(hovers).toEqual([{ col: 0, row: 0 }, null]);
  });
});

// ADR 0007 test 38
describe('createClickGuard', () => {
  it('a press and a release within 4 px is a click', () => {
    const guard = createClickGuard(4);
    guard.down({ x: 10, y: 10 });
    guard.move({ x: 12, y: 11 });
    expect(guard.isClick({ x: 12, y: 11 })).toBe(true);
  });

  it('the release of a drag is no click, even back where it started', () => {
    const guard = createClickGuard(4);
    guard.down({ x: 10, y: 10 });
    guard.move({ x: 40, y: 10 });
    guard.move({ x: 10, y: 10 });
    expect(guard.isClick({ x: 10, y: 10 })).toBe(false);
    expect(guard.isClick({ x: 10, y: 10 }), 'the next click, without a press seen, is a click').toBe(true);
  });
});

import { describe, expect, it } from 'vitest';
import { isClick, moveTo, press } from './drag';

const threshold = 4;

// ADR 0004 test 14
describe('a right click', () => {
  it('is a press and a release within 4 px', () => {
    const held = press({ x: 10, y: 10 });
    const moved = moveTo(held, { x: 12, y: 12 }, threshold);
    expect(moved.delta).toBeNull();
    expect(isClick(moved.held, { x: 13, y: 10 }, threshold)).toBe(true);
  });

  it('is not a click when released 4 px away or more', () => {
    expect(isClick(press({ x: 0, y: 0 }), { x: 4, y: 0 }, threshold)).toBe(false);
  });
});

// ADR 0004 test 15
describe('a right drag', () => {
  it('starts past 4 px, its first move carrying everything since the press', () => {
    const first = moveTo(press({ x: 10, y: 10 }), { x: 15, y: 10 }, threshold);
    expect(first.delta).toEqual({ x: 5, y: 0 });
    expect(first.held.dragging).toBe(true);
  });

  it('then gives each move since the last one', () => {
    const first = moveTo(press({ x: 10, y: 10 }), { x: 15, y: 10 }, threshold);
    const second = moveTo(first.held, { x: 16, y: 12 }, threshold);
    expect(second.delta).toEqual({ x: 1, y: 2 });
    const third = moveTo(second.held, { x: 15, y: 11 }, threshold);
    expect(third.delta).toEqual({ x: -1, y: -1 });
  });

  it('gives no click on release, even back where it started', () => {
    const dragged = moveTo(press({ x: 10, y: 10 }), { x: 20, y: 10 }, threshold).held;
    const back = moveTo(dragged, { x: 10, y: 10 }, threshold).held;
    expect(isClick(back, { x: 10, y: 10 }, threshold)).toBe(false);
  });
});

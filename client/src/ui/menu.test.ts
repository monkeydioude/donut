import { describe, expect, it } from 'vitest';
import { menuPosition } from './menu';

const viewport = { width: 800, height: 600 };
const menu = { width: 120, height: 80 };

// ADR 0002 test 20
describe('menuPosition', () => {
  it('opens to the right of the tile, vertically centered on it', () => {
    expect(menuPosition({ left: 300, right: 396, y: 300 }, menu, viewport)).toEqual({ left: 404, top: 260 });
  });

  it('opens to the left of the tile when the right side is too narrow', () => {
    expect(menuPosition({ left: 650, right: 746, y: 300 }, menu, viewport)).toEqual({ left: 522, top: 260 });
  });

  it('stays inside the viewport near the top and bottom', () => {
    expect(menuPosition({ left: 300, right: 396, y: 10 }, menu, viewport).top).toBe(8);
    expect(menuPosition({ left: 300, right: 396, y: 595 }, menu, viewport).top).toBe(512);
  });

  it('stays inside the viewport when neither side fits', () => {
    expect(menuPosition({ left: 20, right: 760, y: 300 }, menu, viewport).left).toBe(8);
  });
});

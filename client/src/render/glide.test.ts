import { describe, expect, it } from 'vitest';
import { createIsoProjection } from '../iso/projection';
import { drawnTarget, glideAt, nextGlide, placePlayer, shiftGlide } from './glide';

const a = { x: 0, y: 0 };
const b = { x: 32, y: 16 };
const c = { x: 64, y: 0 };

// ADR 0002 test 24
describe('smooth steps', () => {
  it('a glide goes from the shown position to the new cell in its time, then stays', () => {
    const glide = nextGlide(nextGlide(undefined, a, 'place', 0, 200), b, 'glide', 1000, 200);
    expect(glideAt(glide, 1000)).toEqual(a);
    expect(glideAt(glide, 1100)).toEqual({ x: 16, y: 8 });
    expect(glideAt(glide, 1200)).toEqual(b);
    expect(glideAt(glide, 5000)).toEqual(b);
  });

  it('a step coming mid-glide starts from where the sprite is', () => {
    const first = nextGlide(nextGlide(undefined, a, 'place', 0, 200), b, 'glide', 1000, 200);
    const second = nextGlide(first, c, 'glide', 1100, 200);
    expect(second.from).toEqual({ x: 16, y: 8 });
    expect(glideAt(second, 1300)).toEqual(c);
  });

  it('joining, a welcome or stopped place the sprite at once', () => {
    expect(glideAt(nextGlide(undefined, b, 'glide', 1000, 200), 1000)).toEqual(b);
    const moving = nextGlide(nextGlide(undefined, a, 'place', 0, 200), b, 'glide', 1000, 200);
    expect(glideAt(nextGlide(moving, c, 'place', 1100, 200), 1100)).toEqual(c);
  });

  it('the draw order follows the target cell', () => {
    const projection = createIsoProjection(64, 32);
    const from = placePlayer(undefined, { col: 1, row: 1 }, 'place', 0, 200, projection);
    const to = placePlayer(from.glide, { col: 2, row: 2 }, 'glide', 1000, 200, projection);
    expect(to.zIndex).toBe(4);
    expect(glideAt(to.glide, 1000)).toEqual(projection.toScreen({ col: 1, row: 1 }));
  });
});

// ADR 0003 test 28
describe('drawn cells across the seams', () => {
  const world = { width: 1024, height: 512, wraps: true };

  it('your step across the seam glides one cell: 1023 to 1024', () => {
    expect(drawnTarget(world, { col: 0, row: 5 }, { col: 1023, row: 5 }, 'glide', { col: 1023, row: 5 })).toEqual({ col: 1024, row: 5 });
    expect(drawnTarget(world, { col: 1, row: 5 }, { col: 1024, row: 5 }, 'glide', { col: 1024, row: 5 }), 'and keeps counting').toEqual({
      col: 1025,
      row: 5,
    });
  });

  it("another player's step glides one cell from its drawn copy", () => {
    expect(drawnTarget(world, { col: 1023, row: 5 }, { col: 1024, row: 5 }, 'glide', { col: 1030, row: 5 })).toEqual({ col: 1023, row: 5 });
  });

  it('a player placed at once is drawn at its copy nearest to your drawn cell', () => {
    expect(drawnTarget(world, { col: 1020, row: 5 }, undefined, 'place', { col: 3, row: 5 })).toEqual({ col: -4, row: 5 });
    expect(drawnTarget(world, { col: 10, row: 5 }, undefined, 'place', null), 'you, at your welcome').toEqual({ col: 10, row: 5 });
  });

  it('a glide moves whole to another copy', () => {
    const glide = { from: a, to: b, startedAt: 0, durationMs: 200 };
    expect(shiftGlide(glide, { x: 100, y: -50 })).toEqual({ from: { x: 100, y: -50 }, to: { x: 132, y: -34 }, startedAt: 0, durationMs: 200 });
  });
});

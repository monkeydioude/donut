import { describe, expect, it } from 'vitest';
import { createIsoProjection, type ScreenPoint, type TilePoint } from '../iso/projection';
import {
  backOnPlayer,
  cameraCenter,
  centerOnPlayer,
  createCamera,
  followPlayer,
  initialCamera,
  pan,
  toggleLock,
  worldPlacement,
  zoomStep,
  type CameraState,
} from './camera';

const projection = createIsoProjection(32, 16);
const rules = { projection, zoom: { min: 2, max: 6 }, leashCells: 32 };
const viewport = { width: 800, height: 600 };
const at = (col: number, row: number) => projection.toScreen({ col, row });
const player = at(10, 10);
const locked = initialCamera(true, 3);
const freeAt = (center: ScreenPoint, zoom = 3): CameraState => ({ locked: false, zoom, center });

/** How far the camera's center is from your player, in fractional cells. */
function offsetInCells(camera: CameraState, from: ScreenPoint): TilePoint {
  const center = cameraCenter(camera, from);
  return projection.toTileFraction({ x: center.x - from.x, y: center.y - from.y });
}

/** Where a world point lands on the screen. */
function onScreen(camera: CameraState, from: ScreenPoint, point: ScreenPoint): ScreenPoint {
  const { origin, zoom } = worldPlacement(camera, from, viewport);
  return { x: origin.x + point.x * zoom, y: origin.y + point.y * zoom };
}

const middle = { x: viewport.width / 2, y: viewport.height / 2 };

// ADR 0004 test 16
describe('a locked camera', () => {
  it('is on your player, and stays on them as they walk', () => {
    expect(onScreen(locked, player, player)).toEqual(middle);
    const walked = at(14, 9);
    expect(followPlayer(locked, walked, rules)).toBe(locked);
    expect(onScreen(locked, walked, walked)).toEqual(middle);
  });
});

// ADR 0004 test 17
describe('toggleLock', () => {
  it('freeing keeps the camera where it is, and it stays there as your player walks', () => {
    const free = toggleLock(locked, player);
    expect(free.locked).toBe(false);
    const walked = at(15, 10);
    expect(cameraCenter(followPlayer(free, walked, rules), walked)).toEqual(player);
  });

  it('locking puts it back on your player', () => {
    const relocked = toggleLock(freeAt(at(20, 10)), player);
    expect(relocked.locked).toBe(true);
    expect(cameraCenter(relocked, player)).toEqual(player);
  });
});

// ADR 0004 test 18
describe('centerOnPlayer', () => {
  it('puts a free camera back on your player, and it stays free', () => {
    const back = centerOnPlayer(freeAt(at(20, 10)), player);
    expect(back.locked).toBe(false);
    expect(cameraCenter(back, player)).toEqual(player);
  });

  it('changes nothing when locked', () => {
    expect(centerOnPlayer(locked, player)).toBe(locked);
  });
});

// ADR 0012: entering or leaving a city
describe('backOnPlayer', () => {
  it('puts a free camera back on your player, wherever they are, and it follows them until moved', () => {
    const back = backOnPlayer(freeAt(at(1021, 511), 4));
    expect(back).toEqual({ locked: false, zoom: 4, center: null });
    expect(cameraCenter(back, at(15, 15))).toEqual(at(15, 15));
    expect(cameraCenter(followPlayer(back, at(16, 15), rules), at(16, 15))).toEqual(at(16, 15));
  });

  it('changes nothing when locked, or already on your player', () => {
    expect(backOnPlayer(locked)).toBe(locked);
    const following = initialCamera(false, 3);
    expect(backOnPlayer(following)).toBe(following);
  });
});

// ADR 0004 test 19
describe('pan', () => {
  it('dragging 30 px right at zoom 3 moves the camera 10 art pixels left', () => {
    const moved = pan(freeAt(player), { x: 30, y: 0 }, player, rules);
    expect(cameraCenter(moved, player)).toEqual({ x: player.x - 10, y: player.y });
  });

  it('changes nothing when locked', () => {
    expect(pan(locked, { x: 30, y: 0 }, player, rules)).toBe(locked);
  });
});

// ADR 0004 test 20
describe('the leash', () => {
  /** The screen drag that would take the camera from your player to that offset, at zoom 3. */
  const dragTo = (offset: TilePoint) => {
    const world = projection.toScreen(offset);
    return { x: -world.x * 3, y: -world.y * 3 };
  };

  it('stops a pan at 32 cells along the columns', () => {
    const moved = pan(freeAt(player), dragTo({ col: 50, row: 0 }), player, rules);
    expect(offsetInCells(moved, player)).toEqual({ col: 32, row: 0 });
  });

  it('stops a pan at 32 cells along the rows', () => {
    const moved = pan(freeAt(player), dragTo({ col: 3, row: -50 }), player, rules);
    expect(offsetInCells(moved, player)).toEqual({ col: 3, row: -32 });
  });

  it('pulls the camera to 32 cells when your player walks 40 away', () => {
    const walked = at(50, 10);
    expect(offsetInCells(followPlayer(freeAt(player), walked, rules), walked)).toEqual({ col: -32, row: 0 });
  });

  it('counts in drawn cells past the seam', () => {
    const pastSeam = at(1030, 5);
    const within = freeAt(at(1000, 5));
    expect(followPlayer(within, pastSeam, rules)).toBe(within);
    expect(offsetInCells(followPlayer(freeAt(at(990, 5)), pastSeam, rules), pastSeam)).toEqual({ col: -32, row: 0 });
  });
});

// ADR 0004 test 21
describe('zoomStep', () => {
  it('steps by 1 from 2 to 6, and a step past a limit changes nothing', () => {
    expect(zoomStep(locked, 1, null, player, rules).zoom).toBe(4);
    const top = initialCamera(true, 6);
    const bottom = initialCamera(true, 2);
    expect(zoomStep(top, 1, null, player, rules)).toBe(top);
    expect(zoomStep(bottom, -1, null, player, rules)).toBe(bottom);
  });

  it('keeps your player in the middle when locked, even under a pointer', () => {
    const zoomed = zoomStep(locked, 1, { x: 300, y: 200 }, player, rules);
    expect(onScreen(zoomed, player, player)).toEqual(middle);
  });

  it('keeps the middle of the screen still with the keys, when free', () => {
    const camera = freeAt(at(20, 12));
    expect(cameraCenter(zoomStep(camera, -1, null, player, rules), player)).toEqual(at(20, 12));
  });

  it('keeps the world point under the pointer under it with the wheel, when free', () => {
    const camera = freeAt(at(14, 10));
    const pointer = { x: 200, y: -120 };
    const under = (c: CameraState) => {
      const center = cameraCenter(c, player);
      return { x: center.x + pointer.x / c.zoom, y: center.y + pointer.y / c.zoom };
    };
    const zoomed = zoomStep(camera, 1, pointer, player, rules);
    expect(zoomed.zoom).toBe(4);
    expect(under(zoomed).x).toBeCloseTo(under(camera).x);
    expect(under(zoomed).y).toBeCloseTo(under(camera).y);
  });

  it("can't push the camera past the leash", () => {
    const atLimit = freeAt({ x: player.x + at(32, 0).x, y: player.y + at(32, 0).y });
    const towardsMore = { x: 380, y: 190 };
    const zoomed = zoomStep(atLimit, 1, towardsMore, player, rules);
    expect(offsetInCells(zoomed, player).col).toBeCloseTo(32);
    expect(offsetInCells(zoomed, player).col).toBeLessThanOrEqual(32);
  });
});

// ADR 0004 test 22
describe('worldPlacement', () => {
  it("puts the camera's center in the middle of the screen, on whole pixels, at the current zoom", () => {
    const camera = freeAt({ x: 103.4, y: -57.7 }, 5);
    const odd = { width: 801, height: 601 };
    const { origin, zoom } = worldPlacement(camera, player, odd);
    expect(zoom).toBe(5);
    expect(Number.isInteger(origin.x) && Number.isInteger(origin.y)).toBe(true);
    expect(Math.abs(origin.x + 103.4 * 5 - 400.5)).toBeLessThanOrEqual(0.5);
    expect(Math.abs(origin.y - 57.7 * 5 - 300.5)).toBeLessThanOrEqual(0.5);
  });
});

// ADR 0004 test 24, the camera's side
describe('createCamera', () => {
  const setup = (startLocked: boolean, you: ScreenPoint | null = player) => {
    const saved: boolean[] = [];
    const camera = createCamera({
      rules,
      start: initialCamera(startLocked, 3),
      player: () => you,
      viewport: () => viewport,
      saveLock: (next) => saved.push(next) > 0,
    });
    return { camera, saved };
  };

  it('saves the lock each time it switches', () => {
    const { camera, saved } = setup(true);
    camera.toggleLock();
    camera.toggleLock();
    expect(saved).toEqual([false, true]);
    expect(camera.locked()).toBe(true);
  });

  it('places nothing before your player shows', () => {
    expect(setup(true, null).camera.place()).toBeNull();
  });

  it('places the world on your player at the starting zoom', () => {
    expect(setup(false).camera.place()).toEqual(worldPlacement(locked, player, viewport));
  });

  it('after a map switch, places a free camera on your player on the new map, still free', () => {
    let you: ScreenPoint | null = at(1021, 511);
    const camera = createCamera({ rules, start: initialCamera(true, 3), player: () => you, viewport: () => viewport, saveLock: () => true });
    camera.toggleLock();
    camera.pan({ x: 300, y: 0 });

    you = null;
    camera.recenter();
    expect(camera.place()).toBeNull();
    you = at(15, 15);
    expect(camera.place()).toEqual(worldPlacement(locked, at(15, 15), viewport));
    expect(camera.locked()).toBe(false);
  });
});

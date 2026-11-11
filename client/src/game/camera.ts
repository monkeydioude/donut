import { silent, type Log } from '../debug/log';
import type { IsoProjection, ScreenPoint } from '../iso/projection';
import { followPosition, type Viewport, type WorldPlacement } from '../render/camera';

export interface CameraRules {
  projection: IsoProjection;
  zoom: { min: number; max: number };
  /** How far a free camera's center may go from your player, in cells, along the columns and along the rows. */
  leashCells: number;
}

/**
 * The camera: locked on your player, or free. A free camera's `center` is the world point (drawn coordinates, art
 * pixels) in the middle of the screen; null until it moves away from your player.
 */
export interface CameraState {
  locked: boolean;
  zoom: number;
  center: ScreenPoint | null;
}

export function initialCamera(locked: boolean, zoom: number): CameraState {
  return { locked, zoom, center: null };
}

/** The world point in the middle of the screen. */
export function cameraCenter(camera: CameraState, player: ScreenPoint): ScreenPoint {
  return camera.locked ? player : (camera.center ?? player);
}

/** Freeing leaves the camera where it is; locking puts it back on your player. */
export function toggleLock(camera: CameraState, player: ScreenPoint | null): CameraState {
  return camera.locked ? { ...camera, locked: false, center: player } : { ...camera, locked: true, center: null };
}

/** A free camera jumps back onto your player, and stays free. */
export function centerOnPlayer(camera: CameraState, player: ScreenPoint): CameraState {
  return camera.locked ? camera : { ...camera, center: player };
}

/**
 * On another map, a free camera goes back onto your player, wherever they appear, and follows them until it's moved, as
 * when the game starts (ADR 0012). It stays free, at its zoom. Same state when locked or already on them.
 */
export function backOnPlayer(camera: CameraState): CameraState {
  return camera.center === null ? camera : { ...camera, center: null };
}

/** A free camera follows a drag of `delta` screen pixels: the world moves with the mouse, pixel for pixel. */
export function pan(camera: CameraState, delta: ScreenPoint, player: ScreenPoint, rules: CameraRules): CameraState {
  if (camera.locked) return camera;
  const from = cameraCenter(camera, player);
  const center = { x: from.x - delta.x / camera.zoom, y: from.y - delta.y / camera.zoom };
  return { ...camera, center: leash(center, player, rules) };
}

/**
 * One zoom step, within the rules' range. `anchor` is the point that stays still, in screen pixels from the middle
 * of the screen: null keeps the middle still. Locked, your player stays in the middle.
 */
export function zoomStep(
  camera: CameraState,
  step: 1 | -1,
  anchor: ScreenPoint | null,
  player: ScreenPoint,
  rules: CameraRules,
): CameraState {
  const zoom = clamp(camera.zoom + step, rules.zoom.min, rules.zoom.max);
  if (zoom === camera.zoom) return camera;
  if (camera.locked || !anchor) return { ...camera, zoom };
  const from = cameraCenter(camera, player);
  const under = { x: from.x + anchor.x / camera.zoom, y: from.y + anchor.y / camera.zoom };
  const center = { x: under.x - anchor.x / zoom, y: under.y - anchor.y / zoom };
  return { ...camera, zoom, center: leash(center, player, rules) };
}

/** Each frame: a free camera stays on the leash, pulled along when your player walks away. Same state when not. */
export function followPlayer(camera: CameraState, player: ScreenPoint, rules: CameraRules): CameraState {
  if (camera.locked || !camera.center) return camera;
  const center = leash(camera.center, player, rules);
  return center === camera.center ? camera : { ...camera, center };
}

/**
 * A point kept within `leashCells` of your player, along the columns and along the rows. It counts in drawn
 * coordinates, which don't wrap around your player, so it holds across the seams. The same point when within.
 */
export function leash(center: ScreenPoint, player: ScreenPoint, rules: CameraRules): ScreenPoint {
  const { projection, leashCells } = rules;
  const offset = projection.toTileFraction({ x: center.x - player.x, y: center.y - player.y });
  const kept = { col: clamp(offset.col, -leashCells, leashCells), row: clamp(offset.row, -leashCells, leashCells) };
  if (kept.col === offset.col && kept.row === offset.row) return center;
  const back = projection.toScreen(kept);
  return { x: player.x + back.x, y: player.y + back.y };
}

/** Where the world is drawn: the camera's center in the middle of the screen, on whole pixels, at its zoom. */
export function worldPlacement(camera: CameraState, player: ScreenPoint, viewport: Viewport): WorldPlacement {
  return { origin: followPosition(cameraCenter(camera, player), viewport, camera.zoom), zoom: camera.zoom };
}

export interface CameraDeps {
  rules: CameraRules;
  start: CameraState;
  /** Your player's drawn position: null before the welcome. */
  player: () => ScreenPoint | null;
  viewport: () => Viewport;
  /** Saves the lock: false when it couldn't. */
  saveLock: (locked: boolean) => boolean;
  log?: Log;
}

/** The camera as the controls drive it. */
export interface Camera {
  locked(): boolean;
  toggleLock(): void;
  center(): void;
  /** Back onto your player after a map switch, before they show on the new map: see `backOnPlayer`. */
  recenter(): void;
  pan(delta: ScreenPoint): void;
  /** One zoom step. `at`, the pointer on the canvas, stays still when the camera is free (the wheel). */
  zoom(step: 1 | -1, at?: ScreenPoint): void;
  /** Each frame: where to draw the world. Null before your player shows. */
  place(): WorldPlacement | null;
}

export function createCamera(deps: CameraDeps): Camera {
  const log = deps.log ?? silent;
  let camera = deps.start;
  let pulling = false;

  const fromMiddle = (at: ScreenPoint): ScreenPoint => {
    const viewport = deps.viewport();
    return { x: at.x - viewport.width / 2, y: at.y - viewport.height / 2 };
  };
  // before your player shows, the camera has nothing to look at
  const change = (next: (camera: CameraState, player: ScreenPoint) => CameraState) => {
    const player = deps.player();
    if (player) camera = next(camera, player);
    return player !== null;
  };

  return {
    locked: () => camera.locked,
    toggleLock() {
      camera = toggleLock(camera, deps.player());
      log(camera.locked ? 'locked on your player' : 'free', { saved: deps.saveLock(camera.locked) });
    },
    center() {
      if (!camera.locked && change(centerOnPlayer)) log('back on your player');
    },
    recenter() {
      const before = camera;
      camera = backOnPlayer(camera);
      if (camera !== before) log('back on your player, on another map');
    },
    pan(delta) {
      change((c, player) => pan(c, delta, player, deps.rules));
    },
    zoom(step, at) {
      const before = camera.zoom;
      const anchor = at ? fromMiddle(at) : null;
      if (!change((c, player) => zoomStep(c, step, anchor, player, deps.rules))) return;
      if (camera.zoom === before) log('zoom refused at its limit', { zoom: before, step });
      else log('zoom', { from: before, to: camera.zoom, stillPoint: anchor && !camera.locked ? 'pointer' : 'middle' });
    },
    place() {
      const player = deps.player();
      if (!player) return null;
      const next = followPlayer(camera, player, deps.rules);
      if ((next !== camera) !== pulling) {
        pulling = next !== camera;
        log(pulling ? 'leash: pulling the camera along' : 'leash: let go');
      }
      camera = next;
      return worldPlacement(camera, player, deps.viewport());
    },
  };
}

function clamp(value: number, low: number, high: number): number {
  return Math.min(Math.max(value, low), high);
}

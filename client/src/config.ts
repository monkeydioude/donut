import type { ControlsConfig } from './input/bindings';

export interface CameraConfig {
  /** Screen pixels per art pixel, in whole steps to keep pixels crisp and chunky. */
  zoom: { min: number; max: number; start: number };
  /** How far a free camera's center may go from your player, in cells, along the columns and along the rows. */
  leashCells: number;
  /** Wheel scroll added up, in pixels, for one zoom step, and the pause after a step that drops scrolls. */
  wheel: { stepPx: number; pauseMs: number };
  /** How far, in CSS pixels, the right button moves between press and release before it's a drag, not a click. */
  dragThresholdPx: number;
}

export interface GameConfig {
  /** Seeds the generated art: which variants and which details. */
  seed: number;
  /** The map to play on, as the back names it. */
  mapSlug: string;
  /** Size of a tile's ground diamond, in art pixels. */
  tile: { width: number; height: number };
  /** Room in art pixels above the ground for rocks. */
  rockHeadroom: number;
  /** Art pixels a cell is drawn higher per level of height (ADR 0007). */
  levelHeight: number;
  /** How long each frame of the water's flow lasts (ADR 0007). */
  waterFrameMs: number;
  variantsPerKind: number;
  camera: CameraConfig;
  /** The inputs of each game state's actions (ADR 0004, ADR 0005). */
  controls: ControlsConfig;
  background: string;
  /** Screen pixels per pixel of the finger cursor and the icons. */
  cursorScale: number;
  /** Debug logs in the browser console: on in dev builds. */
  debug: boolean;
}

export function createDefaultConfig(): GameConfig {
  return {
    seed: 42,
    mapSlug: 'world',
    tile: { width: 32, height: 16 },
    rockHeadroom: 6,
    levelHeight: 8,
    waterFrameMs: 250,
    variantsPerKind: 16,
    camera: {
      zoom: { min: 2, max: 6, start: 3 },
      leashCells: 32,
      wheel: { stepPx: 100, pauseMs: 150 },
      dragThresholdPx: 4,
    },
    controls: {
      worldmap: {
        toggleLock: [{ code: 'KeyY' }],
        center: [{ code: 'Space' }],
        zoomIn: [{ code: 'KeyZ' }, { wheel: 'up' }, { char: '+', mod: true }, { char: '=', mod: true }],
        zoomOut: [{ code: 'KeyX' }, { wheel: 'down' }, { char: '-', mod: true }],
        // the left button, so a trackpad pans without a two-finger drag (ADR 0007)
        pan: [{ button: 'left', gesture: 'drag' }],
        cancel: [{ button: 'right', gesture: 'click' }],
        // by its printed letter, M on any layout (ADR 0005)
        openMap: [{ char: 'm' }, { char: 'M' }],
      },
      map: {
        close: [{ char: 'm' }, { char: 'M' }, { code: 'Escape' }],
      },
    },
    background: '#1d2330',
    cursorScale: 2,
    debug: import.meta.env.DEV,
  };
}

import { silent, type Log } from '../debug/log';
import type { ControlStates } from '../input/bindings';
import type { ActionHandlers, Controls } from '../input/controls';

/** The map's window, as the screens drive it. */
export interface MapWindow {
  open(): void;
  close(): void;
}

/** The world map's actions, but opening the map: the screens add it. */
export type WorldmapHandlers = ActionHandlers<Exclude<ControlStates['worldmap'], 'openMap'>>;

export interface ScreensDeps {
  controls: Pick<Controls, 'enter'>;
  worldmap: WorldmapHandlers;
  mapWindow: MapWindow;
  /** Cancels the tile selection, which closes the action menu. */
  cancelSelection: () => void;
  log?: Log;
}

export interface Screens {
  openMap(): void;
  closeMap(): void;
}

/** Which screen is on, the world map or its map, and their controls (ADR 0005). Starts on the world map. */
export function startScreens(deps: ScreensDeps): Screens {
  const log = deps.log ?? silent;
  const enterWorldmap = () => deps.controls.enter('worldmap', { ...deps.worldmap, openMap: screens.openMap });
  const screens: Screens = {
    openMap() {
      log('map opened');
      deps.cancelSelection();
      deps.mapWindow.open();
      deps.controls.enter('map', { close: screens.closeMap });
    },
    closeMap() {
      log('map closed');
      deps.mapWindow.close();
      enterWorldmap();
    },
  };
  enterWorldmap();
  return screens;
}

import { describe, expect, it } from 'vitest';
import type { StateName } from '../input/bindings';
import type { ActionHandlers } from '../input/controls';
import { startScreens } from './screens';

function setup() {
  const events: string[] = [];
  const handlers: Partial<Record<StateName, ActionHandlers<string>>> = {};
  const screens = startScreens({
    controls: {
      enter: (state, given) => {
        events.push(`enter ${state}`);
        handlers[state] = given as ActionHandlers<string>;
      },
    },
    worldmap: {
      toggleLock: () => events.push('toggleLock'),
      center: () => {},
      chat: () => {},
      zoomIn: () => {},
      zoomOut: () => {},
      pan: () => {},
      cancel: () => {},
    },
    mapWindow: { open: () => events.push('window open'), close: () => events.push('window close') },
    cancelSelection: () => events.push('selection cancelled'),
  });
  return { screens, events, handlers };
}

// ADR 0005 test 11
describe('startScreens', () => {
  it('starts on the world map, its own actions kept', () => {
    const { events, handlers } = setup();
    expect(events).toEqual(['enter worldmap']);
    handlers.worldmap?.toggleLock?.({});
    expect(events.at(-1)).toBe('toggleLock');
  });

  it("the world map's openMap cancels the selection, shows the window and enters the map state", () => {
    const { events, handlers } = setup();
    handlers.worldmap?.openMap?.({});
    expect(events.slice(1)).toEqual(['selection cancelled', 'window open', 'enter map']);
  });

  it("the map's close hides the window and enters the world map again", () => {
    const { screens, events, handlers } = setup();
    screens.openMap();
    handlers.map?.close?.({});
    expect(events.slice(-2)).toEqual(['window close', 'enter worldmap']);
  });
});

import { describe, expect, it } from 'vitest';
import type { TilePoint } from '../iso/projection';
import type { Chunk } from '../map/chunks';
import type { GameSocketHandlers } from '../net/gameSocket';
import type { Cell } from '../net/protocol';
import type { PointerHandlers } from '../ui/pointer';
import type { MenuChoices } from './session';
import { startPlayersSession } from './session';

/** Chunk (0, 0) of the map, all grass, in region 1. */
function chunk(region = 1): Chunk {
  return { key: { cx: 0, cy: 0 }, version: 1, ground: Array(1024).fill('grass'), height: Array(1024).fill(0), region: Array(1024).fill(region), objects: [] };
}

function setup() {
  const sent: { type: 'plan' | 'walk'; to: Cell }[] = [];
  const menu: { shown: { cell: TilePoint; moveEnabled: boolean } | null } = { shown: null };
  const drawn: { path: readonly TilePoint[]; hover: TilePoint | null; chunks: string[]; banners: string[] } = {
    path: [],
    hover: null,
    chunks: [],
    banners: [],
  };
  let socket: GameSocketHandlers | undefined;
  let pointer: PointerHandlers | undefined;
  let choices: MenuChoices | undefined;
  let disconnected = false;
  startPlayersSession({
    map: { slug: 'world', name: 'World', width: 160, height: 160, wraps: true, chunkSize: 32, regions: [{ id: 1, name: 'Sprinkle Meadow', biome: 'meadow' }] },
    socketUrl: 'ws://test',
    openSocket: (_url, handlers) => {
      socket = handlers;
      return { plan: (to) => sent.push({ type: 'plan', to }), walk: (to) => sent.push({ type: 'walk', to }), close: () => {} };
    },
    listenToPointer: (handlers) => {
      pointer = handlers;
      return () => {};
    },
    createMenu: (c) => {
      choices = c;
      return { show: (cell, moveEnabled) => (menu.shown = { cell, moveEnabled }), hide: () => (menu.shown = null) };
    },
    players: { sync: () => {}, ownPosition: () => null, ownCell: () => ({ col: 2, row: 2 }) },
    chunks: {
      show: (c) => drawn.chunks.push(`show ${c.key.cx},${c.key.cy}`),
      forget: (key) => drawn.chunks.push(`forget ${key.cx},${key.cy}`),
      follow: () => drawn.chunks.push('follow'),
    },
    overlay: { showHover: (cell) => (drawn.hover = cell), showSelection: () => {}, showPath: (path) => (drawn.path = path) },
    banner: { show: (name) => drawn.banners.push(name) },
    onDisconnect: () => (disconnected = true),
    now: () => 0,
  });
  socket!.onMessage({ type: 'welcome', you: 'a', players: [{ id: 'a', name: 'Ann', col: 2, row: 2 }], stepMs: 200 });
  socket!.onMessage({ type: 'chunk', chunk: chunk() });
  return { sent, menu, drawn, socket: socket!, pointer: pointer!, choices: choices!, isDisconnected: () => disconnected };
}

const tile = { col: 4, row: 2 };
const path = [{ col: 3, row: 2 }, { col: 4, row: 2 }];

describe('startPlayersSession', () => {
  it('a click asks the path and opens the menu, Move on once the path comes', () => {
    const session = setup();
    session.pointer.select(tile);
    expect(session.sent).toEqual([{ type: 'plan', to: tile }]);
    expect(session.menu.shown).toEqual({ cell: tile, moveEnabled: false });

    session.socket.onMessage({ type: 'planned', ...tile, path });
    expect(session.menu.shown).toEqual({ cell: tile, moveEnabled: true });
    expect(session.drawn.path).toEqual(path);
  });

  it('Move asks the walk and closes the menu', () => {
    const session = setup();
    session.pointer.select(tile);
    session.socket.onMessage({ type: 'planned', ...tile, path });
    session.choices.onMove();
    expect(session.sent.at(-1)).toEqual({ type: 'walk', to: tile });
    expect(session.menu.shown).toBeNull();
  });

  it('Cancel and right clicks close the menu', () => {
    const session = setup();
    session.pointer.select(tile);
    session.choices.onCancel();
    expect(session.menu.shown).toBeNull();
    session.pointer.select(tile);
    session.pointer.cancel();
    expect(session.menu.shown).toBeNull();
  });

  it('asks the path again when your player steps while a tile is selected', () => {
    const session = setup();
    session.pointer.select(tile);
    session.socket.onMessage({ type: 'moved', id: 'a', col: 3, row: 2 });
    expect(session.sent.filter((s) => s.type === 'plan')).toHaveLength(2);
    session.socket.onMessage({ type: 'moved', id: 'someone else', col: 0, row: 0 });
    expect(session.sent.filter((s) => s.type === 'plan')).toHaveLength(2);
  });

  it('draws the chunks that come and drops the forgotten ones', () => {
    const session = setup();
    session.socket.onMessage({ type: 'forget', cx: 0, cy: 0 });
    expect(session.drawn.chunks).toEqual(['follow', 'show 0,0', 'forget 0,0']);
  });

  it('only points at the cells of chunks you hold', () => {
    const session = setup();
    session.pointer.hover({ col: 40, row: 2 });
    expect(session.drawn.hover, 'chunk (1, 0) is not held').toBeNull();
    session.pointer.select({ col: 40, row: 2 });
    expect(session.sent).toEqual([]);
    expect(session.menu.shown).toBeNull();
    session.pointer.hover({ col: 3, row: 2 });
    expect(session.drawn.hover).toEqual({ col: 3, row: 2 });
  });

  it("shows the region banner once your cell's region is known", () => {
    const session = setup();
    expect(session.drawn.banners).toEqual(['Sprinkle Meadow']);
    session.socket.onMessage({ type: 'moved', id: 'a', col: 3, row: 2 });
    expect(session.drawn.banners, 'not again in the same region').toEqual(['Sprinkle Meadow']);
  });

  it('tells when the connection ends', () => {
    const session = setup();
    session.socket.onClose();
    expect(session.isDisconnected()).toBe(true);
  });
});

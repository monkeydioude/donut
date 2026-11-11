import { describe, expect, it } from 'vitest';
import type { TilePoint } from '../iso/projection';
import type { Chunk, HeldChunksRef } from '../map/chunks';
import type { GameSocketHandlers } from '../net/gameSocket';
import type { Cell } from '../net/protocol';
import type { PointerHandlers } from '../ui/pointer';
import { emptyChunks } from '../map/chunks';
import { emptyPlayers } from './players';
import type { MenuChoices } from './session';
import { startPlayersSession, whereaboutsOf } from './session';

/** Chunk (0, 0) of the map, all grass, in region 1. */
function chunk(region = 1): Chunk {
  return { key: { cx: 0, cy: 0 }, version: 1, ground: Array(1024).fill('grass'), height: Array(1024).fill(0), region: Array(1024).fill(region), objects: [] };
}

function setup(held?: HeldChunksRef) {
  const sent: ({ type: 'plan' | 'walk'; to: Cell } | { type: 'teleport'; map: string; to: Cell })[] = [];
  const menu: { shown: { cell: TilePoint; moveEnabled: boolean } | null } = { shown: null };
  const drawn: { path: readonly TilePoint[]; hover: TilePoint | null; chunks: string[]; banners: string[] } = {
    path: [],
    hover: null,
    chunks: [],
    banners: [],
  };
  const chat: { lines: string[]; focused: number; type: (text: string) => void } = { lines: [], focused: 0, type: () => {} };
  let socket: GameSocketHandlers | undefined;
  let pointer: PointerHandlers | undefined;
  let choices: MenuChoices | undefined;
  let disconnected = false;
  const playing = startPlayersSession({
    map: { slug: 'world', name: 'World', width: 160, height: 160, wraps: true, chunkSize: 32, regions: [{ id: 1, name: 'Sprinkle Meadow', biome: 'meadow' }] },
    socketUrl: 'ws://test',
    openSocket: (_url, handlers) => {
      socket = handlers;
      return {
        plan: (to) => sent.push({ type: 'plan', to }),
        walk: (to) => sent.push({ type: 'walk', to }),
        teleport: (map, to) => sent.push({ type: 'teleport', map, to }),
        close: () => {},
      };
    },
    listenToPointer: (handlers) => {
      pointer = handlers;
      return { repick: () => {}, stop: () => {} };
    },
    createMenu: (c) => {
      choices = c;
      return { show: (cell, moveEnabled) => (menu.shown = { cell, moveEnabled }), hide: () => (menu.shown = null) };
    },
    createChatbox: (onLine) => {
      chat.type = onLine;
      return { addLine: (line) => chat.lines.push(`${line.kind}: ${line.text}`), focus: () => chat.focused++ };
    },
    players: { sync: () => {}, ownPosition: () => null, ownCell: () => ({ col: 2, row: 2 }), relift: () => drawn.chunks.push('relift') },
    chunks: {
      show: (c) => drawn.chunks.push(`show ${c.key.cx},${c.key.cy}`),
      forget: (key) => drawn.chunks.push(`forget ${key.cx},${key.cy}`),
      follow: () => drawn.chunks.push('follow'),
    },
    overlay: { showHover: (cell) => (drawn.hover = cell), showSelection: () => {}, showPath: (path) => (drawn.path = path) },
    banner: { show: (name) => drawn.banners.push(name) },
    ...(held ? { held } : {}),
    onDisconnect: () => (disconnected = true),
    now: () => 0,
  });
  socket!.onMessage({ type: 'welcome', you: 'a', players: [{ id: 'a', name: 'Ann', robot: '', col: 2, row: 2 }], stepMs: 200 });
  socket!.onMessage({ type: 'chunk', chunk: chunk() });
  return { playing, sent, menu, drawn, chat, socket: socket!, pointer: pointer!, choices: choices!, isDisconnected: () => disconnected, hovered: () => playing.hovered() };
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
    expect(session.drawn.chunks).toEqual(['follow', 'show 0,0', 'relift', 'forget 0,0']);
  });

  // ADR 0007: the views read the heights of the chunks held
  it('keeps the chunks held where the views read them, and lifts the players once a chunk comes', () => {
    const held = { current: emptyChunks() };
    const session = setup(held);
    expect(held.current.size).toBe(1);
    expect(session.drawn.chunks.indexOf('relift')).toBe(session.drawn.chunks.indexOf('show 0,0') + 1);
    session.socket.onMessage({ type: 'forget', cx: 0, cy: 0 });
    expect(held.current.size).toBe(0);
  });

  it('only points at the cells of chunks you hold', () => {
    const session = setup();
    session.pointer.hover({ col: 40, row: 2 });
    expect(session.drawn.hover, 'chunk (1, 0) is not held').toBeNull();
    expect(session.hovered()).toBeNull();
    session.pointer.select({ col: 40, row: 2 });
    expect(session.sent).toEqual([]);
    expect(session.menu.shown).toBeNull();
    session.pointer.hover({ col: 3, row: 2 });
    expect(session.drawn.hover).toEqual({ col: 3, row: 2 });
    expect(session.hovered()).toEqual({ col: 3, row: 2 });
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

// ADR 0005 test 19
describe('whereabouts', () => {
  it('gives your cell, the other players you see, and your region', () => {
    const session = setup();
    session.socket.onMessage({ type: 'joined', player: { id: 'b', name: 'Bob', robot: '', col: 7, row: 9 } });
    expect(session.playing.whereabouts()).toEqual({ you: { col: 2, row: 2 }, others: [{ col: 7, row: 9 }], region: 1 });
  });

  it('follows your steps and the players leaving', () => {
    const session = setup();
    session.socket.onMessage({ type: 'joined', player: { id: 'b', name: 'Bob', robot: '', col: 7, row: 9 } });
    session.socket.onMessage({ type: 'moved', id: 'a', col: 3, row: 2 });
    session.socket.onMessage({ type: 'left', id: 'b' });
    expect(session.playing.whereabouts()).toEqual({ you: { col: 3, row: 2 }, others: [], region: 1 });
  });

  it('gives nothing before the welcome', () => {
    expect(whereaboutsOf(emptyPlayers(), emptyChunks())).toEqual({ you: null, others: [], region: null });
  });
});

// ADR 0010 tests 11 to 14, through the session
describe('the chatbox and the game log', () => {
  it('a teleport line is echoed and sent; the others are answered here', () => {
    const session = setup();
    session.chat.type('/teleport world 1023 511');
    expect(session.sent).toEqual([{ type: 'teleport', map: 'world', to: { col: 1023, row: 511 } }]);
    session.chat.type('hello');
    session.chat.type('/fly');
    expect(session.sent).toHaveLength(1);
    expect(session.chat.lines).toEqual([
      'you: /teleport world 1023 511',
      'you: hello',
      'error: Chat comes later: only /teleport <map> <col> <row> works for now',
      'you: /fly',
      'error: Unknown command: /fly',
    ]);
  });

  it('a teleport places you, logs it, and clears the selection and its path', () => {
    const session = setup();
    session.pointer.select(tile);
    session.socket.onMessage({ type: 'planned', ...tile, path });
    expect(session.menu.shown?.moveEnabled).toBe(true);

    session.socket.onMessage({ type: 'teleported', col: 100, row: 100, walkable: false });
    expect(session.menu.shown).toBeNull();
    expect(session.drawn.path).toEqual([]);
    expect(session.playing.whereabouts().you).toEqual({ col: 100, row: 100 });
    expect(session.chat.lines).toEqual(['log: Teleported to world (100, 100)', "warning: (100, 100) isn't walkable: teleport again to leave"]);
  });

  it('logs the others coming and going by name, Move chosen, stops and refusals', () => {
    const session = setup();
    session.socket.onMessage({ type: 'joined', player: { id: 'b', name: 'Bob', robot: '', col: 3, row: 3 } });
    session.socket.onMessage({ type: 'left', id: 'b' });
    session.pointer.select(tile);
    session.socket.onMessage({ type: 'planned', ...tile, path });
    session.choices.onMove();
    session.socket.onMessage({ type: 'stopped', col: 3, row: 2 });
    session.socket.onMessage({ type: 'refused', request: 'teleport', reason: 'unknown map: mars' });
    expect(session.chat.lines).toEqual([
      'log: Bob entered the area',
      'log: Bob left the area',
      'log: You head to (4, 2)',
      'log: You stopped at (3, 2)',
      'error: unknown map: mars',
    ]);
  });

  it('opens the chat on demand', () => {
    const session = setup();
    session.playing.openChat();
    expect(session.chat.focused).toBe(1);
  });
});

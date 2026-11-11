import { describe, expect, it } from 'vitest';
import { worldDescription } from '../testing/maps';
import type { ServerMessage } from '../net/protocol';
import { initialGameLog, logMessage, logMove, WALK_PAUSE_MS, type GameLogState, type LogLine, logMapSwitch } from './gameLog';

const bob = { id: 'b', name: 'Bob', robot: '', col: 3, row: 3 };
const names: Record<string, string> = { a: 'Ann', b: 'Bob' };

function linesOf(message: ServerMessage, state = initialGameLog(), now = 0): { state: GameLogState; lines: LogLine[] } {
  return logMessage(state, message, { you: 'a', nameOf: (id) => names[id] ?? null, map: 'world', now });
}

const log = (text: string): LogLine => ({ kind: 'log', text });

// ADR 0010 test 14
describe('logMessage', () => {
  it('another player coming and going', () => {
    expect(linesOf({ type: 'joined', player: bob }).lines).toEqual([log('Bob entered the area')]);
    expect(linesOf({ type: 'left', id: 'b' }).lines).toEqual([log('Bob left the area')]);
    expect(linesOf({ type: 'left', id: 'zz' }).lines).toEqual([log('Someone left the area')]);
  });

  it('you coming is nothing', () => {
    expect(linesOf({ type: 'joined', player: { ...bob, id: 'a', name: 'Ann' } }).lines).toEqual([]);
  });

  it("another player's walk is one line: its first step, and the first after a pause", () => {
    const step = (state: GameLogState, now: number) => linesOf({ type: 'moved', id: 'b', col: 4, row: 3 }, state, now);
    const first = step(initialGameLog(), 1000);
    expect(first.lines).toEqual([log('Bob is walking')]);
    const second = step(first.state, 1000 + WALK_PAUSE_MS - 1);
    expect(second.lines).toEqual([]);
    const third = step(second.state, 1000 + WALK_PAUSE_MS - 1 + WALK_PAUSE_MS);
    expect(third.lines).toEqual([log('Bob is walking')]);
  });

  it('your own steps are nothing', () => {
    expect(linesOf({ type: 'moved', id: 'a', col: 4, row: 3 }).lines).toEqual([]);
  });

  it('a player who left starts a new walk when they come back', () => {
    const walked = linesOf({ type: 'moved', id: 'b', col: 4, row: 3 }, initialGameLog(), 1000);
    const gone = linesOf({ type: 'left', id: 'b' }, walked.state, 1100);
    expect(linesOf({ type: 'moved', id: 'b', col: 5, row: 3 }, gone.state, 1200).lines).toEqual([log('Bob is walking')]);
  });

  it('your stop, teleport and refusals', () => {
    expect(linesOf({ type: 'stopped', col: 1030, row: 505 }).lines).toEqual([log('You stopped at (1030, 505)')]);
    expect(linesOf({ type: 'teleported', col: 1023, row: 511, walkable: true }).lines).toEqual([log('Teleported to world (1023, 511)')]);
    expect(linesOf({ type: 'teleported', col: 600, row: 300, walkable: false }).lines).toEqual([
      log('Teleported to world (600, 300)'),
      { kind: 'warning', text: "(600, 300) isn't walkable: teleport again to leave" },
    ]);
    expect(linesOf({ type: 'refused', request: 'teleport', reason: 'unknown map: mars' }).lines).toEqual([{ kind: 'error', text: 'unknown map: mars' }]);
  });

  it('chunks, paths and the welcome are nothing', () => {
    const silent: ServerMessage[] = [
      { type: 'welcome', you: 'a', players: [bob], stepMs: 200, map: worldDescription() },
      { type: 'planned', col: 1, row: 1, path: [], action: 'move' },
      { type: 'walking', path: [] },
      { type: 'forget', cx: 0, cy: 0 },
    ];
    for (const message of silent) expect(linesOf(message).lines, message.type).toEqual([]);
  });
});

describe('logMove', () => {
  it('says where you head', () => {
    expect(logMove({ col: 1040, row: 500 })).toEqual(log('You head to (1040, 500)'));
  });
});

// ADR 0012 test 30
describe('logMapSwitch', () => {
  it('"You enter" a city, "You leave" it, nothing between maps that wrap', () => {
    const city = { ...worldDescription(), slug: 'doma-1', name: 'Doma-1', width: 32, height: 32, wraps: false };
    expect(logMapSwitch(worldDescription(), city)).toEqual([{ kind: 'log', text: 'You enter Doma-1' }]);
    expect(logMapSwitch(city, worldDescription())).toEqual([{ kind: 'log', text: 'You leave Doma-1' }]);
    expect(logMapSwitch(worldDescription(), { ...worldDescription(), slug: 'mars' })).toEqual([]);
  });
});

import { describe, expect, it } from 'vitest';
import { withChunk, emptyChunks, type Chunk } from '../map/chunks';
import type { GroundKind, MapObject } from '../map/worldMap';
import { actionAt, actionLabel } from './actions';

function chunkWith(cells: Record<number, GroundKind>, objects: MapObject[] = []): Chunk {
  const ground = Array<GroundKind>(1024).fill('floor');
  for (const [i, kind] of Object.entries(cells)) ground[Number(i)] = kind;
  return { key: { cx: 0, cy: 0 }, version: 1, ground, height: Array(1024).fill(0), region: Array(1024).fill(1), objects };
}

const world = { width: 160, height: 160, wraps: true };
const city = { width: 32, height: 32, wraps: false };

// ADR 0012 test 25
describe('actionAt', () => {
  it("a dome's 4 cells are Enter, the cells beside it Move", () => {
    const held = withChunk(emptyChunks(), chunkWith({}, [{ id: 'd', kind: 'dome', col: 4, row: 6, leadsTo: 'doma-1' }]));
    for (const cell of [{ col: 4, row: 6 }, { col: 5, row: 6 }, { col: 4, row: 7 }, { col: 5, row: 7 }]) {
      expect(actionAt(cell, held, world), `${cell.col},${cell.row}`).toBe('enter');
    }
    expect(actionAt({ col: 6, row: 6 }, held, world)).toBe('move');
    expect(actionAt({ col: 3, row: 6 }, held, world)).toBe('move');
  });

  it('in a city, the void, a wall and past the map are Exit; floor and grass are Move', () => {
    const held = withChunk(emptyChunks(), chunkWith({ 0: 'void', 33: 'wall', 66: 'grass' }));
    expect(actionAt({ col: 0, row: 0 }, held, city)).toBe('exit');
    expect(actionAt({ col: 1, row: 1 }, held, city)).toBe('exit');
    expect(actionAt({ col: -3, row: 5 }, held, city)).toBe('exit');
    expect(actionAt({ col: 5, row: 40 }, held, city)).toBe('exit');
    expect(actionAt({ col: 2, row: 2 }, held, city)).toBe('move');
    expect(actionAt({ col: 9, row: 9 }, held, city)).toBe('move');
  });

  it("on the world, the void is Move: there's no outside", () => {
    const held = withChunk(emptyChunks(), chunkWith({ 0: 'void' }));
    expect(actionAt({ col: 0, row: 0 }, held, world)).toBe('move');
  });

  it("the menu's words", () => {
    expect(['move', 'enter', 'exit'].map((a) => actionLabel(a as 'move'))).toEqual(['Move', 'Enter', 'Exit']);
  });
});

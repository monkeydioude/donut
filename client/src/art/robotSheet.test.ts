import { describe, expect, it } from 'vitest';
import { parseRobotSheet, robotToDraw } from './robotSheet';

const size = { width: 32, height: 16 };
const sheet = {
  cell: 16,
  sprites: [
    { slug: 'dome-head', x: 0, y: 0, rowId: 797552 },
    { slug: 'rusty-toy', x: 16, y: 0, rowId: 222810 },
  ],
};

// ADR 0006 test 17
describe('parseRobotSheet', () => {
  it('reads the cell size and each robot with its cell', () => {
    expect(parseRobotSheet(sheet, size)).toEqual({
      cell: 16,
      robots: [
        { slug: 'dome-head', x: 0, y: 0 },
        { slug: 'rusty-toy', x: 16, y: 0 },
      ],
    });
  });

  // ADR 0006 test 18
  it('refuses a malformed sheet', () => {
    const withSprites = (sprites: unknown) => ({ ...sheet, sprites });
    const bad: Record<string, unknown> = {
      'not an object': [sheet],
      'no sprites': { cell: 16 },
      'empty sprites': withSprites([]),
      'a duplicate slug': withSprites([sheet.sprites[0], { ...sheet.sprites[1], slug: 'dome-head' }]),
      'a cell past the right edge': withSprites([{ slug: 'a', x: 17, y: 0 }]),
      'a cell past the bottom': withSprites([{ slug: 'a', x: 0, y: 1 }]),
      'a negative cell': withSprites([{ slug: 'a', x: -16, y: 0 }]),
      'a cell not a whole number': withSprites([{ slug: 'a', x: 0.5, y: 0 }]),
      'no slug': withSprites([{ x: 0, y: 0 }]),
      'no cell size': { sprites: sheet.sprites },
      'a cell size of 0': { ...sheet, cell: 0 },
    };
    for (const [name, json] of Object.entries(bad)) {
      expect(parseRobotSheet(json, size), name).toBeNull();
    }
  });
});

// ADR 0006 test 19
describe('robotToDraw', () => {
  const robots: readonly [{ slug: string }, ...{ slug: string }[]] = [{ slug: 'dome-head' }, { slug: 'rusty-toy' }];

  it('draws a known slug as itself', () => {
    expect(robotToDraw(robots, 'rusty-toy')).toBe(robots[1]);
  });

  it('draws the first robot for an unknown or empty slug', () => {
    expect(robotToDraw(robots, 'gone-bot')).toBe(robots[0]);
    expect(robotToDraw(robots, '')).toBe(robots[0]);
  });
});

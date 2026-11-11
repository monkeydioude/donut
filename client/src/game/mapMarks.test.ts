import { describe, expect, it } from 'vitest';
import { blinkOn, donutMark, donutPlace, donutShape, mapCellAt, mapScale, rectangleMark, regionLine } from './mapMarks';
import { cityDescription, worldDescription } from '../testing/maps';

const world = { width: 1024, height: 512, wraps: true };
const ring = { center: { x: 32, y: 32 }, outer: 28, inner: 12 };

const expectAt = (actual: { x: number; y: number }, expected: { x: number; y: number }) => {
  expect(actual.x).toBeCloseTo(expected.x);
  expect(actual.y).toBeCloseTo(expected.y);
};

// ADR 0005 test 14
describe('rectangleMark', () => {
  it("is at the middle of the cell's pixel, times the scale", () => {
    expect(rectangleMark({ col: 10, row: 4 }, 1)).toEqual({ x: 10.5, y: 4.5 });
    expect(rectangleMark({ col: 10, row: 4 }, 2)).toEqual({ x: 21, y: 9 });
    expect(rectangleMark({ col: 10, row: 4 }, 0.5)).toEqual({ x: 5.25, y: 2.25 });
  });
});

// ADR 0005 test 15
describe('donutMark', () => {
  const at = (col: number, row: number) => donutMark({ col, row }, world, ring).at;

  it('goes around the ring clockwise from the top, with the columns', () => {
    expectAt(at(0, 0), { x: 32, y: 4 });
    expectAt(at(256, 0), { x: 60, y: 32 });
    expectAt(at(512, 0), { x: 32, y: 60 });
    expectAt(at(768, 0), { x: 4, y: 32 });
  });

  it('goes from the outer edge to the inner edge and back, with the rows', () => {
    expectAt(at(0, 0), { x: 32, y: 32 - 28 });
    expectAt(at(0, 256), { x: 32, y: 32 - 12 });
    expectAt(at(0, 128), { x: 32, y: 32 - 20 });
    expectAt(at(0, 384), { x: 32, y: 32 - 20 });
  });
});

// ADR 0005 test 16
describe('donutShape', () => {
  const shapes = (rows: number[]) => rows.map((row) => donutShape(row, 512));

  it('is a square within a chunk of an equator, across the seam too', () => {
    expect(shapes([0, 32, 480, 511, 224, 256, 288])).toEqual(Array(7).fill('square'));
  });

  it('is ▲ on the top face', () => {
    expect(shapes([33, 128, 223])).toEqual(['up', 'up', 'up']);
  });

  it('is ▼ on the bottom face', () => {
    expect(shapes([289, 384, 479])).toEqual(['down', 'down', 'down']);
  });

  it('comes with the donut mark', () => {
    expect(donutMark({ col: 0, row: 128 }, world, ring).shape).toBe('up');
  });
});

// ADR 0005 test 17
describe('blinkOn', () => {
  it('shows for 0.5 s, then hides for 0.5 s', () => {
    expect([0, 499, 500, 999, 1000].map(blinkOn)).toEqual([true, true, false, false, true]);
  });
});

// ADR 0005 test 18
describe('mapScale', () => {
  const map = { width: 1024, height: 512 };

  it('is the biggest of ×2, ×1 and ×½ that fits', () => {
    expect(mapScale(map, { width: 2100, height: 1100 })).toBe(2);
    expect(mapScale(map, { width: 2100, height: 1000 })).toBe(1);
    expect(mapScale(map, { width: 1100, height: 600 })).toBe(1);
    expect(mapScale(map, { width: 1000, height: 600 })).toBe(0.5);
  });

  it('is ×½ when nothing fits', () => {
    expect(mapScale(map, { width: 300, height: 200 })).toBe(0.5);
  });
});

describe('mapCellAt', () => {
  const map = { width: 1024, height: 512 };

  it("is the cell whose pixel holds the point, at the map's scale", () => {
    expect(mapCellAt({ x: 10.5, y: 4.5 }, 1, map)).toEqual({ col: 10, row: 4 });
    expect(mapCellAt({ x: 21, y: 9 }, 2, map)).toEqual({ col: 10, row: 4 });
    expect(mapCellAt({ x: 5.25, y: 2.25 }, 0.5, map)).toEqual({ col: 10, row: 4 });
  });

  it('is the cell of its own mark', () => {
    for (const scale of [2, 1, 0.5]) expect(mapCellAt(rectangleMark({ col: 1023, row: 511 }, scale), scale, map)).toEqual({ col: 1023, row: 511 });
  });

  it('is null off the map', () => {
    expect(mapCellAt({ x: -0.1, y: 4 }, 1, map)).toBeNull();
    expect(mapCellAt({ x: 1024, y: 4 }, 1, map)).toBeNull();
    expect(mapCellAt({ x: 4, y: 1024 }, 2, map)).toBeNull();
  });
});

// ADR 0012 test 29
describe('the map screen in a city', () => {
  it('shows Doma-1 at ×8 on a 1080p screen, and the world as before', () => {
    const room = { width: 1920 - 48, height: 1080 - 48 };
    expect(mapScale({ width: 32, height: 32 }, room)).toBe(8);
    expect(mapScale({ width: 2048, height: 1024 }, room)).toBe(0.5);
    expect(mapScale({ width: 160, height: 160 }, room)).toBe(4);
  });

  it("puts you on the donut: your cell on the world, the city's dome in a city, nowhere on a map that stands nowhere", () => {
    expect(donutPlace(worldDescription(), { col: 7, row: 9 })).toEqual({ cell: { col: 7, row: 9 }, grid: { width: 1024, height: 512, wraps: true } });
    expect(donutPlace(worldDescription(), null)).toBeNull();
    expect(donutPlace(cityDescription(), { col: 3, row: 3 })).toEqual({ cell: { col: 1022, row: 510 }, grid: { width: 1024, height: 512, wraps: true } });
    expect(donutPlace({ ...cityDescription(), standsOn: null }, { col: 3, row: 3 })).toBeNull();
  });

  it("leaves the region line out when it's the map's name", () => {
    expect(regionLine('Doma-1', 'Doma-1')).toBe('');
    expect(regionLine('World', 'Sprinkle Meadow')).toBe('Sprinkle Meadow');
    expect(regionLine('World', null)).toBe('');
  });
});

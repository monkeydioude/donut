import { describe, expect, it } from 'vitest';
import { groundCenterAt, pickOnHeights } from './heights';
import { createIsoProjection, type TilePoint } from './projection';

const projection = createIsoProjection(32, 16);
const heights = { projection, levelPx: 8 };

/** Heights by cell, "col,row", 0 for the others. */
function heightsOf(cells: Record<string, number>) {
  return (cell: TilePoint) => cells[`${cell.col},${cell.row}`] ?? 0;
}

// ADR 0007 test 28
describe('groundCenterAt', () => {
  it('draws a cell at 2 16 px higher than at 0', () => {
    expect(groundCenterAt(heights, { col: 5, row: 5 }, 2)).toEqual({ x: 0, y: 80 - 16 });
    expect(groundCenterAt(heights, { col: 5, row: 5 }, -1)).toEqual({ x: 0, y: 88 });
  });
});

// ADR 0007 test 29
describe('pickOnHeights', () => {
  it('over flat ground, the cell under the pointer, as before', () => {
    for (const cell of [{ col: 5, row: 5 }, { col: 0, row: 3 }, { col: -2, row: 7 }]) {
      expect(pickOnHeights(heights, projection.toScreen(cell), () => 0)).toEqual(cell);
    }
  });

  it("over a cell at 2, its top picks it, not the cell its flat position gives", () => {
    const raised = groundCenterAt(heights, { col: 5, row: 5 }, 2);
    expect(projection.toTile(raised)).toEqual({ col: 4, row: 4 });
    expect(pickOnHeights(heights, raised, heightsOf({ '5,5': 2 }))).toEqual({ col: 5, row: 5 });
  });

  it('over the face of a raised cell, that cell', () => {
    // (8, 80) is under the lower right edge of (5, 5) drawn at 1: on its face, in front of (6, 5) and (5, 6) at 0
    expect(pickOnHeights(heights, { x: 8, y: 80 }, heightsOf({ '5,5': 1 }))).toEqual({ col: 5, row: 5 });
    expect(pickOnHeights(heights, { x: 8, y: 80 }, () => 0), 'flat, the same point is on (5, 5) too').toEqual({ col: 5, row: 5 });
    expect(pickOnHeights(heights, { x: 8, y: 87 }, heightsOf({ '5,5': 1 })), 'past the face').toEqual({ col: 6, row: 5 });
  });

  it('a raised cell in front hides the cell behind it', () => {
    const behind = projection.toScreen({ col: 5, row: 5 });
    expect(pickOnHeights(heights, behind, heightsOf({ '6,6': 2 }))).toEqual({ col: 6, row: 6 });
  });

  it('picks the water down in its bed, the banks in front hiding its lower half', () => {
    const center = groundCenterAt(heights, { col: 5, row: 5 }, -1);
    expect(pickOnHeights(heights, { x: center.x, y: center.y - 1 }, heightsOf({ '5,5': -1 }))).toEqual({ col: 5, row: 5 });
    expect(pickOnHeights(heights, { x: center.x, y: center.y + 1 }, heightsOf({ '5,5': -1 }))).toEqual({ col: 6, row: 6 });
  });

  it("between two levels, the face of the cell in front, not the cell behind it", () => {
    // (4, 78) is on the lower left face of (5, 4), which stands over the water at (5, 5); (4, 4) is behind it
    expect(pickOnHeights(heights, { x: 4, y: 78 }, heightsOf({ '5,5': -1 }))).toEqual({ col: 5, row: 4 });
    expect(pickOnHeights(heights, { x: 0.5, y: 79 }, heightsOf({ '5,5': -1 }))).toEqual({ col: 5, row: 4 });
  });
});

import { describe, expect, it } from 'vitest';
import { coordinatesText } from './coordinates';

describe('coordinatesText', () => {
  it('is the column, then the row', () => {
    expect(coordinatesText({ col: 12, row: 340 })).toBe('12, 340');
    expect(coordinatesText({ col: 0, row: 0 })).toBe('0, 0');
  });

  it('is empty without a cell', () => {
    expect(coordinatesText(null)).toBe('');
  });
});

import { describe, expect, it } from 'vitest';
import { createExplorations, explore, exploredShare, exploredText, isExplored, noExploration } from './exploration';

const world = { width: 1024, height: 512, wraps: true };

// ADR 0005 test 12
describe('explore', () => {
  it('counts a chunk once', () => {
    const once = explore(noExploration, { cx: 3, cy: 4 });
    expect(explore(once, { cx: 3, cy: 4 })).toBe(once);
    expect(once.size).toBe(1);
    expect(isExplored(once, { cx: 3, cy: 4 })).toBe(true);
    expect(isExplored(once, { cx: 4, cy: 3 })).toBe(false);
  });

  it('leaves the previous exploration untouched', () => {
    const once = explore(noExploration, { cx: 0, cy: 0 });
    explore(once, { cx: 1, cy: 0 });
    expect(once.size).toBe(1);
    expect(noExploration.size).toBe(0);
  });
});

// ADR 0005 test 13
describe('the explored share', () => {
  const chunks = (count: number) =>
    Array.from({ length: count }, (_, i) => ({ cx: i % 32, cy: Math.floor(i / 32) })).reduce(explore, noExploration);

  it('9 chunks of 512 are 1.8%', () => {
    expect(exploredText(exploredShare(chunks(9), world))).toBe('1.8% explored');
  });

  it('none is 0.0%, all are 100.0%', () => {
    expect(exploredText(exploredShare(noExploration, world))).toBe('0.0% explored');
    expect(exploredText(exploredShare(chunks(512), world))).toBe('100.0% explored');
  });
});

// ADR 0012 test 31
describe('explorations, per map', () => {
  it('going back to a map finds its explored chunks; other maps are apart', () => {
    const explorations = createExplorations();
    explorations.explore('world', { cx: 3, cy: 4 });
    explorations.explore('doma-1', { cx: 0, cy: 0 });
    expect(isExplored(explorations.of('world'), { cx: 3, cy: 4 })).toBe(true);
    expect(isExplored(explorations.of('world'), { cx: 0, cy: 0 })).toBe(false);
    expect(isExplored(explorations.of('doma-1'), { cx: 0, cy: 0 })).toBe(true);
    expect(explorations.of('mars').size).toBe(0);
  });
});

import { describe, expect, it } from 'vitest';
import { addScroll, initialWheel, scrollPx, type WheelState } from './wheel';

const config = { stepPx: 100, pauseMs: 150 };

/** Feeds the scrolls in order, and gives the step each one made. */
function feed(scrolls: readonly { px: number; at: number }[], from: WheelState = initialWheel) {
  let state = from;
  const steps: (string | null)[] = [];
  for (const { px, at } of scrolls) {
    const next = addScroll(state, px, at, config);
    state = next.state;
    steps.push(next.step);
  }
  return { state, steps };
}

// ADR 0004 test 9
describe('a mouse notch', () => {
  it('gives one step, up or down', () => {
    expect(feed([{ px: -100, at: 0 }]).steps).toEqual(['up']);
    expect(feed([{ px: 100, at: 0 }]).steps).toEqual(['down']);
  });
});

// ADR 0004 test 10
describe('trackpad scrolls', () => {
  it('ten scrolls of 10 px give one step, on the tenth', () => {
    const { steps } = feed(Array.from({ length: 10 }, (_, i) => ({ px: -10, at: i * 16 })));
    expect(steps.filter(Boolean)).toEqual(['up']);
    expect(steps.at(-1)).toBe('up');
  });
});

// ADR 0004 test 11
describe('scrollPx', () => {
  it('counts 40 px a line and 800 px a page', () => {
    expect(scrollPx(3, 1)).toBe(120);
    expect(scrollPx(1, 2)).toBe(800);
    expect(scrollPx(-53, 0)).toBe(-53);
  });

  it('3 lines give one step', () => {
    expect(feed([{ px: scrollPx(3, 1), at: 0 }]).steps).toEqual(['down']);
  });
});

// ADR 0004 test 12
describe('the pause after a step', () => {
  it('drops scrolls for 150 ms, then counts them again', () => {
    const { steps } = feed([
      { px: 100, at: 0 },
      { px: 100, at: 100 },
      { px: 100, at: 150 },
    ]);
    expect(steps).toEqual(['down', null, 'down']);
  });

  it('dropped scrolls add nothing up', () => {
    const { steps } = feed([
      { px: 100, at: 0 },
      { px: 90, at: 100 },
      { px: 50, at: 200 },
    ]);
    expect(steps).toEqual(['down', null, null]);
  });
});

// ADR 0004 test 13
describe('a change of direction', () => {
  it('restarts the count', () => {
    const { steps } = feed([
      { px: 60, at: 0 },
      { px: -60, at: 16 },
      { px: -40, at: 32 },
    ]);
    expect(steps).toEqual([null, null, 'up']);
  });
});

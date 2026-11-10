import { describe, expect, it } from 'vitest';
import { BANNER_COOLDOWN_MS, enterRegion, initialBanner, type BannerState } from './banner';

/** Enters the regions in turn, a second apart, and gives the banners shown. */
function walk(regions: (number | null)[], state: BannerState = initialBanner(), start = 0): (number | null)[] {
  return regions.map((region, i) => {
    const next = enterRegion(state, region, start + i * 1000);
    state = next.state;
    return next.show;
  });
}

// ADR 0003 test 32
describe('enterRegion', () => {
  it('shows at spawn, once your cell’s region is known', () => {
    expect(walk([null, 3])).toEqual([null, 3]);
  });

  it('shows on entering another region, not while staying in one', () => {
    expect(walk([3, 3, 5, 5])).toEqual([3, null, 5, null]);
  });

  it('does not show the same region again within 30 s', () => {
    expect(walk([3, 5, 3, 5])).toEqual([3, 5, null, null]);
  });

  it('shows it again after 30 s', () => {
    const first = enterRegion(initialBanner(), 3, 0);
    const away = enterRegion(first.state, 5, 1000);
    expect(enterRegion(away.state, 3, BANNER_COOLDOWN_MS - 1).show).toBeNull();
    expect(enterRegion(away.state, 3, BANNER_COOLDOWN_MS).show).toBe(3);
  });

  it('shows nothing for region 0, but entering a region from it shows', () => {
    expect(walk([0, 0, 4])).toEqual([null, null, 4]);
  });
});

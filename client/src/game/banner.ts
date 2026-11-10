/** A region's banner isn't shown again within this time, so walking along a border doesn't flood the screen. */
export const BANNER_COOLDOWN_MS = 30_000;

/** The region you stand in, and when each region's banner was last shown. */
export interface BannerState {
  region: number | null;
  shownAt: ReadonlyMap<number, number>;
}

export function initialBanner(): BannerState {
  return { region: null, shownAt: new Map() };
}

/**
 * The banner state once your cell's region is known to be `region` (null: not known yet, its chunk hasn't come),
 * and the region whose banner to show, if any. A banner shows at spawn and whenever you enter another region,
 * but not for region 0 (none), nor for a region shown less than 30 s ago.
 */
export function enterRegion(state: BannerState, region: number | null, now: number): { state: BannerState; show: number | null } {
  if (region === null || region === state.region) return { state, show: null };
  const last = state.shownAt.get(region);
  if (region === 0 || (last !== undefined && now - last < BANNER_COOLDOWN_MS)) {
    return { state: { ...state, region }, show: null };
  }
  return { state: { region, shownAt: new Map(state.shownAt).set(region, now) }, show: region };
}

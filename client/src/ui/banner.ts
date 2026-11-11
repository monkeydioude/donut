import { silent, type Log } from '../debug/log';

export interface RegionBanner {
  /** Shows a region's name, Dark Souls style but cute: it fades in, stays, and fades out. A new name replaces it. */
  show(name: string): void;
}

/** The region banner: DOM over the canvas, centered a bit above the middle, with no window behind it. */
export function createRegionBanner(host: HTMLElement, log: Log = silent): RegionBanner {
  const root = document.createElement('div');
  root.className = 'region-banner';
  root.setAttribute('aria-live', 'polite');
  const stars = document.createElement('div');
  stars.className = 'region-banner-stars';
  stars.textContent = '·  ✦  ·';
  const name = document.createElement('div');
  name.className = 'region-banner-name';
  const rule = document.createElement('div');
  rule.className = 'region-banner-rule';
  rule.textContent = '♥';
  root.append(stars, name, rule);
  host.appendChild(root);

  return {
    show(text) {
      log('shown', text);
      name.textContent = text;
      // restart the animation, even when a banner is still showing
      root.classList.remove('showing');
      void root.offsetWidth;
      root.classList.add('showing');
    },
  };
}

/** A banner that can wait: while the screen is dark between maps, the last name asked shows once it's light again. */
export interface HeldBanner extends RegionBanner {
  hold(): void;
  release(): void;
}

/** Holds the names shown while held, and shows the last one on release (ADR 0012). */
export function createHeldBanner(banner: RegionBanner): HeldBanner {
  let held = false;
  let waiting: string | null = null;
  return {
    show(name) {
      if (held) waiting = name;
      else banner.show(name);
    },
    hold() {
      held = true;
    },
    release() {
      held = false;
      if (waiting !== null) banner.show(waiting);
      waiting = null;
    },
  };
}

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

import { silent, type Log } from '../debug/log';

/** The screen going dark and back between maps (ADR 0012). */
export interface Fade {
  /** Darkens the screen, then calls `done`. */
  out(done: () => void): void;
  /** Lights the screen again, then calls `done`. */
  in(done?: () => void): void;
}

/**
 * A black layer over the game, dark at first: the first map fades in. Each way takes `ms`. The timers are the
 * page's; tests give their own.
 */
export function createFade(host: HTMLElement, ms: number, log: Log = silent, later: (fn: () => void, ms: number) => void = setTimeout): Fade {
  const layer = document.createElement('div');
  layer.className = 'fade';
  layer.style.transitionDuration = `${ms}ms`;
  host.appendChild(layer);
  return {
    out(done) {
      log('fade out');
      layer.classList.remove('fade-clear');
      later(done, ms);
    },
    in(done) {
      log('fade in');
      layer.classList.add('fade-clear');
      later(() => done?.(), ms);
    },
  };
}

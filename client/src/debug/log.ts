/** Writes a debug line: a message, then any data worth seeing. */
export type Log = (message: string, ...data: unknown[]) => void;

/** Logs nothing: the default of every module, and what tests use. */
export const silent: Log = () => {};

export interface Logger {
  /** A log for one part of the game, its lines prefixed with [donut:part]. */
  scope(part: string): Log;
}

/** Debug logs in the browser console, through console.log so they show without the "Verbose" level. */
export function createLogger(enabled: boolean, sink: (...args: unknown[]) => void = console.log): Logger {
  return {
    scope: (part) => (enabled ? (message, ...data) => sink(`[donut:${part}]`, message, ...data) : silent),
  };
}

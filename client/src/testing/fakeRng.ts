import type { Rng } from '../core/rng';

/** Rng that returns the given values in a loop. */
export function sequenceRng(...values: number[]): Rng {
  let index = 0;
  return () => values[index++ % values.length] ?? 0;
}

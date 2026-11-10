import { describe, expect, it } from 'vitest';
import { createLogger } from './log';

describe('createLogger', () => {
  it('prefixes each line with its part of the game', () => {
    const lines: unknown[][] = [];
    createLogger(true, (...args) => lines.push(args)).scope('socket')('open', { code: 1 });
    expect(lines).toEqual([['[donut:socket]', 'open', { code: 1 }]]);
  });

  it('writes nothing when off', () => {
    const lines: unknown[][] = [];
    createLogger(false, (...args) => lines.push(args)).scope('socket')('open');
    expect(lines).toEqual([]);
  });
});

import { describe, expect, it } from 'vitest';
import { addLine, chatKeyOf, emptyHistory, MAX_LINES, scrollBy, VISIBLE_LINES, visibleLines, type ChatHistory } from './chatHistory';
import type { LogLine } from './gameLog';

const line = (n: number): LogLine => ({ kind: 'log', text: `line ${n}` });
const texts = (history: ChatHistory) => visibleLines(history).map((l) => l.text);

function filled(count: number): ChatHistory {
  let history = emptyHistory();
  for (let n = 1; n <= count; n++) history = addLine(history, line(n));
  return history;
}

// ADR 0010 test 15
describe('the chat history', () => {
  it('shows the last 6 lines', () => {
    expect(VISIBLE_LINES).toBe(6);
    expect(texts(filled(3))).toEqual(['line 1', 'line 2', 'line 3']);
    expect(texts(filled(8))).toEqual(['line 3', 'line 4', 'line 5', 'line 6', 'line 7', 'line 8']);
  });

  it('keeps 1,000 lines: the 1,001st drops the first', () => {
    expect(MAX_LINES).toBe(1000);
    const history = filled(MAX_LINES + 1);
    expect(history.lines).toHaveLength(MAX_LINES);
    expect(history.lines[0]?.text).toBe('line 2');
  });

  it('scrolls up one line at a time, stops at the first, and down stops at the last', () => {
    let history = scrollBy(filled(8), 1);
    expect(texts(history)).toEqual(['line 2', 'line 3', 'line 4', 'line 5', 'line 6', 'line 7']);
    history = scrollBy(history, 5);
    expect(texts(history)[0], 'stops at the first line').toBe('line 1');
    history = scrollBy(history, -10);
    expect(texts(history).at(-1), 'stops at the last line').toBe('line 8');
    expect(scrollBy(filled(3), 1).scrolledUp, 'nothing to scroll with few lines').toBe(0);
  });

  it('a new line scrolls back to the bottom', () => {
    const history = addLine(scrollBy(filled(8), 2), line(9));
    expect(texts(history).at(-1)).toBe('line 9');
  });
});

describe('chatKeyOf', () => {
  it('Enter sends, Escape blurs, the arrows scroll, other keys nothing', () => {
    expect(chatKeyOf('Enter')).toBe('send');
    expect(chatKeyOf('Escape')).toBe('blur');
    expect(chatKeyOf('ArrowUp')).toBe('up');
    expect(chatKeyOf('ArrowDown')).toBe('down');
    expect(chatKeyOf('y')).toBeNull();
  });
});

import type { LogLine } from './gameLog';

/** How many lines the chatbox shows at once, and how many it keeps: past that, the oldest goes (ADR 0010). */
export const VISIBLE_LINES = 6;
export const MAX_LINES = 1000;

/** The chatbox's history: its lines, and how far up from the bottom the view is scrolled. */
export interface ChatHistory {
  lines: readonly LogLine[];
  scrolledUp: number;
}

export function emptyHistory(): ChatHistory {
  return { lines: [], scrolledUp: 0 };
}

/** A new line goes at the bottom, and the view comes back to it. */
export function addLine(history: ChatHistory, line: LogLine): ChatHistory {
  const lines = [...history.lines, line];
  return { lines: lines.length > MAX_LINES ? lines.slice(lines.length - MAX_LINES) : lines, scrolledUp: 0 };
}

/** Scrolls the view by lines: up is positive. It stops at the first line and at the last. */
export function scrollBy(history: ChatHistory, lines: number): ChatHistory {
  const most = Math.max(0, history.lines.length - VISIBLE_LINES);
  return { ...history, scrolledUp: Math.min(most, Math.max(0, history.scrolledUp + lines)) };
}

/** The lines in view: the last VISIBLE_LINES, moved up by the scroll. */
export function visibleLines(history: ChatHistory): readonly LogLine[] {
  const end = history.lines.length - history.scrolledUp;
  return history.lines.slice(Math.max(0, end - VISIBLE_LINES), end);
}

/** What a key does in the focused input: send the line, blur, or scroll the history. */
export type ChatKey = 'send' | 'blur' | 'up' | 'down';

export function chatKeyOf(key: string): ChatKey | null {
  switch (key) {
    case 'Enter':
      return 'send';
    case 'Escape':
      return 'blur';
    case 'ArrowUp':
      return 'up';
    case 'ArrowDown':
      return 'down';
    default:
      return null;
  }
}

import { silent, type Log } from '../debug/log';
import type { Cell } from '../net/protocol';
import { parseLine } from './commands';
import type { LogLine } from './gameLog';

export interface TypedLineDeps {
  addLine(line: LogLine): void;
  teleport(map: string, to: Cell): void;
  log?: Log;
}

/** A line typed in the chatbox (ADR 0010): echoed, then answered here, or sent as a teleport, on any map (ADR 0012). */
export function answerTypedLine(text: string, deps: TypedLineDeps): void {
  const log = deps.log ?? silent;
  deps.addLine({ kind: 'you', text });
  const parsed = parseLine(text);
  if (!parsed) return;
  if ('answer' in parsed) {
    log('command refused here', { text, answer: parsed.answer });
    deps.addLine({ kind: 'error', text: parsed.answer });
    return;
  }
  log('teleport sent', parsed.command);
  deps.teleport(parsed.command.map, parsed.command.to);
}

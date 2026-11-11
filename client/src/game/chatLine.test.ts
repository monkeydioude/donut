import { describe, expect, it } from 'vitest';
import type { Cell } from '../net/protocol';
import { answerTypedLine } from './chatLine';

// ADR 0010 test 10, at the game's level since ADR 0012: the chatbox stays across maps
describe('answerTypedLine', () => {
  it('a teleport line is echoed and sent; the others are answered here', () => {
    const lines: string[] = [];
    const sent: { map: string; to: Cell }[] = [];
    const deps = { addLine: (line: { kind: string; text: string }) => lines.push(`${line.kind}: ${line.text}`), teleport: (map: string, to: Cell) => sent.push({ map, to }) };
    answerTypedLine('/teleport doma-1 16 16', deps);
    answerTypedLine('hello', deps);
    answerTypedLine('/fly', deps);
    expect(sent).toEqual([{ map: 'doma-1', to: { col: 16, row: 16 } }]);
    expect(lines).toEqual([
      'you: /teleport doma-1 16 16',
      'you: hello',
      'error: Chat comes later: only /teleport <map> <col> <row> works for now',
      'you: /fly',
      'error: Unknown command: /fly',
    ]);
  });
});

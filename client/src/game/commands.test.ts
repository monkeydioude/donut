import { describe, expect, it } from 'vitest';
import { CHAT_COMES_LATER, commandsMatching, parseLine, TELEPORT_USAGE, usageOf } from './commands';

// ADR 0010 test 10
describe('parseLine', () => {
  it('parses a teleport to a map and a cell, spaces around included', () => {
    const command = { type: 'teleport', map: 'world', to: { col: 1023, row: 511 } };
    expect(parseLine('/teleport world 1023 511')).toEqual({ command });
    expect(parseLine('  /teleport   world 1023   511 ')).toEqual({ command });
  });

  it('sends a negative cell as is: the server says it is outside', () => {
    expect(parseLine('/teleport world -1 2')).toEqual({ command: { type: 'teleport', map: 'world', to: { col: -1, row: 2 } } });
  });

  it('answers plain text, an unknown command and bad arguments without the server', () => {
    expect(parseLine('hello')).toEqual({ answer: CHAT_COMES_LATER });
    expect(parseLine('/fly')).toEqual({ answer: 'Unknown command: /fly' });
    for (const line of ['/teleport', '/teleport world 1', '/teleport world a b', '/teleport world 1.5 2', '/teleport world 1 2 3']) {
      expect(parseLine(line), line).toEqual({ answer: TELEPORT_USAGE });
    }
  });

  it('an empty line is nothing', () => {
    expect(parseLine('')).toBeNull();
    expect(parseLine('   ')).toBeNull();
  });
});

describe('commandsMatching', () => {
  const commands = [
    { name: '/teleport', args: '<map> <col> <row>', does: 'teleports' },
    { name: '/tell', args: '<name> <text>', does: 'tells' },
    { name: '/who', args: '', does: 'lists' },
  ];
  const names = (line: string) => commandsMatching(line, commands).map((command) => command.name);

  it('lists every command on a lone /', () => {
    expect(names('/')).toEqual(['/teleport', '/tell', '/who']);
  });

  it('filters them by the name typed so far, spaces before ignored', () => {
    expect(names('/te')).toEqual(['/teleport', '/tell']);
    expect(names('/tel')).toEqual(['/teleport', '/tell']);
    expect(names('/tele')).toEqual(['/teleport']);
    expect(names('  /w')).toEqual(['/who']);
    expect(names('/fly')).toEqual([]);
    expect(names('/Te'), 'names are case sensitive, like parseLine').toEqual([]);
  });

  it('keeps only the command named once a space follows its name', () => {
    expect(names('/tel ')).toEqual([]);
    expect(names('/tell ')).toEqual(['/tell']);
    expect(names('/teleport world 1 2')).toEqual(['/teleport']);
  });

  it('lists nothing for a line that is not a command', () => {
    expect(names('')).toEqual([]);
    expect(names('hello /teleport')).toEqual([]);
  });

  it('knows the teleport, with the usage its errors give', () => {
    const [teleport] = commandsMatching('/teleport');
    expect(teleport && `Usage: ${usageOf(teleport)}`).toBe(TELEPORT_USAGE);
  });
});

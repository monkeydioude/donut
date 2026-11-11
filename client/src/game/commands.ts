import type { Cell } from '../net/protocol';

/** A command typed in the chatbox (ADR 0010). Only the teleport exists. */
export type ChatCommand = { type: 'teleport'; map: string; to: Cell };

/** A typed line, parsed: a command for the server, or an answer to show without asking it. */
export type ParsedLine = { command: ChatCommand } | { answer: string };

/** A command the chatbox knows: its name, its arguments and what it does, listed while a command is typed. */
export interface CommandHelp {
  name: string;
  args: string;
  does: string;
}

const TELEPORT: CommandHelp = { name: '/teleport', args: '<map> <col> <row>', does: 'takes you to a cell of a map' };

/** Every command, in the order they're listed. */
export const CHAT_COMMANDS: readonly CommandHelp[] = [TELEPORT];

export const TELEPORT_USAGE = `Usage: ${usageOf(TELEPORT)}`;
export const CHAT_COMES_LATER = `Chat comes later: only ${usageOf(TELEPORT)} works for now`;

/** A command as it's typed: its name, then its arguments. */
export function usageOf(command: CommandHelp): string {
  return `${command.name} ${command.args}`;
}

/**
 * The commands to list while a line is typed: while the command's name is typed, those it starts; once a space
 * follows the name, the one it names. Nothing for a line that isn't a command. Spaces before are ignored.
 */
export function commandsMatching(line: string, commands: readonly CommandHelp[] = CHAT_COMMANDS): readonly CommandHelp[] {
  const typed = line.trimStart();
  if (!typed.startsWith('/')) return [];
  const name = typed.split(/\s/, 1)[0] ?? '';
  const named = typed.length > name.length;
  return commands.filter((command) => (named ? command.name === name : command.name.startsWith(name)));
}

/** Parses a line: spaces around are ignored, and an empty line is nothing. A command starts with `/`. */
export function parseLine(line: string): ParsedLine | null {
  const words = line.trim().split(/\s+/).filter((word) => word !== '');
  const [name, ...args] = words;
  if (name === undefined) return null;
  if (!name.startsWith('/')) return { answer: CHAT_COMES_LATER };
  if (name !== TELEPORT.name) return { answer: `Unknown command: ${name}` };
  return parseTeleport(args);
}

/** `/teleport <map> <col> <row>`: a map name and two whole numbers, which may be negative: the server says they're outside. */
function parseTeleport(args: readonly string[]): ParsedLine {
  const [map, col, row] = args;
  if (args.length !== 3 || map === undefined || !isWholeNumber(col) || !isWholeNumber(row)) return { answer: TELEPORT_USAGE };
  return { command: { type: 'teleport', map, to: { col: Number(col), row: Number(row) } } };
}

function isWholeNumber(word: string | undefined): word is string {
  return word !== undefined && /^-?\d+$/.test(word);
}

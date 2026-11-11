import type { MapDescription } from '../map/worldMap';
import type { Cell, ServerMessage } from '../net/protocol';

/** A line of the chatbox: what you typed, the log of what happens in view, a warning, or an error. */
export type LineKind = 'you' | 'log' | 'warning' | 'error';

export interface LogLine {
  kind: LineKind;
  text: string;
}

/** A walk is one line: a step this long after the last one starts a new walk. */
export const WALK_PAUSE_MS = 1000;

/** When each player in view last stepped, by id: to log a walk once, not each step. */
export interface GameLogState {
  lastStepAt: ReadonlyMap<string, number>;
}

export function initialGameLog(): GameLogState {
  return { lastStepAt: new Map() };
}

export interface LogContext {
  /** Your id, once welcomed. */
  you: string | null;
  /** The name of a player in view, before the message is applied: `left` only tells the id. */
  nameOf: (id: string) => string | null;
  /** The map's slug, named in the teleport line. */
  map: string;
  now: number;
}

/** The lines a server message adds to the log (ADR 0010), and the state after it. Most messages add none. */
export function logMessage(state: GameLogState, message: ServerMessage, context: LogContext): { state: GameLogState; lines: LogLine[] } {
  switch (message.type) {
    case 'joined':
      return message.player.id === context.you ? { state, lines: [] } : { state, lines: [log(`${message.player.name} entered the area`)] };
    case 'left':
      return { state: without(state, message.id), lines: [log(`${nameOr(context, message.id)} left the area`)] };
    case 'moved':
      return message.id === context.you ? { state, lines: [] } : logStep(state, message.id, context);
    case 'stopped':
      return { state, lines: [log(`You stopped at ${at(message)}`)] };
    case 'teleported':
      return { state, lines: teleportedLines(message, context.map) };
    case 'refused':
      return { state, lines: [{ kind: 'error', text: message.reason }] };
    default:
      return { state, lines: [] };
  }
}

/** The lines of a move between maps (ADR 0012): into a city, "You enter Doma-1"; out of one, "You leave Doma-1". */
export function logMapSwitch(from: MapDescription, to: MapDescription): LogLine[] {
  if (!to.wraps) return [log(`You enter ${to.name}`)];
  if (!from.wraps) return [log(`You leave ${from.name}`)];
  return [];
}

/** The line for choosing Move toward a cell. */
export function logMove(to: Cell): LogLine {
  return log(`You head to ${at(to)}`);
}

/** A step is a line when it starts a walk: the first one, or one after a pause. */
function logStep(state: GameLogState, id: string, context: LogContext): { state: GameLogState; lines: LogLine[] } {
  const last = state.lastStepAt.get(id);
  const lastStepAt = new Map(state.lastStepAt);
  lastStepAt.set(id, context.now);
  const starts = last === undefined || context.now - last >= WALK_PAUSE_MS;
  return { state: { lastStepAt }, lines: starts ? [log(`${nameOr(context, id)} is walking`)] : [] };
}

function teleportedLines(message: { col: number; row: number; walkable: boolean }, map: string): LogLine[] {
  const lines = [log(`Teleported to ${map} ${at(message)}`)];
  if (!message.walkable) lines.push({ kind: 'warning', text: `${at(message)} isn't walkable: teleport again to leave` });
  return lines;
}

function without(state: GameLogState, id: string): GameLogState {
  if (!state.lastStepAt.has(id)) return state;
  const lastStepAt = new Map(state.lastStepAt);
  lastStepAt.delete(id);
  return { lastStepAt };
}

function nameOr(context: LogContext, id: string): string {
  return context.nameOf(id) ?? 'Someone';
}

function at(cell: Cell): string {
  return `(${cell.col}, ${cell.row})`;
}

function log(text: string): LogLine {
  return { kind: 'log', text };
}

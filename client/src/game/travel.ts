import type { MapDescription } from '../map/worldMap';
import type { ServerMessage } from '../net/protocol';

/**
 * How the game goes from map to map on its one socket (ADR 0012): the map shown, and while the screen fades out to
 * another, the messages waiting for it, a welcome first.
 */
export interface TravelState {
  map: MapDescription | null;
  waiting: readonly ServerMessage[] | null;
}

/** What to do with a message: hand it to the map shown, build the first map, fade out to another, or keep it. */
export type TravelStep =
  | { kind: 'apply' }
  | { kind: 'first'; map: MapDescription }
  | { kind: 'leave'; from: MapDescription; to: MapDescription }
  | { kind: 'wait' };

export function initialTravel(): TravelState {
  return { map: null, waiting: null };
}

/**
 * A message from the server. The first welcome builds its map; a welcome for another map starts the fade out, and
 * every message until the screen is dark waits; a welcome for the map shown is applied to it, as before.
 */
export function receive(state: TravelState, message: ServerMessage): { state: TravelState; step: TravelStep } {
  if (state.waiting) return { state: { ...state, waiting: [...state.waiting, message] }, step: { kind: 'wait' } };
  if (!state.map) {
    return message.type === 'welcome' ? { state: { map: message.map, waiting: null }, step: { kind: 'first', map: message.map } } : { state, step: { kind: 'wait' } };
  }
  if (message.type !== 'welcome' || message.map.slug === state.map.slug) return { state, step: { kind: 'apply' } };
  return { state: { ...state, waiting: [message] }, step: { kind: 'leave', from: state.map, to: message.map } };
}

/**
 * The screen is dark: the map to build, and the messages to apply to it, in order. When another welcome came while
 * fading, as for two teleports in a row, the last one wins: the messages before it were for a map gone already.
 */
export function arrive(state: TravelState): { state: TravelState; map: MapDescription; messages: readonly ServerMessage[] } | null {
  const waiting = state.waiting ?? [];
  let last = waiting.length - 1;
  while (last >= 0 && waiting[last]?.type !== 'welcome') last--;
  const welcome = waiting[last];
  if (!welcome || welcome.type !== 'welcome') return null;
  return { state: { map: welcome.map, waiting: null }, map: welcome.map, messages: waiting.slice(last) };
}

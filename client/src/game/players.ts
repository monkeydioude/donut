import type { PlayerInfo, ServerMessage } from '../net/protocol';

/** The players on the map, as the server told them. */
export interface PlayersState {
  /** Your player's id, once welcomed. */
  you: string | null;
  players: ReadonlyMap<string, PlayerInfo>;
  /** The time of one step on the server, from the welcome. */
  stepMs: number;
}

/** Before the welcome. stepMs is a placeholder until the server tells it. */
export function emptyPlayers(): PlayersState {
  return { you: null, players: new Map(), stepMs: 200 };
}

/** The state after a server message. The previous state is left untouched. Messages about paths and chunks are ignored. */
export function applyMessage(state: PlayersState, message: ServerMessage): PlayersState {
  switch (message.type) {
    case 'welcome':
      return { you: message.you, players: new Map(message.players.map((p) => [p.id, p])), stepMs: message.stepMs };
    case 'joined':
      return withPlayers(state, (players) => players.set(message.player.id, message.player));
    case 'moved':
      return moveTo(state, message.id, message.col, message.row);
    case 'left':
      return withPlayers(state, (players) => players.delete(message.id));
    case 'stopped':
    case 'teleported':
      return state.you ? moveTo(state, state.you, message.col, message.row) : state;
    case 'planned':
    case 'walking':
    case 'refused':
    case 'chunk':
    case 'forget':
      return state;
  }
}

export function ownPlayer(state: PlayersState): PlayerInfo | null {
  return (state.you && state.players.get(state.you)) || null;
}

function moveTo(state: PlayersState, id: string, col: number, row: number): PlayersState {
  const player = state.players.get(id);
  if (!player) return state;
  return withPlayers(state, (players) => players.set(id, { ...player, col, row }));
}

function withPlayers(state: PlayersState, change: (players: Map<string, PlayerInfo>) => void): PlayersState {
  const players = new Map(state.players);
  change(players);
  return { ...state, players };
}

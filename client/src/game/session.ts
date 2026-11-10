import type { TilePoint } from '../iso/projection';
import { emptyChunks, isHeld, regionAt, withChunk, withoutChunk, type HeldChunks } from '../map/chunks';
import { regionName, type MapDescription } from '../map/worldMap';
import type { GameSocket, GameSocketHandlers } from '../net/gameSocket';
import type { ServerMessage } from '../net/protocol';
import type { ChunksView } from '../render/chunkView';
import type { OverlayView } from '../render/overlay';
import type { PlayersView } from '../render/playersView';
import type { RegionBanner } from '../ui/banner';
import type { PointerHandlers } from '../ui/pointer';
import { silent, type Log } from '../debug/log';
import { enterRegion, initialBanner } from './banner';
import { applyCommand, canMove, initialCommand, shownPath, type CommandEvent, type CommandState } from './command';
import { applyMessage, emptyPlayers, ownPlayer, type PlayersState } from './players';

/** The action menu, as the session drives it. */
export interface CommandMenu {
  show(cell: TilePoint, moveEnabled: boolean): void;
  hide(): void;
}

export interface MenuChoices {
  onMove(): void;
  onCancel(): void;
}

export interface PlayersSessionDeps {
  map: MapDescription;
  socketUrl: string;
  openSocket: (url: string, handlers: GameSocketHandlers) => GameSocket;
  /** Calls the handlers on hovers, clicks and cancels; returns a function that stops. */
  listenToPointer: (handlers: PointerHandlers) => () => void;
  createMenu: (choices: MenuChoices) => CommandMenu;
  players: PlayersView;
  chunks: ChunksView;
  overlay: OverlayView;
  banner: RegionBanner;
  onDisconnect: () => void;
  now?: () => number;
  log?: Log;
}

export interface PlayersSession {
  /** Redraws the borders, path and menu: after the view moved, for instance. */
  refresh(): void;
  /** Keeps the menu beside its tile while the camera moves: every frame. */
  followCamera(): void;
  leave(): void;
}

/**
 * Plays on a map: point at a tile, click it, choose Move, and the server walks your player there.
 * Everything shown comes from the server: the chunks around you, players, paths and walks.
 */
export function startPlayersSession(deps: PlayersSessionDeps): PlayersSession {
  const log = deps.log ?? silent;
  const now = deps.now ?? (() => performance.now());
  let players = emptyPlayers();
  let command = initialCommand();
  let held: HeldChunks = emptyChunks();
  let banner = initialBanner();

  const applyChunks = (message: ServerMessage) => {
    if (message.type === 'chunk') {
      held = withChunk(held, message.chunk);
      deps.chunks.show(message.chunk);
    } else if (message.type === 'forget') {
      held = withoutChunk(held, message);
      deps.chunks.forget(message);
    } else return;
    log(`chunks after ${message.type}`, { held: held.size });
  };
  // the region banner, once your cell's region is known and each time it changes
  const updateBanner = () => {
    const own = ownPlayer(players);
    const next = enterRegion(banner, own ? regionAt(held, own) : null, now());
    banner = next.state;
    if (next.show === null) return;
    const name = regionName(deps.map, next.show);
    log('region entered', { region: next.show, name });
    if (name) deps.banner.show(name);
  };

  const render = () => {
    deps.overlay.showHover(command.hover);
    deps.overlay.showSelection(command.selection?.cell ?? null);
    deps.overlay.showPath(shownPath(command));
    if (command.selection) menu.show(command.selection.cell, canMove(command));
    else menu.hide();
  };
  const dispatch = (event: CommandEvent) => {
    command = applyCommand(command, event);
    if (event.type !== 'hover') log(`command: ${event.type}`, { event, state: summary(command) });
    render();
  };

  const socket = deps.openSocket(deps.socketUrl, {
    onMessage: (message) => {
      applyChunks(message);
      const before = players;
      players = applyMessage(players, message);
      if (players !== before) {
        log(`players after ${message.type}`, { you: players.you, count: players.players.size, players: [...players.players.values()] });
        deps.players.sync(players, message.type === 'moved' ? 'glide' : 'place');
        followYou(before, players);
      }
      updateBanner();
      command = applyCommand(command, { type: 'server', message, you: players.you });
      if (message.type !== 'moved' || message.id === players.you) log(`command after ${message.type}`, summary(command));
      // your step makes the selected tile's planned path start from the wrong cell: ask again
      if (message.type === 'moved' && message.id === players.you && command.selection) {
        log('you stepped while a tile is selected: asking its path again', command.selection.cell);
        socket.plan(command.selection.cell);
      }
      render();
    },
    onClose: () => {
      log('disconnected');
      deps.onDisconnect();
    },
  });
  const menu = deps.createMenu({
    onMove: () => {
      if (!command.selection) return;
      log('menu: Move', { to: command.selection.cell, path: command.selection.path });
      socket.walk(command.selection.cell);
      dispatch({ type: 'move' });
    },
    onCancel: () => {
      log('menu: Cancel');
      dispatch({ type: 'cancel' });
    },
  });
  // only the cells of chunks you hold can be pointed at: the others aren't drawn
  const stopPointer = deps.listenToPointer({
    hover: (cell) => dispatch({ type: 'hover', cell: cell && isHeld(held, cell) ? cell : null }),
    select: (cell) => {
      if (!isHeld(held, cell)) {
        log('click on a chunk not held: cancel', cell);
        dispatch({ type: 'cancel' });
        return;
      }
      dispatch({ type: 'select', cell });
      log('asking the path to', cell);
      socket.plan(cell);
    },
    cancel: () => dispatch({ type: 'cancel' }),
  });
  // your drawn cell moved: the chunks follow, to stay at their copy nearest to you
  function followYou(before: PlayersState, after: PlayersState) {
    const was = ownPlayer(before);
    const is = ownPlayer(after);
    const anchor = deps.players.ownCell();
    if (is && anchor && (!was || was.col !== is.col || was.row !== is.row)) deps.chunks.follow(anchor);
  }

  return {
    refresh: render,
    followCamera: () => {
      if (command.selection) menu.show(command.selection.cell, canMove(command));
    },
    leave: () => {
      log('leaving');
      stopPointer();
      socket.close();
    },
  };
}

/** The command state in a few words, for the logs. */
function summary(command: CommandState) {
  return {
    hover: command.hover,
    selected: command.selection?.cell ?? null,
    planned: command.selection?.path === null ? 'waiting' : (command.selection?.path?.length ?? null),
    moveEnabled: canMove(command),
    walkingSteps: command.walking.length,
  };
}

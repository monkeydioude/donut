import type { TilePoint } from '../iso/projection';
import { emptyChunks, isHeld, regionAt, withChunk, withoutChunk, type HeldChunks, type HeldChunksRef } from '../map/chunks';
import { regionName, type MapDescription } from '../map/worldMap';
import type { GameSocket, GameSocketHandlers } from '../net/gameSocket';
import type { ServerMessage } from '../net/protocol';
import type { ChunksView } from '../render/chunkView';
import type { OverlayView } from '../render/overlay';
import type { PlayersView } from '../render/playersView';
import type { RegionBanner } from '../ui/banner';
import type { PointerHandlers, PointerListener } from '../ui/pointer';
import { silent, type Log } from '../debug/log';
import { enterRegion, initialBanner } from './banner';
import { applyCommand, canMove, initialCommand, shownPath, type CommandEvent, type CommandState } from './command';
import { parseLine } from './commands';
import { initialGameLog, logMessage, logMove, type LogLine } from './gameLog';
import { applyMessage, emptyPlayers, ownPlayer, type PlayersState } from './players';

/** The chatbox, as the session drives it (ADR 0010). */
export interface SessionChatbox {
  addLine(line: LogLine): void;
  focus(): void;
}

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
  /** Calls the handlers on hovers, clicks and cancels. */
  listenToPointer: (handlers: PointerHandlers) => PointerListener;
  createMenu: (choices: MenuChoices) => CommandMenu;
  /** The chatbox, told what to do with a typed line. */
  createChatbox: (onLine: (text: string) => void) => SessionChatbox;
  players: PlayersView;
  chunks: ChunksView;
  overlay: OverlayView;
  banner: RegionBanner;
  /** Where the session keeps the chunks held, for the views drawing on heights to read. */
  held?: HeldChunksRef;
  onDisconnect: () => void;
  now?: () => number;
  log?: Log;
}

/** What the map shows of the session (ADR 0005): your cell, the other players you see, your region. */
export interface Whereabouts {
  you: TilePoint | null;
  others: TilePoint[];
  /** Your cell's region id, 0 for none; null before your chunk comes. */
  region: number | null;
}

export interface PlayersSession {
  /** Redraws the borders, path and menu: after the view moved, for instance. */
  refresh(): void;
  /** Keeps the hover under the pointer and the menu beside its tile while the camera moves: every frame. */
  followCamera(): void;
  /** Cancels the selection, as the right click does. */
  cancel(): void;
  /** Gives the chatbox the focus (ADR 0010). */
  openChat(): void;
  whereabouts(): Whereabouts;
  /** The cell under the pointer, within the map; null off the chunks you hold. */
  hovered(): TilePoint | null;
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
  const heldRef: HeldChunksRef = deps.held ?? { current: emptyChunks() };
  let banner = initialBanner();
  let gameLog = initialGameLog();

  // a typed line: answered here, or sent as a teleport, echoed either way (ADR 0010)
  const chatbox = deps.createChatbox((text) => {
    chatbox.addLine({ kind: 'you', text });
    const parsed = parseLine(text);
    if (!parsed) return;
    if ('answer' in parsed) {
      log('command refused here', { text, answer: parsed.answer });
      chatbox.addLine({ kind: 'error', text: parsed.answer });
      return;
    }
    log('teleport sent', parsed.command);
    socket.teleport(parsed.command.map, parsed.command.to);
  });
  // the log of what a message tells, named from the players as they were before it: `left` only tells an id
  const logLines = (message: ServerMessage, before: PlayersState) => {
    const nameOf = (id: string) => before.players.get(id)?.name ?? null;
    const next = logMessage(gameLog, message, { you: before.you, nameOf, map: deps.map.slug, now: now() });
    gameLog = next.state;
    next.lines.forEach((line) => chatbox.addLine(line));
  };

  const applyChunks = (message: ServerMessage) => {
    if (message.type === 'chunk') {
      heldRef.current = withChunk(heldRef.current, message.chunk);
      deps.chunks.show(message.chunk);
      // players drawn before their chunk came stand at height 0 until now
      deps.players.relift();
    } else if (message.type === 'forget') {
      heldRef.current = withoutChunk(heldRef.current, message);
      deps.chunks.forget(message);
    } else return;
    log(`chunks after ${message.type}`, { held: heldRef.current.size });
  };
  // the region banner, once your cell's region is known and each time it changes
  const updateBanner = () => {
    const own = ownPlayer(players);
    const next = enterRegion(banner, own ? regionAt(heldRef.current, own) : null, now());
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
      logLines(message, before);
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
      chatbox.addLine(logMove(command.selection.cell));
      dispatch({ type: 'move' });
    },
    onCancel: () => {
      log('menu: Cancel');
      dispatch({ type: 'cancel' });
    },
  });
  // only the cells of chunks you hold can be pointed at: the others aren't drawn
  const pointer = deps.listenToPointer({
    hover: (cell) => dispatch({ type: 'hover', cell: cell && isHeld(heldRef.current, cell) ? cell : null }),
    select: (cell) => {
      if (!isHeld(heldRef.current, cell)) {
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
      pointer.repick();
      if (command.selection) menu.show(command.selection.cell, canMove(command));
    },
    cancel: () => dispatch({ type: 'cancel' }),
    openChat: () => chatbox.focus(),
    whereabouts: () => whereaboutsOf(players, heldRef.current),
    hovered: () => command.hover,
    leave: () => {
      log('leaving');
      pointer.stop();
      socket.close();
    },
  };
}

/** Your cell and region, and the cells of the other players you see: those on the chunks you hold. */
export function whereaboutsOf(players: PlayersState, held: HeldChunks): Whereabouts {
  const own = ownPlayer(players);
  const others = [...players.players.values()].filter((p) => p.id !== players.you).map(({ col, row }) => ({ col, row }));
  return { you: own && { col: own.col, row: own.row }, others, region: own ? regionAt(held, own) : null };
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

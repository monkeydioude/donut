import type { TilePoint } from '../iso/projection';
import { emptyChunks, isHeld, regionAt, withChunk, withoutChunk, type HeldChunks, type HeldChunksRef } from '../map/chunks';
import { gridOf, regionName, type MapDescription } from '../map/worldMap';
import type { GameSocket } from '../net/gameSocket';
import type { CellAction, ServerMessage } from '../net/protocol';
import type { ChunksView } from '../render/chunkView';
import type { OverlayView } from '../render/overlay';
import type { PlayersView } from '../render/playersView';
import type { RegionBanner } from '../ui/banner';
import type { PointerHandlers, PointerListener } from '../ui/pointer';
import { silent, type Log } from '../debug/log';
import { actionAt } from './actions';
import { enterRegion, initialBanner } from './banner';
import { applyCommand, canAct, initialCommand, shownPath, type CommandEvent, type CommandState } from './command';
import { initialGameLog, logMessage, logMove, type LogLine } from './gameLog';
import { applyMessage, emptyPlayers, ownPlayer, type PlayersState } from './players';

/** The chatbox, as the session writes in it (ADR 0010): the game's, which stays across maps. */
export interface SessionChatbox {
  addLine(line: LogLine): void;
}

/** The action menu, as the session drives it: Move, Enter or Exit, then Cancel (ADR 0012). */
export interface CommandMenu {
  show(cell: TilePoint, action: CellAction, enabled: boolean): void;
  hide(): void;
}

export interface MenuChoices {
  /** The menu's action was chosen. */
  onAct(): void;
  onCancel(): void;
}

export interface PlayersSessionDeps {
  map: MapDescription;
  /** The game's socket: it stays open across maps, and the game hands this session the messages of its map. */
  socket: GameSocket;
  /** Calls the handlers on hovers, clicks and cancels. */
  listenToPointer: (handlers: PointerHandlers) => PointerListener;
  createMenu: (choices: MenuChoices) => CommandMenu;
  chatbox: SessionChatbox;
  players: PlayersView;
  chunks: ChunksView;
  overlay: OverlayView;
  banner: RegionBanner;
  /** Where the session keeps the chunks held, for the views drawing on heights to read. */
  held?: HeldChunksRef;
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
  /** A message of the server about this map. */
  receive(message: ServerMessage): void;
  /** Redraws the borders, path and menu: after the view moved, for instance. */
  refresh(): void;
  /** Keeps the hover under the pointer and the menu beside its tile while the camera moves: every frame. */
  followCamera(): void;
  /** Cancels the selection, as the right click does. */
  cancel(): void;
  whereabouts(): Whereabouts;
  /** The cell under the pointer, within the map; null off the chunks you hold. */
  hovered(): TilePoint | null;
  /** Stops listening to the pointer and hides the menu: the game moved to another map (ADR 0012). */
  stop(): void;
}

/**
 * Plays on a map: point at a tile, click it, choose Move (or Enter, or Exit), and the server walks your player there.
 * Everything shown comes from the server: the chunks around you, players, paths and walks.
 */
export function startPlayersSession(deps: PlayersSessionDeps): PlayersSession {
  const log = deps.log ?? silent;
  const now = deps.now ?? (() => performance.now());
  const { socket, chatbox } = deps;
  const grid = gridOf(deps.map);
  let players = emptyPlayers();
  let command = initialCommand();
  const heldRef: HeldChunksRef = deps.held ?? { current: emptyChunks() };
  let banner = initialBanner();
  let gameLog = initialGameLog();

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
    if (command.selection) menu.show(command.selection.cell, command.selection.action, canAct(command));
    else menu.hide();
  };
  const dispatch = (event: CommandEvent) => {
    command = applyCommand(command, event);
    if (event.type !== 'hover') log(`command: ${event.type}`, { event, state: summary(command) });
    render();
  };

  const receive = (message: ServerMessage) => {
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
  };
  const menu = deps.createMenu({
    onAct: () => {
      const selection = command.selection;
      if (!selection) return;
      log(`menu: ${selection.action}`, { to: selection.cell, path: selection.path });
      act(selection.action, selection.cell);
      dispatch({ type: 'act' });
    },
    onCancel: () => {
      log('menu: Cancel');
      dispatch({ type: 'cancel' });
    },
  });
  // a Move walks there; Enter and Exit walk to the door, then through (ADR 0012)
  const act = (action: CellAction, cell: TilePoint) => {
    if (action === 'enter') socket.enter(cell);
    else if (action === 'exit') socket.exit(cell);
    else {
      socket.walk(cell);
      chatbox.addLine(logMove(cell));
    }
  };
  // only the cells of chunks you hold can be pointed at: the others aren't drawn. Leaving a city is the exception: a
  // click past its edge is on no chunk.
  const pointer = deps.listenToPointer({
    hover: (cell) => dispatch({ type: 'hover', cell: cell && isHeld(heldRef.current, cell) ? cell : null }),
    select: (cell) => {
      const action = actionAt(cell, heldRef.current, grid);
      if (action !== 'exit' && !isHeld(heldRef.current, cell)) {
        log('click on a chunk not held: cancel', cell);
        dispatch({ type: 'cancel' });
        return;
      }
      dispatch({ type: 'select', cell, action });
      log('asking the path to', { cell, action });
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
    receive,
    refresh: render,
    followCamera: () => {
      pointer.repick();
      if (command.selection) menu.show(command.selection.cell, command.selection.action, canAct(command));
    },
    cancel: () => dispatch({ type: 'cancel' }),
    whereabouts: () => whereaboutsOf(players, heldRef.current),
    hovered: () => command.hover,
    stop: () => {
      log('stopped: the game left this map');
      pointer.stop();
      menu.hide();
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
    action: command.selection?.action ?? null,
    enabled: canAct(command),
    walkingSteps: command.walking.length,
  };
}

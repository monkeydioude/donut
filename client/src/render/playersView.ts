import { Container, Sprite, type Texture } from 'pixi.js';
import { silent, type Log } from '../debug/log';
import type { PlayersState } from '../game/players';
import { groundCenterAt, type HeightOf, type Heights } from '../iso/heights';
import type { ScreenPoint, TilePoint } from '../iso/projection';
import { nearestCopy, type Grid } from '../map/grid';
import type { PlayerInfo } from '../net/protocol';
import { drawnTarget, glideAt, placePlayer, retarget, shiftGlide, type Arrival, type Glide } from './glide';
import { placeOnCell } from './mapView';
import type { SpriteGraphic } from './tileGraphics';

export interface PlayersView {
  /** Shows the players of the state: new ones appear, gone ones disappear, the others glide or jump to their cell. */
  sync(state: PlayersState, arrival: Arrival): void;
  /** Where your player is drawn right now, mid-glide or not. Null before the welcome. */
  ownPosition(): ScreenPoint | null;
  /**
   * The cell your player is drawn on, in coordinates that don't wrap: it keeps counting past the seam (1023, 1024…).
   * Everything else is drawn at its copy nearest to it. Null before the welcome.
   */
  ownCell(): TilePoint | null;
  /** Puts every player at its cell's height again: a chunk came in, with heights that weren't known. */
  relift(): void;
}

/** Something that calls back every frame, like Pixi's ticker. */
export interface FrameTicker {
  add(onFrame: () => void): unknown;
}

/** What a player is drawn with (ADR 0006). */
export interface PlayerGraphics {
  shadow: SpriteGraphic;
  /** Under your own robot only. */
  ring: SpriteGraphic;
  robotOf(slug: string): Texture;
}

interface ShownPlayer {
  /** The player's layers: your ring, the shadow, the robot. */
  sprite: Container;
  glide: Glide;
  /** The player's cell within the map, as the server told it. */
  cell: TilePoint;
  /** The cell the player is drawn on. */
  drawn: TilePoint;
}

export interface PlayersViewDeps {
  layer: Container;
  heights: Heights;
  /** A cell's height, for a cell within the map: 0 when it isn't known. */
  heightAt: HeightOf;
  graphics: PlayerGraphics;
  ticker: FrameTicker;
  grid: Grid;
  now?: () => number;
  log?: Log;
}

export function createPlayersView(deps: PlayersViewDeps): PlayersView {
  const { layer, heights, heightAt, graphics, ticker, grid, now = () => performance.now(), log = silent } = deps;
  const { projection } = heights;
  const shown = new Map<string, ShownPlayer>();
  let you: string | null = null;
  ticker.add(() => {
    const t = now();
    for (const { sprite, glide } of shown.values()) {
      const at = glideAt(glide, t);
      sprite.position.set(at.x, at.y);
    }
  });

  const ownCell = () => (you ? (shown.get(you)?.drawn ?? null) : null);

  const remove = (id: string, player: ShownPlayer) => {
    log('sprite removed', { id, cell: player.cell });
    player.sprite.destroy({ children: true });
    shown.delete(id);
  };

  /** Moves a player who didn't step to its copy nearest to you, when you walked far enough for it to change. */
  const keepNearYou = (id: string, player: ShownPlayer) => {
    const anchor = ownCell();
    if (!anchor || id === you) return;
    const drawn = nearestCopy(grid, player.cell, anchor);
    if (drawn.col === player.drawn.col && drawn.row === player.drawn.row) return;
    const from = projection.toScreen(player.drawn);
    const to = projection.toScreen(drawn);
    log('sprite moved to its nearest copy', { id, from: player.drawn, to: drawn });
    player.glide = shiftGlide(player.glide, { x: to.x - from.x, y: to.y - from.y });
    player.drawn = drawn;
    player.sprite.zIndex = projection.depth(drawn);
  };

  const show = (player: PlayerInfo, state: PlayersState, arrival: Arrival) => {
    const current = shown.get(player.id);
    if (current && current.cell.col === player.col && current.cell.row === player.row) {
      keepNearYou(player.id, current);
      return;
    }
    const how = current ? arrival : 'place';
    const target = drawnTarget(grid, player, current?.drawn, how, ownCell());
    const sprite = current?.sprite ?? newFigure(player, player.id === state.you, target, heightAt(player));
    const placed = placePlayer(current?.glide, target, heightAt(player), how, now(), state.stepMs, heights);
    log(current ? `sprite ${how === 'glide' ? 'glides' : 'jumps'}` : 'sprite added', {
      id: player.id,
      you: player.id === state.you,
      from: current?.drawn ?? null,
      to: target,
      cell: { col: player.col, row: player.row },
      ms: placed.glide.durationMs,
    });
    if (player.id === state.you && current && !sameCopy(current.drawn, current.cell, target, player)) {
      log('your drawn cell crossed a seam', { drawn: target, cell: { col: player.col, row: player.row } });
    }
    sprite.zIndex = placed.zIndex;
    shown.set(player.id, { sprite, glide: placed.glide, cell: { col: player.col, row: player.row }, drawn: target });
  };

  return {
    ownPosition() {
      const glide = you ? shown.get(you)?.glide : undefined;
      return glide ? glideAt(glide, now()) : null;
    },
    ownCell,
    relift() {
      const t = now();
      for (const player of shown.values()) player.glide = retarget(player.glide, groundCenterAt(heights, player.drawn, heightAt(player.cell)), t);
    },
    sync(state, arrival) {
      you = state.you;
      for (const [id, player] of shown) if (!state.players.has(id)) remove(id, player);
      // you first: the others are drawn near where you are drawn
      const own = state.you ? state.players.get(state.you) : undefined;
      if (own) show(own, state, arrival);
      for (const player of state.players.values()) if (player.id !== state.you) show(player, state, arrival);
    },
  };

  /** Both drawn cells are in the same copy of the world: drawn minus cell is the same. */
  function sameCopy(drawnA: TilePoint, cellA: TilePoint, drawnB: TilePoint, cellB: TilePoint): boolean {
    return drawnA.col - cellA.col === drawnB.col - cellB.col && drawnA.row - cellA.row === drawnB.row - cellB.row;
  }

  function newFigure(player: PlayerInfo, isYou: boolean, cell: TilePoint, height: number): Container {
    const figure = playerFigure(graphics, player.robot, isYou);
    placeOnCell(figure, cell, heights, height);
    layer.addChild(figure);
    return figure;
  }
}

/** A player's layers, bottom to top: your ring (yours only), the shadow, the robot standing on the ground center. */
function playerFigure(graphics: PlayerGraphics, robot: string, isYou: boolean): Container {
  const figure = new Container();
  if (isYou) figure.addChild(markSprite(graphics.ring));
  figure.addChild(markSprite(graphics.shadow));
  const body = new Sprite(graphics.robotOf(robot));
  body.anchor.set(0.5, 1);
  figure.addChild(body);
  return figure;
}

function markSprite(graphic: SpriteGraphic): Sprite {
  const sprite = new Sprite(graphic.texture);
  sprite.pivot.set(graphic.pivot.x, graphic.pivot.y);
  return sprite;
}

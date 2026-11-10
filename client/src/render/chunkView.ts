import { Container, Sprite, type Texture } from 'pixi.js';
import { pickForCell } from '../core/rng';
import { silent, type Log } from '../debug/log';
import type { IsoProjection, ScreenPoint, TilePoint } from '../iso/projection';
import type { Chunk } from '../map/chunks';
import { CHUNK_SIZE, chunkId, chunkOrigin, nearestCopy, type ChunkKey, type Grid } from '../map/grid';
import type { DrawnGroundKind } from '../map/worldMap';
import { createCellSprite, placeOnCell } from './mapView';
import type { WorldGraphics } from './tileGraphics';

/** A ground tile of a chunk: its kind, its map cell, and where it goes from the chunk's first cell. */
export interface PlacedTile {
  kind: DrawnGroundKind;
  cell: TilePoint;
  local: ScreenPoint;
  depth: number;
}

/** A texture drawn from a container, and where its top-left corner is in the container. */
export interface Baked {
  texture: Texture;
  x: number;
  y: number;
}

/** Draws a container into one texture, once, then destroys the container. Pixi's renderer does it in the app. */
export type Bake = (container: Container) => Baked;

export interface ChunksView {
  /** Draws a chunk that came in: its ground in one texture, its rocks among the entities. */
  show(chunk: Chunk): void;
  /** Destroys a forgotten chunk's texture and rocks. */
  forget(key: ChunkKey): void;
  /** Moves every chunk to its copy nearest to your drawn cell, which moved. */
  follow(anchor: TilePoint): void;
}

export interface ChunksViewDeps {
  ground: Container;
  entities: Container;
  projection: IsoProjection;
  graphics: WorldGraphics;
  grid: Grid;
  bake: Bake;
  /** Your drawn cell: chunks are drawn at their copy nearest to it. */
  anchor: () => TilePoint | null;
  now?: () => number;
  log?: Log;
}

/** Salts keep the ground and the rocks from picking the same variant on a cell. */
const GROUND_SALT = 1;
const ROCK_SALT = 2;

/**
 * Where each ground tile of a chunk goes, from the chunk's first cell, back to front. Void cells are left out: they
 * are not drawn. Pure: no Pixi involved.
 */
export function layoutChunkGround(chunk: Chunk, projection: IsoProjection): PlacedTile[] {
  const origin = chunkOrigin(chunk.key);
  const placed: PlacedTile[] = [];
  chunk.ground.forEach((kind, i) => {
    if (kind === 'void') return;
    const local = { col: i % CHUNK_SIZE, row: Math.floor(i / CHUNK_SIZE) };
    const cell = { col: origin.col + local.col, row: origin.row + local.row };
    placed.push({ kind, cell, local: projection.toScreen(local), depth: projection.depth(local) });
  });
  return placed.sort((a, b) => a.depth - b.depth);
}

/** Where a chunk's first cell is drawn: the copy of the chunk nearest to your drawn cell. */
export function drawnChunkOrigin(grid: Grid, key: ChunkKey, anchor: TilePoint): TilePoint {
  const origin = chunkOrigin(key);
  const half = CHUNK_SIZE / 2;
  const center = nearestCopy(grid, { col: origin.col + half, row: origin.row + half }, anchor);
  return { col: center.col - half, row: center.row - half };
}

interface ShownChunk {
  /** Null for a chunk with nothing to draw: all void. */
  ground: { sprite: Sprite; baked: Baked } | null;
  rocks: { sprite: Sprite; cell: TilePoint }[];
  /** Where the chunk's first cell is drawn. */
  drawnOrigin: TilePoint;
}

export function createChunksView(deps: ChunksViewDeps): ChunksView {
  const { projection, grid, log = silent, now = () => performance.now() } = deps;
  const shown = new Map<string, ShownChunk>();

  const place = (key: ChunkKey, chunk: ShownChunk, drawnOrigin: TilePoint) => {
    chunk.drawnOrigin = drawnOrigin;
    if (chunk.ground) {
      const screen = projection.toScreen(drawnOrigin);
      chunk.ground.sprite.position.set(screen.x + chunk.ground.baked.x, screen.y + chunk.ground.baked.y);
      chunk.ground.sprite.zIndex = projection.depth(drawnOrigin);
    }
    const origin = chunkOrigin(key);
    for (const rock of chunk.rocks) {
      placeOnCell(rock.sprite, { col: rock.cell.col - origin.col + drawnOrigin.col, row: rock.cell.row - origin.row + drawnOrigin.row }, projection);
    }
  };

  const remove = (id: string) => {
    const chunk = shown.get(id);
    if (!chunk) return;
    chunk.ground?.sprite.destroy({ texture: true, textureSource: true });
    for (const rock of chunk.rocks) rock.sprite.destroy();
    shown.delete(id);
  };

  return {
    show(chunk) {
      const started = now();
      const id = chunkId(chunk.key);
      remove(id); // a chunk sent again replaces the one shown
      const ground = drawGround(chunk, projection, deps.graphics, deps.bake);
      if (ground) deps.ground.addChild(ground.sprite);
      const rocks = chunk.objects.map((object) => {
        const sprite = createCellSprite(pickForCell(deps.graphics.objects[object.kind], object.col, object.row, ROCK_SALT), object, projection);
        deps.entities.addChild(sprite);
        return { sprite, cell: { col: object.col, row: object.row } };
      });
      const entry: ShownChunk = { ground, rocks, drawnOrigin: chunkOrigin(chunk.key) };
      shown.set(id, entry);
      const anchor = deps.anchor();
      place(chunk.key, entry, anchor ? drawnChunkOrigin(grid, chunk.key, anchor) : chunkOrigin(chunk.key));
      log('chunk drawn', { chunk: id, drawnOrigin: entry.drawnOrigin, rocks: rocks.length, ms: Math.round(now() - started), shown: shown.size });
    },
    forget(key) {
      const id = chunkId(key);
      log('chunk forgotten', { chunk: id, shown: shown.size - (shown.has(id) ? 1 : 0) });
      remove(id);
    },
    follow(anchor) {
      for (const [id, chunk] of shown) {
        const [cx, cy] = id.split(',').map(Number) as [number, number];
        const key = { cx, cy };
        const drawnOrigin = drawnChunkOrigin(grid, key, anchor);
        if (drawnOrigin.col !== chunk.drawnOrigin.col || drawnOrigin.row !== chunk.drawnOrigin.row) {
          log('chunk moved to its nearest copy', { chunk: id, from: chunk.drawnOrigin, to: drawnOrigin });
          place(key, chunk, drawnOrigin);
        }
      }
    },
  };
}

/** The chunk's ground tiles drawn into one texture, from its first cell. Null when every cell is void. */
function drawGround(chunk: Chunk, projection: IsoProjection, graphics: WorldGraphics, bake: Bake): ShownChunk['ground'] {
  const tiles = layoutChunkGround(chunk, projection);
  if (tiles.length === 0) return null;
  const container = new Container();
  for (const tile of tiles) {
    const graphic = pickForCell(graphics.ground[tile.kind], tile.cell.col, tile.cell.row, GROUND_SALT);
    const sprite = new Sprite(graphic.texture);
    sprite.pivot.set(graphic.pivot.x, graphic.pivot.y);
    sprite.position.set(tile.local.x, tile.local.y);
    container.addChild(sprite);
  }
  const baked = bake(container);
  return { sprite: new Sprite(baked.texture), baked };
}

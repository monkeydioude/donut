import { Container, Sprite, type Texture } from 'pixi.js';
import type { FaceMaterial, FaceSide } from '../art/cliff';
import { CORNER_STEPS, CORNERS, EDGE_STEPS, EDGES, type Corner, type Edge } from '../art/edges';
import { WATER_FRAMES } from '../art/water';
import { pickForCell } from '../core/rng';
import { silent, type Log } from '../debug/log';
import { groundCenterAt, type Heights } from '../iso/heights';
import type { ScreenPoint, TilePoint } from '../iso/projection';
import { localIndex, terrainAt, type CellTerrain, type Chunk, type HeldChunksRef } from '../map/chunks';
import { CHUNK_SIZE, chunkId, chunkOrigin, chunksBeside, nearestCopy, normalize, type ChunkKey, type Grid } from '../map/grid';
import { MIN_HEIGHT, type BakedGroundKind } from '../map/worldMap';
import { placeOnCell } from './mapView';
import type { FrameTicker } from './playersView';
import type { SpriteGraphic, WorldGraphics } from './tileGraphics';

/** A cliff face of a cell: on which side, and how many levels it goes down. */
export interface PlacedFace {
  side: FaceSide;
  levels: number;
}

/** A tile baked into a chunk's ground: its kind, its map cell, where its ground center is drawn, at its height. */
export interface PlacedTile {
  kind: BakedGroundKind;
  cell: TilePoint;
  /** From the chunk's first cell, drawn flat. */
  local: ScreenPoint;
  depth: number;
  faces: PlacedFace[];
  /** The edges toward a cell known to be of another kind: a hole's rim, a bridge's rails, the ocean's foam. */
  edges: Edge[];
  /** The edges toward a lower cell, or one that is void or not held: a grass top's rim. */
  rims: Edge[];
  /** The corners toward a lower diagonal neighbor, counted the same: the rim there joins its neighbors' rims. */
  rimCorners: Corner[];
}

/** A water cell of a chunk: drawn under the ground, with foam on its edges toward cells that aren't water. */
export interface PlacedWater {
  cell: TilePoint;
  local: ScreenPoint;
  foam: Edge[];
}

export interface ChunkLayout {
  /** Back to front. */
  tiles: PlacedTile[];
  water: PlacedWater[];
}

/** A cell around a chunk's cells, in map coordinates that may run past the map's edges: null when it isn't held. */
export type TerrainAround = (cell: TilePoint) => CellTerrain | null;

/** A texture drawn from a container, and where its top-left corner is in the container. */
export interface Baked {
  texture: Texture;
  x: number;
  y: number;
}

/** Draws a container into one texture, once, then destroys the container. Pixi's renderer does it in the app. */
export type Bake = (container: Container) => Baked;

export interface ChunksView {
  /** Draws a chunk that came in, and again the held chunks beside it: their faces and edges depend on it. */
  show(chunk: Chunk): void;
  /** Destroys a forgotten chunk's textures and rocks. */
  forget(key: ChunkKey): void;
  /** Moves every chunk to its copy nearest to your drawn cell, which moved. */
  follow(anchor: TilePoint): void;
}

export interface ChunksViewDeps {
  /** Under the ground: the water. */
  water: Container;
  ground: Container;
  entities: Container;
  heights: Heights;
  graphics: WorldGraphics;
  grid: Grid;
  /** The chunks held: a chunk is drawn from it, with its neighbors. */
  held: HeldChunksRef;
  bake: Bake;
  /** Chunks that came in are drawn on the next frame, together, and the water flows. */
  ticker: FrameTicker;
  /** Your drawn cell: chunks are drawn at their copy nearest to it. */
  anchor: () => TilePoint | null;
  waterFrameMs: number;
  now?: () => number;
  log?: Log;
}

/** Salts keep the ground and the rocks from picking the same variant on a cell. */
const GROUND_SALT = 1;
const ROCK_SALT = 2;

/** The face on each side of a cell, and the neighbor it faces: the next row on the left, the next column on the right. */
const FACE_SIDES: readonly { side: FaceSide; toward: Edge }[] = [
  { side: 'left', toward: 'sw' },
  { side: 'right', toward: 'se' },
];

/**
 * Where everything of a chunk is drawn, from its first cell. Each ground tile at its height, back to front, with a face
 * on each front side down to the neighbor's height, or to the lowest level when that neighbor is void or isn't held.
 * Water is laid out apart, for its own layer. Void cells are left out. Pure: no Pixi involved.
 */
export function layoutChunk(chunk: Chunk, heights: Heights, around: TerrainAround): ChunkLayout {
  const origin = chunkOrigin(chunk.key);
  const terrain = (cell: TilePoint) => terrainInChunk(chunk, cell) ?? around(cell);
  const tiles: PlacedTile[] = [];
  const water: PlacedWater[] = [];
  chunk.ground.forEach((kind, i) => {
    if (kind === 'void') return;
    const local = { col: i % CHUNK_SIZE, row: Math.floor(i / CHUNK_SIZE) };
    const cell = { col: origin.col + local.col, row: origin.row + local.row };
    const height = chunk.height[i] ?? 0;
    const at = groundCenterAt(heights, local, height);
    const edges = edgesToward(cell, kind, terrain);
    if (kind === 'water') water.push({ cell, local: at, foam: edges });
    else tiles.push({ kind, cell, local: at, depth: heights.projection.depth(local), faces: facesOf(cell, height, terrain), edges, ...rimsOf(cell, height, terrain) });
  });
  return { tiles: tiles.sort((a, b) => a.depth - b.depth), water };
}

/** A cell's ground and height when it's in the chunk; null when it's outside it. */
function terrainInChunk(chunk: Chunk, cell: TilePoint): CellTerrain | null {
  const origin = chunkOrigin(chunk.key);
  const inside = cell.col >= origin.col && cell.row >= origin.row && cell.col < origin.col + CHUNK_SIZE && cell.row < origin.row + CHUNK_SIZE;
  if (!inside) return null;
  const i = localIndex(chunk.key, cell);
  return { kind: chunk.ground[i] ?? 'void', height: chunk.height[i] ?? 0 };
}

function neighbor(cell: TilePoint, edge: Edge): TilePoint {
  const step = EDGE_STEPS[edge];
  return { col: cell.col + step.col, row: cell.row + step.row };
}

/** The edges toward a cell known to be of another kind. A neighbor not held counts as the same kind. */
function edgesToward(cell: TilePoint, kind: string, terrain: TerrainAround): Edge[] {
  return EDGES.filter((edge) => {
    const other = terrain(neighbor(cell, edge));
    return other !== null && other.kind !== kind;
  });
}

/** The edges and corners toward a lower cell: like faces, a neighbor that is void or not held counts as lower. */
function rimsOf(cell: TilePoint, height: number, terrain: TerrainAround): { rims: Edge[]; rimCorners: Corner[] } {
  const lower = (step: TilePoint) => faceLevels(height, terrain({ col: cell.col + step.col, row: cell.row + step.row })) > 0;
  return { rims: EDGES.filter((edge) => lower(EDGE_STEPS[edge])), rimCorners: CORNERS.filter((corner) => lower(CORNER_STEPS[corner])) };
}

function facesOf(cell: TilePoint, height: number, terrain: TerrainAround): PlacedFace[] {
  return FACE_SIDES.map(({ side, toward }) => ({ side, levels: faceLevels(height, terrain(neighbor(cell, toward))) })).filter(
    (face) => face.levels > 0,
  );
}

/** How many levels a face goes down toward its neighbor: to its height, or to the lowest level past the ground. */
export function faceLevels(height: number, toward: CellTerrain | null): number {
  if (!toward || toward.kind === 'void') return height - MIN_HEIGHT;
  return Math.max(0, height - toward.height);
}

/**
 * The chunks to draw on a frame: the ones that came in, then the shown ones beside them, each once, in that order.
 * A chunk's faces and edges depend on the chunks beside it.
 */
export function chunksToDraw(arrived: readonly ChunkKey[], grid: Grid, isShown: (key: ChunkKey) => boolean): ChunkKey[] {
  const ids = new Set<string>();
  const keys: ChunkKey[] = [];
  const add = (key: ChunkKey) => {
    if (ids.has(chunkId(key))) return;
    ids.add(chunkId(key));
    keys.push(key);
  };
  arrived.forEach(add);
  for (const key of arrived) chunksBeside(grid, key).filter(isShown).forEach(add);
  return keys;
}

/** The water's frame at a time: a new one every `frameMs`, looping after WATER_FRAMES. */
export function waterFrame(now: number, frameMs: number): number {
  return Math.floor(now / frameMs) % WATER_FRAMES;
}

/** Where a chunk's first cell is drawn: the copy of the chunk nearest to your drawn cell. */
export function drawnChunkOrigin(grid: Grid, key: ChunkKey, anchor: TilePoint): TilePoint {
  const origin = chunkOrigin(key);
  const half = CHUNK_SIZE / 2;
  const center = nearestCopy(grid, { col: origin.col + half, row: origin.row + half }, anchor);
  return { col: center.col - half, row: center.row - half };
}

interface ShownChunk {
  key: ChunkKey;
  /** Null for a chunk with no ground to bake: all void or water. */
  ground: { sprite: Sprite; baked: Baked } | null;
  water: { layer: Container; tiles: { sprite: Sprite; foam: Edge[] }[] };
  rocks: { sprite: Sprite; cell: TilePoint; height: number }[];
  /** Where the chunk's first cell is drawn. */
  drawnOrigin: TilePoint;
}

export function createChunksView(deps: ChunksViewDeps): ChunksView {
  const { heights, grid, graphics, log = silent, now = () => performance.now() } = deps;
  const shown = new Map<string, ShownChunk>();
  let arrived: ChunkKey[] = [];
  let flowFrame = waterFrame(now(), deps.waterFrameMs);

  const around: TerrainAround = (cell) => {
    const within = normalize(grid, cell);
    return within ? terrainAt(deps.held.current, within) : null;
  };

  const place = (chunk: ShownChunk, drawnOrigin: TilePoint) => {
    chunk.drawnOrigin = drawnOrigin;
    const screen = heights.projection.toScreen(drawnOrigin);
    if (chunk.ground) {
      chunk.ground.sprite.position.set(screen.x + chunk.ground.baked.x, screen.y + chunk.ground.baked.y);
      chunk.ground.sprite.zIndex = heights.projection.depth(drawnOrigin);
    }
    chunk.water.layer.position.set(screen.x, screen.y);
    const origin = chunkOrigin(chunk.key);
    for (const rock of chunk.rocks) {
      const cell = { col: rock.cell.col - origin.col + drawnOrigin.col, row: rock.cell.row - origin.row + drawnOrigin.row };
      placeOnCell(rock.sprite, cell, heights, rock.height);
    }
  };

  const remove = (id: string) => {
    const chunk = shown.get(id);
    if (!chunk) return;
    chunk.ground?.sprite.destroy({ texture: true, textureSource: true });
    chunk.water.layer.destroy({ children: true });
    for (const rock of chunk.rocks) rock.sprite.destroy();
    shown.delete(id);
  };

  const draw = (chunk: Chunk, again: boolean) => {
    const started = now();
    const id = chunkId(chunk.key);
    remove(id);
    const layout = layoutChunk(chunk, heights, around);
    const ground = drawGround(layout.tiles, graphics, heights.levelPx, deps.bake);
    if (ground) deps.ground.addChild(ground.sprite);
    const water = drawWater(layout.water, graphics, flowFrame);
    deps.water.addChild(water.layer);
    const rocks = chunk.objects.map((object) => {
      const sprite = cellSprite(pickForCell(graphics.objects[object.kind], object.col, object.row, ROCK_SALT));
      deps.entities.addChild(sprite);
      return { sprite, cell: { col: object.col, row: object.row }, height: chunk.height[localIndex(chunk.key, object)] ?? 0 };
    });
    const entry: ShownChunk = { key: chunk.key, ground, water, rocks, drawnOrigin: chunkOrigin(chunk.key) };
    shown.set(id, entry);
    const anchor = deps.anchor();
    place(entry, anchor ? drawnChunkOrigin(grid, chunk.key, anchor) : chunkOrigin(chunk.key));
    log(again ? 'chunk drawn again: a chunk beside it came' : 'chunk drawn', {
      chunk: id,
      drawnOrigin: entry.drawnOrigin,
      water: water.tiles.length,
      rocks: rocks.length,
      ms: Math.round(now() - started),
      shown: shown.size,
    });
  };

  const drawArrived = () => {
    if (arrived.length === 0) return;
    const keys = chunksToDraw(arrived, grid, (key) => shown.has(chunkId(key)));
    const fresh = new Set(arrived.map(chunkId));
    arrived = [];
    for (const key of keys) {
      const chunk = deps.held.current.get(chunkId(key));
      if (chunk) draw(chunk, !fresh.has(chunkId(key)));
    }
  };

  const flow = () => {
    const frame = waterFrame(now(), deps.waterFrameMs);
    if (frame === flowFrame) return;
    flowFrame = frame;
    for (const chunk of shown.values()) {
      for (const tile of chunk.water.tiles) tile.sprite.texture = graphics.terrain.water(frame, tile.foam).texture;
    }
  };

  deps.ticker.add(() => {
    drawArrived();
    flow();
  });

  return {
    show(chunk) {
      arrived.push(chunk.key);
    },
    forget(key) {
      const id = chunkId(key);
      log('chunk forgotten', { chunk: id, shown: shown.size - (shown.has(id) ? 1 : 0) });
      arrived = arrived.filter((k) => chunkId(k) !== id);
      remove(id);
    },
    follow(anchor) {
      for (const [id, chunk] of shown) {
        const drawnOrigin = drawnChunkOrigin(grid, chunk.key, anchor);
        if (drawnOrigin.col !== chunk.drawnOrigin.col || drawnOrigin.row !== chunk.drawnOrigin.row) {
          log('chunk moved to its nearest copy', { chunk: id, from: chunk.drawnOrigin, to: drawnOrigin });
          place(chunk, drawnOrigin);
        }
      }
    },
  };
}

function cellSprite(graphic: SpriteGraphic): Sprite {
  const sprite = new Sprite(graphic.texture);
  sprite.pivot.set(graphic.pivot.x, graphic.pivot.y);
  return sprite;
}

function spriteAt(graphic: SpriteGraphic, at: ScreenPoint): Sprite {
  const sprite = cellSprite(graphic);
  sprite.position.set(at.x, at.y);
  return sprite;
}

/** The chunk's ground tiles and their faces drawn into one texture, from its first cell. Null when there are none. */
function drawGround(tiles: readonly PlacedTile[], graphics: WorldGraphics, levelPx: number, bake: Bake): ShownChunk['ground'] {
  if (tiles.length === 0) return null;
  const container = new Container();
  for (const tile of tiles) {
    for (const face of tile.faces) {
      for (let level = 0; level < face.levels; level++) {
        const graphic = graphics.terrain.face(faceMaterial(tile.kind), face.side);
        container.addChild(spriteAt(graphic, { x: tile.local.x, y: tile.local.y + level * levelPx }));
      }
    }
    container.addChild(spriteAt(topOf(tile, graphics), tile.local));
    if (tile.kind === 'grass' && tile.rims.length + tile.rimCorners.length > 0) {
      container.addChild(spriteAt(graphics.terrain.rim(tile.rims, tile.rimCorners), tile.local));
    }
  }
  const baked = bake(container);
  return { sprite: new Sprite(baked.texture), baked };
}

function topOf(tile: PlacedTile, graphics: WorldGraphics): SpriteGraphic {
  switch (tile.kind) {
    case 'grass':
      return pickForCell(graphics.grass, tile.cell.col, tile.cell.row, GROUND_SALT);
    case 'hole':
      return graphics.terrain.hole(tile.edges);
    case 'bridge':
      return graphics.terrain.bridge(tile.edges);
    case 'ocean':
      return graphics.terrain.ocean(tile.edges);
  }
}

function faceMaterial(kind: BakedGroundKind): FaceMaterial {
  return kind === 'bridge' ? 'wood' : 'earth';
}

/** One sprite per water cell, all on the same frame of the flow. */
function drawWater(water: readonly PlacedWater[], graphics: WorldGraphics, frame: number): ShownChunk['water'] {
  const layer = new Container();
  const tiles = water.map(({ local, foam }) => {
    const sprite = spriteAt(graphics.terrain.water(frame, foam), local);
    layer.addChild(sprite);
    return { sprite, foam };
  });
  return { layer, tiles };
}

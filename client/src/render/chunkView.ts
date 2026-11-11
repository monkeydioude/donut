import { Container, Sprite, type Texture } from 'pixi.js';
import { FLAT_BLOCK, type BlockShape } from '../art/block';
import type { FaceMaterial, FaceSide } from '../art/cliff';
import { CORNER_STEPS, CORNERS, EDGE_STEPS, EDGES, type Corner, type Edge } from '../art/edges';
import { floorLookOf, glassLookOf } from '../art/city';
import { patternPlace } from '../art/facets';
import { WATER_FRAMES } from '../art/water';
import { pickForCell } from '../core/rng';
import { silent, type Log } from '../debug/log';
import { groundCenterAt, type Heights } from '../iso/heights';
import type { ScreenPoint, TilePoint } from '../iso/projection';
import { localIndex, terrainAt, type CellTerrain, type Chunk, type HeldChunksRef } from '../map/chunks';
import { CHUNK_SIZE, chunkId, chunkOrigin, chunksAround, nearestCopy, normalize, type ChunkKey, type Grid } from '../map/grid';
import { MIN_HEIGHT, type BakedGroundKind, type MapObject } from '../map/worldMap';
import { placeObject } from './mapView';
import type { FrameTicker } from './playersView';
import type { ObjectGraphic, SpriteGraphic, WorldGraphics } from './tileGraphics';

/** A cliff face of a cell: on which side, and how many levels it goes down. */
export interface PlacedFace {
  side: FaceSide;
  levels: number;
}

/** A tile baked into a chunk's ground: its kind, its map cell, where its ground center is drawn, at its height. */
export interface PlacedTile {
  kind: BakedGroundKind;
  cell: TilePoint;
  height: number;
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
  /** A mountain's or a glacier's block; null for the other kinds. */
  block: PlacedBlock | null;
}

/** A mountain's or a glacier's block (ADR 0011): its shape, the edges its top is outlined on, and an icy top. */
export interface PlacedBlock {
  shape: BlockShape;
  outline: Edge[];
  iced: boolean;
}

/** A mountain's top is drawn in ice from this height up (ADR 0011). */
export const ICED_FROM = 6;

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
  /** Destroys a forgotten chunk's textures and objects. */
  forget(key: ChunkKey): void;
  /** Moves every chunk to its copy nearest to your drawn cell, which moved. */
  follow(anchor: TilePoint): void;
  /** The map cell of the dome drawn at a world point, by its opaque pixels; null when there's none (ADR 0012). */
  objectAt(point: ScreenPoint): TilePoint | null;
  /** Destroys every chunk shown and its textures: the game left this map (ADR 0012). */
  clear(): void;
}

/** An object drawn somewhere: its sprite's position and draw order, its graphic, and its map cell. */
export interface DrawnObject {
  position: ScreenPoint;
  zIndex: number;
  graphic: ObjectGraphic;
  cell: TilePoint;
}

/** The frontmost of the objects whose opaque pixels hold the world point: its map cell, or null. */
export function pickObject(objects: readonly DrawnObject[], point: ScreenPoint): TilePoint | null {
  let picked: DrawnObject | null = null;
  for (const object of objects) {
    const x = point.x - object.position.x + object.graphic.pivot.x;
    const y = point.y - object.position.y + object.graphic.pivot.y;
    if (object.graphic.opaqueAt(x, y) && (!picked || object.zIndex > picked.zIndex)) picked = object;
  }
  return picked ? picked.cell : null;
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
    else {
      const faces = facesOf(cell, height, terrain);
      const block = blockOf(cell, kind, height, terrain);
      tiles.push({ kind, cell, height, local: at, depth: heights.projection.depth(local), faces, edges, ...rimsOf(cell, height, terrain), block });
    }
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
  return stepFrom(cell, EDGE_STEPS[edge]);
}

function stepFrom(cell: TilePoint, step: TilePoint): TilePoint {
  return { col: cell.col + step.col, row: cell.row + step.row };
}

/** A block's corners that may rise, each with the neighbors that touch it behind or beside the cell. */
const RISING_CORNERS: readonly { corner: 'n' | 'w' | 'e'; touching: readonly TilePoint[] }[] = [
  { corner: 'n', touching: [CORNER_STEPS.n, EDGE_STEPS.nw, EDGE_STEPS.ne] },
  { corner: 'w', touching: [EDGE_STEPS.nw, CORNER_STEPS.w] },
  { corner: 'e', touching: [EDGE_STEPS.ne, CORNER_STEPS.e] },
];
const ALL_AROUND: readonly TilePoint[] = [...Object.values(EDGE_STEPS), ...Object.values(CORNER_STEPS)];

/** A mountain's or glacier's block: its shape, its outline and its icy top. Null for the other kinds. */
function blockOf(cell: TilePoint, kind: BakedGroundKind, height: number, terrain: TerrainAround): PlacedBlock | null {
  if (kind !== 'mountain' && kind !== 'glacier') return null;
  return { shape: shapeOf(cell, kind, height, terrain), outline: outlineOf(cell, kind, height, terrain), iced: kind === 'mountain' && height >= ICED_FROM };
}

/**
 * The shape of a mountain or glacier cell (ADR 0011): each of its back and side corners rises to the highest cell of
 * its kind touching it behind or beside, so the top leans up to it. A mountain with no higher mountain around is a
 * peak. A cell not held, or of another kind, raises nothing.
 */
export function shapeOf(cell: TilePoint, kind: BakedGroundKind, height: number, terrain: TerrainAround): BlockShape {
  const levelsUp = (step: TilePoint) => {
    const other = terrain(stepFrom(cell, step));
    return other?.kind === kind ? Math.max(0, other.height - height) : 0;
  };
  const rise = (touching: readonly TilePoint[]) => Math.max(0, ...touching.map(levelsUp));
  const [n = 0, w = 0, e = 0] = RISING_CORNERS.map(({ touching }) => rise(touching));
  return { n, w, e, peak: kind === 'mountain' && rise(ALL_AROUND) === 0 };
}

/** A block's back edges: behind them, a lower cell of its kind shows as a step down. In front, its wall joins them. */
const BACK_EDGES: readonly Edge[] = ['nw', 'ne'];

/**
 * The edges a block's top is outlined on: toward a cell of another kind, or a lower one of its kind behind it. A lower
 * cell of its kind in front leans up to this one, which joins them. A cell not held gets none: it's drawn again when
 * that cell comes.
 */
function outlineOf(cell: TilePoint, kind: BakedGroundKind, top: number, terrain: TerrainAround): Edge[] {
  return EDGES.filter((edge) => {
    const other = terrain(neighbor(cell, edge));
    if (other === null) return false;
    return other.kind !== kind || (BACK_EDGES.includes(edge) && other.height < top);
  });
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
 * The chunks to draw on a frame: the ones that came in, then the shown ones around them, each once, in that order.
 * A chunk's faces, edges and walls depend on the chunks around it, diagonals included (ADR 0011).
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
  for (const key of arrived) chunksAround(grid, key).filter(isShown).forEach(add);
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
  objects: { sprite: Sprite; object: MapObject; graphic: ObjectGraphic; height: number }[];
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
    for (const { sprite, object, height } of chunk.objects) {
      const back = { col: object.col - origin.col + drawnOrigin.col, row: object.row - origin.row + drawnOrigin.row };
      placeObject(sprite, object.kind, back, heights, height);
    }
  };

  const remove = (id: string) => {
    const chunk = shown.get(id);
    if (!chunk) return;
    chunk.ground?.sprite.destroy({ texture: true, textureSource: true });
    chunk.water.layer.destroy({ children: true });
    for (const { sprite } of chunk.objects) sprite.destroy();
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
    const objects = chunk.objects.map((object) => {
      const graphic = pickForCell(graphics.objects[object.kind], object.col, object.row, ROCK_SALT);
      const sprite = cellSprite(graphic);
      deps.entities.addChild(sprite);
      return { sprite, object, graphic, height: chunk.height[localIndex(chunk.key, object)] ?? 0 };
    });
    const entry: ShownChunk = { key: chunk.key, ground, water, objects, drawnOrigin: chunkOrigin(chunk.key) };
    shown.set(id, entry);
    const anchor = deps.anchor();
    place(entry, anchor ? drawnChunkOrigin(grid, chunk.key, anchor) : chunkOrigin(chunk.key));
    log(again ? 'chunk drawn again: a chunk beside it came' : 'chunk drawn', {
      chunk: id,
      drawnOrigin: entry.drawnOrigin,
      water: water.tiles.length,
      objects: objects.length,
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
    clear() {
      log('every chunk cleared', { shown: shown.size });
      arrived = [];
      for (const id of [...shown.keys()]) remove(id);
    },
    objectAt(point) {
      const domes: DrawnObject[] = [];
      for (const chunk of shown.values()) {
        for (const { sprite, object, graphic } of chunk.objects) {
          if (object.kind === 'dome') domes.push({ position: sprite.position, zIndex: sprite.zIndex, graphic, cell: { col: object.col, row: object.row } });
        }
      }
      return pickObject(domes, point);
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
        container.addChild(spriteAt(faceOf(tile, face.side, level, graphics), { x: tile.local.x, y: tile.local.y + level * levelPx }));
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

/** One level of a tile's face: a wall's is glass, its panels picked from its place (ADR 0012). */
function faceOf(tile: PlacedTile, side: FaceSide, level: number, graphics: WorldGraphics): SpriteGraphic {
  if (tile.kind === 'wall') return graphics.terrain.glass(side, glassLookOf(tile.cell, tile.height, level));
  return graphics.terrain.face(faceMaterial(tile.kind), side);
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
    case 'mountain':
      return graphics.terrain.mountain({ place: patternPlace(tile.cell), outline: tile.block?.outline ?? [], shape: tile.block?.shape ?? FLAT_BLOCK, iced: tile.block?.iced ?? false });
    case 'glacier':
      return graphics.terrain.glacier({ place: patternPlace(tile.cell), edges: tile.block?.outline ?? [], shape: tile.block?.shape ?? FLAT_BLOCK });
    case 'floor':
      return graphics.terrain.floor(floorLookOf(tile.cell));
    case 'wall':
      return graphics.terrain.wallTop();
  }
}

/** What a cell's cliff faces are made of: wood under a bridge, stone under a mountain, ice under a glacier, else earth. */
export function faceMaterial(kind: BakedGroundKind): FaceMaterial {
  switch (kind) {
    case 'bridge':
      return 'wood';
    case 'mountain':
      return 'stone';
    case 'glacier':
      return 'ice';
    default:
      return 'earth';
  }
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

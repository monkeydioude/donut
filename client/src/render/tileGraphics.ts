import { CanvasSource, Texture } from 'pixi.js';
import type { Rng } from '../core/rng';
import { shapeKey } from '../art/block';
import { floorKey, glassKey, type FloorLook, type GlassLook } from '../art/city';
import type { FaceMaterial, FaceSide } from '../art/cliff';
import { edgesKey, type Corner, type Edge } from '../art/edges';
import type { GlacierLook } from '../art/glacier';
import type { MountainLook } from '../art/mountain';
import type { PixelBuffer } from '../art/pixelBuffer';
import type { ArtSet, SpriteArt, TerrainArt } from '../art/tileArtSet';
import { groundCenter, type TileFrame } from '../art/tileFrame';
import type { ObjectKind } from '../map/worldMap';

/** A texture ready to draw, with the point of it that sits on its cell's ground center. */
export interface SpriteGraphic {
  texture: Texture;
  pivot: { x: number; y: number };
}

/** An object's sprite, which also tells where it's drawn: the pointer picks a dome by its pixels (ADR 0012). */
export interface ObjectGraphic extends SpriteGraphic {
  /** True on the texture's opaque pixels. */
  opaqueAt(x: number, y: number): boolean;
}

/** The terrain's sprites, painted the first time they're needed, then kept. */
export interface TerrainGraphics {
  face(material: FaceMaterial, side: FaceSide): SpriteGraphic;
  rim(edges: readonly Edge[], corners: readonly Corner[]): SpriteGraphic;
  water(flowFrame: number, foam: readonly Edge[]): SpriteGraphic;
  hole(rim: readonly Edge[]): SpriteGraphic;
  ocean(foam: readonly Edge[]): SpriteGraphic;
  bridge(rails: readonly Edge[]): SpriteGraphic;
  mountain(look: MountainLook): SpriteGraphic;
  glacier(look: GlacierLook): SpriteGraphic;
  floor(look: FloorLook): SpriteGraphic;
  wallTop(): SpriteGraphic;
  glass(side: FaceSide, look: GlassLook): SpriteGraphic;
}

/** Several variants of grass and of each object kind, so the world doesn't look repetitive; the terrain's pieces. */
export interface WorldGraphics {
  grass: readonly SpriteGraphic[];
  objects: Record<ObjectKind, readonly ObjectGraphic[]>;
  terrain: TerrainGraphics;
}

export type TextureFactory = (buffer: PixelBuffer) => Texture;

export interface WorldGraphicsOptions {
  artSet: ArtSet;
  variantsPerKind: number;
  rng: Rng;
  toTexture?: TextureFactory;
}

export function createWorldGraphics(options: WorldGraphicsOptions): WorldGraphics {
  const { artSet, variantsPerKind, rng, toTexture = pixelBufferToTexture } = options;
  const graphicOf = (art: SpriteArt): ObjectGraphic => {
    const buffer = art.paint(art.frame, rng);
    return { texture: toTexture(buffer), pivot: groundCenter(art.frame), opaqueAt: (x, y) => isOpaqueAt(buffer, x, y) };
  };
  const variantsOf = (art: SpriteArt, count = variantsPerKind) => Array.from({ length: count }, () => graphicOf(art));
  return {
    grass: variantsOf(artSet.grass),
    // a dome looks the same everywhere: one variant
    objects: { rock: variantsOf(artSet.objects.rock), dome: variantsOf(artSet.objects.dome, 1) },
    terrain: terrainGraphics(artSet.terrain, toTexture),
  };
}

/**
 * Each terrain piece painted once for its key, then kept. Mountains and glaciers have the most: one per place in their
 * pattern, edges, shape and icy top (ADR 0011), the few a map's cells use.
 */
function terrainGraphics(art: TerrainArt, toTexture: TextureFactory): TerrainGraphics {
  const kept = new Map<string, SpriteGraphic>();
  const once = (key: string, frame: TileFrame, paint: () => PixelBuffer): SpriteGraphic => {
    let graphic = kept.get(key);
    if (!graphic) {
      graphic = spriteGraphic(paint(), frame, toTexture);
      kept.set(key, graphic);
    }
    return graphic;
  };
  return {
    face: (material, side) => once(`face ${material} ${side}`, art.tileFrame, () => art.face(material, side)),
    rim: (edges, corners) => once(`rim ${edgesKey([...edges, ...corners])}`, art.tileFrame, () => art.rim(edges, corners)),
    water: (flowFrame, foam) => once(`water ${flowFrame} ${edgesKey(foam)}`, art.tileFrame, () => art.water(flowFrame, foam)),
    hole: (rim) => once(`hole ${edgesKey(rim)}`, art.tileFrame, () => art.hole(rim)),
    ocean: (foam) => once(`ocean ${edgesKey(foam)}`, art.tileFrame, () => art.ocean(foam)),
    bridge: (rails) => once(`bridge ${edgesKey(rails)}`, art.bridgeFrame, () => art.bridge(rails)),
    mountain: (look) =>
      once(`mountain ${placeKey(look.place)} ${edgesKey(look.outline)} ${shapeKey(look.shape)} ${look.iced ? 'iced' : 'rock'}`, art.blockFrame(look.shape), () => art.mountain(look)),
    glacier: (look) => once(`glacier ${placeKey(look.place)} ${edgesKey(look.edges)} ${shapeKey(look.shape)}`, art.blockFrame(look.shape), () => art.glacier(look)),
    floor: (look) => once(floorKey(look), art.tileFrame, () => art.floor(look)),
    wallTop: () => once('wall top', art.tileFrame, () => art.wallTop()),
    glass: (side, look) => once(glassKey(side, look), art.tileFrame, () => art.glass(side, look)),
  };
}

/** A pixel inside the buffer that isn't transparent. */
function isOpaqueAt(buffer: PixelBuffer, x: number, y: number): boolean {
  const px = Math.floor(x);
  const py = Math.floor(y);
  if (px < 0 || py < 0 || px >= buffer.width || py >= buffer.height) return false;
  return buffer.data[(py * buffer.width + px) * 4 + 3] === 255;
}

function placeKey(place: { col: number; row: number }): string {
  return `${place.col},${place.row}`;
}

/** A one-off sprite (a border, a marker) from a buffer painted in a tile frame. */
export function spriteGraphic(buffer: PixelBuffer, frame: TileFrame, toTexture: TextureFactory = pixelBufferToTexture): SpriteGraphic {
  return { texture: toTexture(buffer), pivot: groundCenter(frame) };
}

/** Uploads a pixel buffer as a crisp (no smoothing) Pixi texture. */
export function pixelBufferToTexture(buffer: PixelBuffer): Texture {
  const canvas = document.createElement('canvas');
  canvas.width = buffer.width;
  canvas.height = buffer.height;
  const context = canvas.getContext('2d');
  if (!context) throw new Error('pixelBufferToTexture: 2D canvas is not supported');
  context.putImageData(new ImageData(buffer.data, buffer.width, buffer.height), 0, 0);
  return new Texture({ source: new CanvasSource({ resource: canvas, resolution: 1, scaleMode: 'nearest' }) });
}

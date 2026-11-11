import { CanvasSource, Texture } from 'pixi.js';
import type { Rng } from '../core/rng';
import { shapeKey } from '../art/block';
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
}

/** Several variants of grass and of each object kind, so the world doesn't look repetitive; the terrain's pieces. */
export interface WorldGraphics {
  grass: readonly SpriteGraphic[];
  objects: Record<ObjectKind, readonly SpriteGraphic[]>;
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
  const graphicOf = (art: SpriteArt): SpriteGraphic => ({
    texture: toTexture(art.paint(art.frame, rng)),
    pivot: groundCenter(art.frame),
  });
  const variantsOf = (art: SpriteArt) => Array.from({ length: variantsPerKind }, () => graphicOf(art));
  return {
    grass: variantsOf(artSet.grass),
    objects: { rock: variantsOf(artSet.objects.rock) },
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
  };
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

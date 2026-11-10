import { CanvasSource, Texture } from 'pixi.js';
import type { Rng } from '../core/rng';
import type { PixelBuffer } from '../art/pixelBuffer';
import type { PlayerLook } from '../art/player';
import type { ArtSet, SpriteArt } from '../art/tileArtSet';
import { groundCenter, type TileFrame } from '../art/tileFrame';
import type { DrawnGroundKind, ObjectKind } from '../map/worldMap';

/** A texture ready to draw, with the point of it that sits on its cell's ground center. */
export interface SpriteGraphic {
  texture: Texture;
  pivot: { x: number; y: number };
}

/** Several variants per ground and object kind, so the world doesn't look repetitive; one per player look. */
export interface WorldGraphics {
  ground: Record<DrawnGroundKind, readonly SpriteGraphic[]>;
  objects: Record<ObjectKind, readonly SpriteGraphic[]>;
  players: Record<PlayerLook, SpriteGraphic>;
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
    ground: { grass: variantsOf(artSet.ground.grass) },
    objects: { rock: variantsOf(artSet.objects.rock) },
    players: { you: graphicOf(artSet.players.you), other: graphicOf(artSet.players.other) },
  };
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

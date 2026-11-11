import type { Rng } from '../core/rng';
import type { ObjectKind } from '../map/worldMap';
import { paintBridge, RAIL_PX } from './bridge';
import { paintFace, paintRim, type FaceMaterial, type FaceSide } from './cliff';
import type { Corner, Edge } from './edges';
import { paintGrass } from './grass';
import { paintHole } from './hole';
import { paintOcean } from './ocean';
import type { PixelBuffer } from './pixelBuffer';
import { paintRock } from './rock';
import type { TileFrame } from './tileFrame';
import { paintWater } from './water';

/** How one sprite looks: its texture layout and how to paint a variant of it. */
export interface SpriteArt {
  frame: TileFrame;
  paint(frame: TileFrame, rng: Rng): PixelBuffer;
}

/**
 * The terrain's sprites (ADR 0007), each painted the same every time for the same neighbors: the pieces of a cell
 * that depend on the cells around it. Each is painted in its frame, and sits on its ground center.
 */
export interface TerrainArt {
  /** Art pixels a cell rises per level of height. */
  levelPx: number;
  /** The frame of faces, water and holes: a plain ground tile. A face's texture is levelPx taller. */
  tileFrame: TileFrame;
  /** A bridge's frame: room above the deck for its rails. */
  bridgeFrame: TileFrame;
  /** One level of a cliff face. */
  face(material: FaceMaterial, side: FaceSide): PixelBuffer;
  /** The rim of a grass top along its edges toward a lower cell, and on its corners toward a lower diagonal one. */
  rim(edges: readonly Edge[], corners: readonly Corner[]): PixelBuffer;
  /** Water at one frame of its flow, with foam on the edges toward cells that aren't water. */
  water(flowFrame: number, foam: readonly Edge[]): PixelBuffer;
  /** A hole, its rim and inner walls on the edges toward cells that aren't holes. */
  hole(rim: readonly Edge[]): PixelBuffer;
  /** The still ocean, with foam on the edges toward cells that aren't ocean (ADR 0008). */
  ocean(foam: readonly Edge[]): PixelBuffer;
  /** A bridge's deck, with rails on its sides toward cells that aren't bridge. */
  bridge(rails: readonly Edge[]): PixelBuffer;
}

/** Every sprite of the world painted in code: grass, the terrain and the objects. Players come from a sheet. */
export interface ArtSet {
  grass: SpriteArt;
  objects: Record<ObjectKind, SpriteArt>;
  terrain: TerrainArt;
}

export interface ArtOptions {
  tileWidth: number;
  tileHeight: number;
  rockHeadroom: number;
  levelPx: number;
}

/** Procedurally generated art, until real pre-rendered sprites replace it. */
export function createGeneratedArt({ tileWidth, tileHeight, rockHeadroom, levelPx }: ArtOptions): ArtSet {
  const frame = (headroom: number): TileFrame => ({ tileWidth, tileHeight, headroom });
  const tileFrame = frame(0);
  const bridgeFrame = frame(RAIL_PX + 1);
  return {
    grass: { frame: tileFrame, paint: paintGrass },
    objects: { rock: { frame: frame(rockHeadroom), paint: paintRock } },
    terrain: {
      levelPx,
      tileFrame,
      bridgeFrame,
      face: (material, side) => paintFace(tileFrame, levelPx, material, side),
      rim: (edges, corners) => paintRim(tileFrame, edges, corners),
      water: (flowFrame, foam) => paintWater(tileFrame, flowFrame, foam),
      hole: (rim) => paintHole(tileFrame, rim),
      ocean: (foam) => paintOcean(tileFrame, foam),
      bridge: (rails) => paintBridge(bridgeFrame, rails),
    },
  };
}

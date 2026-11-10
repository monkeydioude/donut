import type { Rng } from '../core/rng';
import type { DrawnGroundKind, ObjectKind } from '../map/worldMap';
import { paintGrass } from './grass';
import type { PixelBuffer } from './pixelBuffer';
import { paintPlayer, type PlayerLook } from './player';
import { paintRock } from './rock';
import type { TileFrame } from './tileFrame';

/** How one sprite looks: its texture layout and how to paint a variant of it. */
export interface SpriteArt {
  frame: TileFrame;
  paint(frame: TileFrame, rng: Rng): PixelBuffer;
}

/** Every sprite of the world: the ground tiles, the objects on them, the players. */
export interface ArtSet {
  ground: Record<DrawnGroundKind, SpriteArt>;
  objects: Record<ObjectKind, SpriteArt>;
  players: Record<PlayerLook, SpriteArt>;
}

export interface ArtOptions {
  tileWidth: number;
  tileHeight: number;
  rockHeadroom: number;
  playerHeadroom: number;
}

/** Procedurally generated art, until real pre-rendered sprites replace it. */
export function createGeneratedArt({ tileWidth, tileHeight, rockHeadroom, playerHeadroom }: ArtOptions): ArtSet {
  const frame = (headroom: number): TileFrame => ({ tileWidth, tileHeight, headroom });
  return {
    ground: { grass: { frame: frame(0), paint: paintGrass } },
    objects: { rock: { frame: frame(rockHeadroom), paint: paintRock } },
    players: {
      you: { frame: frame(playerHeadroom), paint: (f) => paintPlayer(f, 'you') },
      other: { frame: frame(playerHeadroom), paint: (f) => paintPlayer(f, 'other') },
    },
  };
}

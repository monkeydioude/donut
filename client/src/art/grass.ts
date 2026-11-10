import { pickOne, randomInt, type Rng } from '../core/rng';
import { innerGroundPoints, stampOnGround, type PatternPixel } from './drawing';
import { createPixelBuffer, forEachPixel, setPixel, type PixelBuffer } from './pixelBuffer';
import { isOnGround, textureSize, type TileFrame } from './tileFrame';

export const GRASS_COLORS = {
  base: 0x74d3a5,
  tuft: 0x4fb386,
  flowerCenter: 0xf6c75a,
  petals: [0xf8f5ec, 0xf4a9c2, 0xc9b1f3],
} as const;

/** Small "v" and "w" marks, drawn in pixel offsets around their origin. */
const TUFT_SHAPES = [
  [[-1, -1], [1, -1], [0, 0]],
  [[-2, -1], [0, -1], [2, -1], [-1, 0], [1, 0]],
] as const;
const PETAL_OFFSETS = [[-1, 0], [1, 0], [0, -1], [0, 1]] as const;
const TUFT_COUNT = { min: 1, max: 3 };
const FLOWER_CHANCE = 0.3;

export function paintGrass(frame: TileFrame, rng: Rng): PixelBuffer {
  const { width, height } = textureSize(frame);
  const buffer = createPixelBuffer(width, height);
  paintGrassGround(buffer, frame, rng);
  return buffer;
}

/** Flat grass with a few tufts and sometimes a flower. Other tiles paint it under their objects. */
export function paintGrassGround(buffer: PixelBuffer, frame: TileFrame, rng: Rng): void {
  forEachPixel(buffer.width, buffer.height, (x, y) => {
    if (isOnGround(frame, x, y)) setPixel(buffer, x, y, GRASS_COLORS.base);
  });
  const spots = innerGroundPoints(buffer, frame);
  const tuftCount = TUFT_COUNT.min + randomInt(rng, TUFT_COUNT.max - TUFT_COUNT.min + 1);
  for (let i = 0; i < tuftCount; i++) stampOnGround(buffer, frame, pickOne(rng, spots), tuftPattern(rng));
  if (rng() < FLOWER_CHANCE) stampOnGround(buffer, frame, pickOne(rng, spots), flowerPattern(rng));
}

function tuftPattern(rng: Rng): PatternPixel[] {
  return pickOne(rng, TUFT_SHAPES).map(([dx, dy]) => ({ dx, dy, rgb: GRASS_COLORS.tuft }));
}

function flowerPattern(rng: Rng): PatternPixel[] {
  const petal = pickOne(rng, GRASS_COLORS.petals);
  const petals = PETAL_OFFSETS.map(([dx, dy]) => ({ dx, dy, rgb: petal }));
  return [...petals, { dx: 0, dy: 0, rgb: GRASS_COLORS.flowerCenter }];
}

export interface GameConfig {
  /** Seeds the generated art: which variants and which details. */
  seed: number;
  /** The map to play on, as the back names it. */
  mapSlug: string;
  /** Size of a tile's ground diamond, in art pixels. */
  tile: { width: number; height: number };
  /** Room in art pixels above the ground for rocks and players. */
  rockHeadroom: number;
  playerHeadroom: number;
  variantsPerKind: number;
  /** Screen pixels per art pixel. Whole numbers only, to keep pixels crisp and chunky. */
  zoom: number;
  background: string;
  /** Screen pixels per pixel of the finger cursor. */
  cursorScale: number;
  /** Debug logs in the browser console: on in dev builds. */
  debug: boolean;
}

export function createDefaultConfig(): GameConfig {
  return {
    seed: 42,
    mapSlug: 'world',
    tile: { width: 32, height: 16 },
    rockHeadroom: 6,
    playerHeadroom: 10,
    variantsPerKind: 16,
    zoom: 3,
    background: '#1d2330',
    cursorScale: 2,
    debug: import.meta.env.DEV,
  };
}

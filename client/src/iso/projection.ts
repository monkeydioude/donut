export interface TilePoint {
  col: number;
  row: number;
}

export interface ScreenPoint {
  x: number;
  y: number;
}

/** 2:1 isometric projection between grid coordinates and world pixels. */
export interface IsoProjection {
  /** World position of the center of the tile's ground diamond. */
  toScreen(tile: TilePoint): ScreenPoint;
  /** The tile whose ground diamond holds the world point: the inverse of toScreen, rounded to a tile. */
  toTile(point: ScreenPoint): TilePoint;
  /** The exact inverse of toScreen: a world point in fractional tiles, not rounded. */
  toTileFraction(point: ScreenPoint): TilePoint;
  /** Draw order: tiles with a higher depth are drawn on top. */
  depth(tile: TilePoint): number;
}

export function createIsoProjection(tileWidth: number, tileHeight: number): IsoProjection {
  const halfWidth = tileWidth / 2;
  const halfHeight = tileHeight / 2;
  const toTileFraction = ({ x, y }: ScreenPoint): TilePoint => ({
    col: (x / halfWidth + y / halfHeight) / 2,
    row: (y / halfHeight - x / halfWidth) / 2,
  });
  return {
    toScreen: ({ col, row }) => ({
      x: (col - row) * halfWidth,
      y: (col + row) * halfHeight,
    }),
    // Inside a diamond, |col − c| ≤ ½ and |row − r| ≤ ½: rounding the fractional tile finds it.
    toTile: (point) => {
      const { col, row } = toTileFraction(point);
      return { col: roundToTile(col), row: roundToTile(row) };
    },
    toTileFraction,
    depth: ({ col, row }) => col + row,
  };
}

/** Math.round, without the -0 it gives for small negatives. */
function roundToTile(value: number): number {
  return Math.round(value) || 0;
}

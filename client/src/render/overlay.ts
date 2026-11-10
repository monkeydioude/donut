import { Container, Graphics, Sprite } from 'pixi.js';
import { silent, type Log } from '../debug/log';
import type { IsoProjection, ScreenPoint, TilePoint } from '../iso/projection';
import { nearestCopy, type Grid } from '../map/grid';
import { positionOnCell } from './mapView';
import type { FrameTicker } from './playersView';
import type { SpriteGraphic } from './tileGraphics';

/** What is drawn on the ground, under the rocks and players: borders and your path. */
export interface OverlayView {
  showHover(cell: TilePoint | null): void;
  showSelection(cell: TilePoint | null): void;
  /** The path from your player to its end; nothing when empty. */
  showPath(path: readonly TilePoint[]): void;
}

export interface OverlayDeps {
  layer: Container;
  projection: IsoProjection;
  borders: { hover: SpriteGraphic; selection: SpriteGraphic };
  ticker: FrameTicker;
  /** Where your player is drawn right now, gliding or not: the path line starts there. */
  pathStart: () => ScreenPoint | null;
  /** Your drawn cell: borders and paths are drawn at their copy nearest to it. */
  anchor: () => TilePoint | null;
  grid: Grid;
  log?: Log;
}

const LINE = { outline: 0x2b2540, color: 0xffffff, width: 1, outlineWidth: 3 } as const;
/** Half sizes of the diamond that ends the path, in art pixels. */
const END_MARK = { halfWidth: 5, halfHeight: 3 } as const;

/**
 * The points the path line goes through: where it starts, then the ground center of every cell of the path. Each cell
 * is drawn at its copy nearest to the one before, from your drawn cell, so the line never jumps across the world.
 */
export function pathPoints(start: ScreenPoint, from: TilePoint, path: readonly TilePoint[], projection: IsoProjection, grid: Grid): ScreenPoint[] {
  if (path.length === 0) return [];
  let previous = from;
  const points = path.map((cell) => {
    previous = nearestCopy(grid, cell, previous);
    return projection.toScreen(previous);
  });
  return [start, ...points];
}

export function createOverlay({ layer, projection, borders, ticker, pathStart, anchor, grid, log = silent }: OverlayDeps): OverlayView {
  const hover = hiddenBorder(borders.hover);
  const selection = hiddenBorder(borders.selection);
  const line = new Graphics();
  layer.addChild(hover, selection, line); // a fixed order: the line on top, no zIndex
  let hovered: TilePoint | null = null;
  let selected: TilePoint | null = null;
  let path: readonly TilePoint[] = [];

  const drawn = (cell: TilePoint | null) => {
    const at = anchor();
    return cell && at ? nearestCopy(grid, cell, at) : cell;
  };
  const showBorders = () => {
    showBorder(hover, drawn(hovered && !(selected && sameCell(hovered, selected)) ? hovered : null), projection);
    showBorder(selection, drawn(selected), projection);
  };
  // the line follows your sprite as it glides, so it is redrawn every frame while there is a path
  ticker.add(() => {
    const start = pathStart();
    const from = anchor();
    drawPath(line, start && from ? pathPoints(start, from, path, projection, grid) : []);
  });

  return {
    showHover(cell) {
      hovered = cell;
      showBorders();
    },
    showSelection(cell) {
      selected = cell;
      showBorders();
    },
    showPath(cells) {
      if (cells.length !== path.length || cells[0] !== path[0]) log('path line', { steps: cells.length, path: cells });
      path = cells;
    },
  };
}

function hiddenBorder(graphic: SpriteGraphic): Sprite {
  const sprite = new Sprite(graphic.texture);
  sprite.pivot.set(graphic.pivot.x, graphic.pivot.y);
  sprite.visible = false;
  return sprite;
}

function showBorder(sprite: Sprite, cell: TilePoint | null, projection: IsoProjection): void {
  sprite.visible = cell !== null;
  if (cell) positionOnCell(sprite, cell, projection);
}

function sameCell(a: TilePoint, b: TilePoint): boolean {
  return a.col === b.col && a.row === b.row;
}

/** A white line with a dark outline through the points, ending with a small diamond. */
function drawPath(line: Graphics, points: readonly ScreenPoint[]): void {
  line.clear();
  const [first, ...rest] = points;
  const end = rest.at(-1);
  if (!first || !end) return;
  for (const { width, color } of [{ width: LINE.outlineWidth, color: LINE.outline }, { width: LINE.width, color: LINE.color }]) {
    line.moveTo(first.x, first.y);
    for (const point of rest) line.lineTo(point.x, point.y);
    line.stroke({ width, color, cap: 'round', join: 'round' });
  }
  const { halfWidth: w, halfHeight: h } = END_MARK;
  line.poly([end.x, end.y - h, end.x + w, end.y, end.x, end.y + h, end.x - w, end.y]).fill(LINE.color).stroke({ width: 1, color: LINE.outline });
}

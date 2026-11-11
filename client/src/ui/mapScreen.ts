import { silent, type Log } from '../debug/log';
import { exploredShare, exploredText, isExplored, type Exploration } from '../game/exploration';
import { blinkOn, donutMark, mapScale, rectangleMark, type DonutRing, type DonutShape, type Size } from '../game/mapMarks';
import type { ScreenPoint, TilePoint } from '../iso/projection';
import { CHUNK_SIZE, type Grid } from '../map/grid';
import { gridOf, type MapDescription } from '../map/worldMap';

/** What the map screen shows (ADR 0005): your whereabouts, your region's name, what you explored. */
export interface MapView {
  you: TilePoint | null;
  others: readonly TilePoint[];
  regionName: string | null;
  explored: Exploration;
}

/** The screen's art as image URLs, already scaled up by `scale`, and the ring your donut mark moves on. */
export interface MapArt {
  donut: string;
  you: Readonly<Record<DonutShape, string>>;
  other: string;
  /** Screen pixels per art pixel of the donut and the marks. */
  scale: number;
  ring: DonutRing;
}

export interface MapScreenDeps {
  host: HTMLElement;
  map: MapDescription;
  imageUrl: string;
  art: MapArt;
  viewport: () => Size;
  now?: () => number;
  log?: Log;
}

export interface MapScreen {
  open(view: MapView): void;
  /** Every frame while it's open: the marks, their blink, the veil, the region. */
  update(view: MapView): void;
  close(): void;
  isOpen(): boolean;
}

/** The margin left around the map on the screen, in CSS pixels, on each side. */
const MARGIN = 24;
/** The veil over what you haven't explored: dark blue-gray, the land still showing through. */
const VEIL = 'rgba(29, 35, 48, 0.55)';

/**
 * The map screen: the map as one block over a dim backdrop that catches the pointer, so the game under it gets no
 * clicks. The names, the donut and the explored share are laid over the map.
 */
export function createMapScreen(deps: MapScreenDeps): MapScreen {
  const log = deps.log ?? silent;
  const now = deps.now ?? (() => performance.now());
  const grid = gridOf(deps.map);
  const dom = buildScreen(deps.host, deps.map, deps.art, grid);
  const others: HTMLImageElement[] = [];
  let shownExplored: Exploration | null = null;
  let scale = 0;

  const fit = () => {
    const viewport = deps.viewport();
    const next = mapScale(grid, { width: viewport.width - 2 * MARGIN, height: viewport.height - 2 * MARGIN });
    if (next === scale) return;
    scale = next;
    for (const element of [dom.block, dom.image, dom.veil]) sizeTo(element, grid, scale);
    log('map scale', { scale });
  };
  const showOthers = (cells: readonly TilePoint[]) => {
    while (others.length < cells.length) others.push(markImage(dom.marks, deps.art.other, 'map-other'));
    others.forEach((dot, i) => {
      const cell = cells[i];
      dot.hidden = !cell;
      if (cell) placeAt(dot, rectangleMark(cell, scale));
    });
  };
  const showYou = (cell: TilePoint | null) => {
    dom.you.hidden = dom.donutMark.hidden = !cell;
    if (!cell) return;
    placeAt(dom.you, rectangleMark(cell, scale));
    const mark = donutMark(cell, grid, deps.art.ring);
    setSource(dom.donutMark, deps.art.you[mark.shape]);
    placeAt(dom.donutMark, { x: mark.at.x * deps.art.scale, y: mark.at.y * deps.art.scale });
  };
  const showExplored = (explored: Exploration) => {
    if (explored === shownExplored) return;
    shownExplored = explored;
    drawVeil(dom.veil, explored);
    dom.explored.textContent = exploredText(exploredShare(explored, grid));
  };

  const screen: MapScreen = {
    open(view) {
      loadImage(dom, deps.imageUrl, now, log);
      dom.backdrop.hidden = false;
      screen.update(view);
      log('opened', { explored: dom.explored.textContent });
    },
    update(view) {
      fit();
      dom.region.textContent = view.regionName ?? '';
      showExplored(view.explored);
      showOthers(view.others);
      showYou(view.you);
      dom.block.classList.toggle('map-blink-off', !blinkOn(now()));
    },
    close() {
      dom.backdrop.hidden = true;
      log('closed', { explored: dom.explored.textContent });
    },
    isOpen: () => !dom.backdrop.hidden,
  };
  return screen;
}

interface MapScreenDom {
  backdrop: HTMLElement;
  block: HTMLElement;
  region: HTMLElement;
  donutMark: HTMLImageElement;
  image: HTMLImageElement;
  veil: HTMLCanvasElement;
  marks: HTMLElement;
  you: HTMLImageElement;
  explored: HTMLElement;
}

/**
 * The screen, hidden: the map's block, holding the map, its veil and marks, and over them the names (top center),
 * the donut (top right) and the explored share (bottom right).
 */
function buildScreen(host: HTMLElement, map: MapDescription, art: MapArt, grid: Grid): MapScreenDom {
  const backdrop = element('div', 'map-backdrop', host);
  backdrop.hidden = true;
  const block = element('div', 'map-block', backdrop);
  const mapImage = image(block, '', 'map-image');
  mapImage.alt = `Map of ${map.name}`;
  const veil = element('canvas', 'map-veil', block);
  veil.width = Math.ceil(grid.width / CHUNK_SIZE);
  veil.height = Math.ceil(grid.height / CHUNK_SIZE);
  const marks = element('div', 'map-marks', block);
  const you = markImage(marks, art.you.square, 'map-you');
  element('div', 'map-error', block).textContent = "Couldn't load the map.";
  const titles = element('div', 'map-titles map-over', block);
  element('div', 'map-name', titles).textContent = map.name;
  const region = element('div', 'map-region', titles);
  const donutBox = element('div', 'map-donut', block);
  donutBox.hidden = !grid.wraps;
  image(donutBox, art.donut, 'map-donut-image');
  const donutMark = markImage(donutBox, art.you.square, 'map-you');
  const explored = element('div', 'map-explored map-over', block);
  return { backdrop, block, region, donutMark, image: mapImage, veil, marks, you, explored };
}

/** Asks for the map's image the first time the map opens; on failure, the map gives way to a message. */
function loadImage(dom: MapScreenDom, url: string, now: () => number, log: Log): void {
  if (dom.image.getAttribute('src')) return;
  const start = now();
  dom.image.addEventListener('load', () => {
    log('image loaded', { width: dom.image.naturalWidth, height: dom.image.naturalHeight, took: Math.round(now() - start) });
  });
  dom.image.addEventListener('error', () => {
    log('image failed', url);
    dom.block.classList.add('map-failed');
  });
  dom.image.src = url;
}

/** One veil pixel per chunk, scaled up crisp: dark where you haven't been, clear where you have. */
function drawVeil(veil: HTMLCanvasElement, explored: Exploration): void {
  const context = veil.getContext('2d');
  if (!context) return;
  context.clearRect(0, 0, veil.width, veil.height);
  context.fillStyle = VEIL;
  for (let cy = 0; cy < veil.height; cy += 1) {
    for (let cx = 0; cx < veil.width; cx += 1) {
      if (!isExplored(explored, { cx, cy })) context.fillRect(cx, cy, 1, 1);
    }
  }
}

function element<K extends keyof HTMLElementTagNameMap>(tag: K, className: string, parent: HTMLElement): HTMLElementTagNameMap[K] {
  const created = document.createElement(tag);
  created.className = className;
  parent.appendChild(created);
  return created;
}

function image(parent: HTMLElement, src: string, className: string): HTMLImageElement {
  const created = element('img', className, parent);
  created.alt = '';
  if (src) created.src = src;
  return created;
}

function markImage(parent: HTMLElement, src: string, className: string): HTMLImageElement {
  return image(parent, src, `map-mark ${className}`);
}

/** A mark centered on a point of its layer. */
function placeAt(mark: HTMLElement, point: ScreenPoint): void {
  mark.style.left = `${point.x}px`;
  mark.style.top = `${point.y}px`;
}

function setSource(img: HTMLImageElement, src: string): void {
  if (img.getAttribute('src') !== src) img.src = src;
}

function sizeTo(element: HTMLElement, grid: Grid, scale: number): void {
  element.style.width = `${grid.width * scale}px`;
  element.style.height = `${grid.height * scale}px`;
}

import { hashCell, type Rng } from '../core/rng';
import { DOME_CELLS } from '../map/worldMap';
import { isShapeEdge, paintGroundShadow } from './drawing';
import { createPixelBuffer, forEachPixel, setPixel, type PixelBuffer } from './pixelBuffer';
import { groundCenter, isDrawable, textureSize, type TileFrame } from './tileFrame';

/** The dome's shape, in cells: a glass half-sphere on a low metal ring, both centered on the footprint's middle. */
const SHAPE = { glassRadius: 1.0, ringRadius: 1.08, ringHeight: 0.16 };
/** The honeycomb: the side of a hex cell, in glass radii, on the dome flattened from its top (equal-area). */
const HEX_SIZE = 0.3;
/** Room above the footprint's ground for the dome, in art pixels, for a 32 px wide tile. */
const HEADROOM_PX = 14;

const COLORS = {
  outline: 0x3d3646,
  /** Faded teal glass, from lit to dark: a pixel takes the first shade its light reaches, else the darkest. */
  glass: [
    { minLight: 0.96, rgb: 0xe3f5ef },
    { minLight: 0.76, rgb: 0xa2d5cd },
    { minLight: 0.5, rgb: 0x80bdb8 },
    { minLight: 0.2, rgb: 0x66a1a2 },
  ],
  glassDarkest: 0x52868c,
  crack: 0xe3f5ef,
  /** A missing cell: the dark inside, and the faint roofs seen through its lower part. */
  inside: 0x2f2838,
  roof: 0x463c4e,
  roofEdge: 0x5e5266,
  frame: { lit: 0x6d6876, dark: 0x524d5d },
  rust: [0xa5633f, 0x86513a],
  ring: { top: 0xa7a3ad, lit: 0x8e8a96, dark: 0x6c6878 },
  moss: [0x7f9f5a, 0x62814a],
} as const;

/** How often a hex cell is missing or cracked, rust starts on the frame, and moss and grime gather at the base. */
const WEAR = {
  missing: 0.16,
  cracked: 0.2,
  rust: 0.08,
  rustLength: [2, 4],
  /** Moss grows in clumps this many pixels wide along the ring, on this share of them. */
  mossClumpPx: 3,
  moss: 0.45,
  /** The share of glass cells grimy near the base: below this height above the ring, in glass radii. */
  grimy: 0.5,
  baseElevation: 0.3,
} as const;

/** Unit vector toward the light, in screen space: from the top-left, facing the viewer (as for rocks). */
const LIGHT = { x: -0.45, y: -0.65, z: 0.61 };

type Part = 'glass' | 'ringSide' | 'ringTop';

/** What the view ray of a pixel meets first on the dome: where, in cells from the footprint's middle, and its normal. */
interface Hit {
  part: Part;
  point: Vec3;
  normal: Vec3;
}

interface Vec3 {
  a: number;
  b: number;
  c: number;
}

/** The hex cell of the honeycomb a glass point is on. */
interface HexCell {
  q: number;
  r: number;
}

/** The frame of a dome's sprite: its 2 × 2 footprint's ground diamond, with headroom above for the dome. */
export function domeFrame(tileWidth: number, tileHeight: number): TileFrame {
  return { tileWidth: tileWidth * DOME_CELLS, tileHeight: tileHeight * DOME_CELLS, headroom: Math.round((HEADROOM_PX * tileWidth) / 32) };
}

/**
 * An old dome of hexagonal glass cells on a metal ring (ADR 0012): faded teal glass lit from the top left, some
 * cells missing and showing the dark inside, some cracked, rust running down the frame, moss and grime at the base,
 * the colored dark outline and the ground shadow. Which cells are worn comes from `rng`.
 */
export function paintDome(frame: TileFrame, rng: Rng): PixelBuffer {
  const { width, height } = textureSize(frame);
  const buffer = createPixelBuffer(width, height);
  const salt = Math.floor(rng() * 0x7fffffff);
  const geometry = domeGeometry(frame);
  const samples = sampleAll(castAll(frame, geometry), width);
  paintGroundShadow(buffer, frame, shadowOf(frame, geometry));
  forEachPixel(width, height, (x, y) => {
    const sample = samples[y * width + x];
    if (sample) setPixel(buffer, x, y, surfaceColor(sample, salt));
  });
  paintRoofs(buffer, samples, width, salt);
  paintRust(buffer, samples, width, salt);
  paintMoss(buffer, samples, width, salt);
  paintOutline(buffer, samples, width);
  return buffer;
}

/** A pixel of the dome: what its ray hit, the hex cell for glass, and whether it's on the honeycomb's frame. */
interface Sample {
  hit: Hit;
  cell: HexCell | null;
  frame: boolean;
}

function sampleAll(hits: readonly (Hit | null)[], width: number): (Sample | null)[] {
  const cells = hits.map((hit) => (hit?.part === 'glass' ? hexCellOf(hit.point) : null));
  return hits.map((hit, i) => (hit ? { hit, cell: cells[i] ?? null, frame: isFramePixel(cells, i, width) } : null));
}

/** A frame pixel is glass whose neighbor to the right or below is on another hex cell: lines one pixel thick. */
function isFramePixel(cells: readonly (HexCell | null)[], i: number, width: number): boolean {
  const cell = cells[i];
  if (!cell) return false;
  const right = (i + 1) % width === 0 ? null : cells[i + 1];
  return [right, cells[i + width]].some((next) => next && (next.q !== cell.q || next.r !== cell.r));
}

/** How cells map to texture pixels: the footprint's middle, pixels per cell across, down the screen, and up. */
interface Geometry {
  center: { x: number; y: number };
  halfWidth: number;
  halfHeight: number;
  /** Art pixels per cell of height: a true 2:1 dimetric view, so the half-sphere looks round. */
  upPx: number;
}

function domeGeometry(frame: TileFrame): Geometry {
  const halfWidth = frame.tileWidth / DOME_CELLS / 2;
  return { center: groundCenter(frame), halfWidth, halfHeight: frame.tileHeight / DOME_CELLS / 2, upPx: halfWidth * Math.sqrt(1.5) };
}

/** Each pixel's hit, where the sprite may draw only: never over the cells in front of its footprint. */
function castAll(frame: TileFrame, geometry: Geometry): (Hit | null)[] {
  const { width, height } = textureSize(frame);
  const hits: (Hit | null)[] = [];
  forEachPixel(width, height, (x, y) => hits.push(isDrawable(frame, x, y) ? castRay(x, y, geometry) : null));
  return hits;
}

/**
 * The first point of the dome the view ray through pixel (x, y) meets, or null. The ray goes through the ground point
 * under the pixel and toward the viewer along (1, 1, k): one cell toward the front for every k cells up.
 */
function castRay(x: number, y: number, geometry: Geometry): Hit | null {
  const across = (x + 0.5 - geometry.center.x) / geometry.halfWidth;
  const down = (y + 0.5 - geometry.center.y) / geometry.halfHeight;
  const ground = { a: (across + down) / 2, b: (down - across) / 2 };
  const k = (2 * geometry.halfHeight) / geometry.upPx;
  const hits = [hitGlass(ground, k), hitRingSide(ground, k), hitRingTop(ground, k)].filter((hit) => hit !== null);
  // the hit nearest the viewer is the one furthest along the ray
  return hits.sort((p, q) => q.t - p.t)[0]?.hit ?? null;
}

interface RayHit {
  t: number;
  hit: Hit;
}

type Ground = { a: number; b: number };

function along(ground: Ground, k: number, t: number): Vec3 {
  return { a: ground.a + t, b: ground.b + t, c: k * t };
}

/** The upper half of the glass sphere, standing on the ring. */
function hitGlass(ground: Ground, k: number): RayHit | null {
  const { glassRadius: radius, ringHeight: base } = SHAPE;
  const t = frontRoot(2 + k * k, 2 * (ground.a + ground.b) - 2 * k * base, ground.a ** 2 + ground.b ** 2 + base ** 2 - radius ** 2);
  if (t === null) return null;
  const point = along(ground, k, t);
  if (point.c < base) return null;
  return { t, hit: { part: 'glass', point, normal: { a: point.a / radius, b: point.b / radius, c: (point.c - base) / radius } } };
}

function hitRingSide(ground: Ground, k: number): RayHit | null {
  const radius = SHAPE.ringRadius;
  const t = frontRoot(2, 2 * (ground.a + ground.b), ground.a ** 2 + ground.b ** 2 - radius ** 2);
  if (t === null) return null;
  const point = along(ground, k, t);
  if (point.c < 0 || point.c > SHAPE.ringHeight) return null;
  return { t, hit: { part: 'ringSide', point, normal: { a: point.a / radius, b: point.b / radius, c: 0 } } };
}

function hitRingTop(ground: Ground, k: number): RayHit | null {
  const t = SHAPE.ringHeight / k;
  const point = along(ground, k, t);
  if (point.a ** 2 + point.b ** 2 > SHAPE.ringRadius ** 2) return null;
  return { t, hit: { part: 'ringTop', point, normal: { a: 0, b: 0, c: 1 } } };
}

/** The larger root of A t² + B t + C, the side facing the viewer; null when the ray misses. */
function frontRoot(a: number, b: number, c: number): number | null {
  const disc = b * b - 4 * a * c;
  return disc < 0 ? null : (-b + Math.sqrt(disc)) / (2 * a);
}

function surfaceColor({ hit, cell, frame }: Sample, salt: number): number {
  if (!cell) return hit.part === 'ringTop' ? COLORS.ring.top : lit(hit) > 0.3 ? COLORS.ring.lit : COLORS.ring.dark;
  if (frame) return lit(hit) > 0.45 ? COLORS.frame.lit : COLORS.frame.dark;
  const wear = wearOf(cell, salt);
  if (wear === 'missing') return COLORS.inside;
  if (wear === 'cracked' && isOnCrack(hit.point, cell, salt)) return COLORS.crack;
  return glassShade(lit(hit) - (isGrimy(hit, cell, salt) ? 0.3 : 0));
}

/** Some glass cells near the base are grimy: one shade darker there, in patches rather than speckles. */
function isGrimy(hit: Hit, cell: HexCell, salt: number): boolean {
  return hit.point.c - SHAPE.ringHeight < WEAR.baseElevation && hashCell(cell.q, cell.r, salt + 1) / 0x100000000 < WEAR.grimy;
}

/** Through the lower part of each missing cell, on screen, the faint roofs of the city: a lit edge, then roofs. */
function paintRoofs(buffer: PixelBuffer, samples: readonly (Sample | null)[], width: number, salt: number): void {
  const isHole = (i: number) => {
    const sample = samples[i];
    return !!sample?.cell && !sample.frame && wearOf(sample.cell, salt) === 'missing';
  };
  forEachPixel(width, buffer.height, (x, y) => {
    if (!isHole(y * width + x)) return;
    let below = 0;
    while (below < 3 && y + below + 1 < buffer.height && isHole((y + below + 1) * width + x)) below++;
    if (below < 2) setPixel(buffer, x, y, COLORS.roof);
    else if (below === 2) setPixel(buffer, x, y, COLORS.roofEdge);
  });
}

/** Moss in clumps along the ring, a few pixels wide, sometimes creeping one pixel up the glass. */
function paintMoss(buffer: PixelBuffer, samples: readonly (Sample | null)[], width: number, salt: number): void {
  forEachPixel(width, buffer.height, (x, y) => {
    const sample = samples[y * width + x];
    if (!sample || sample.cell || hashCell(Math.floor(x / WEAR.mossClumpPx), 0, salt + 2) / 0x100000000 >= WEAR.moss) return;
    setPixel(buffer, x, y, sample.hit.part === 'ringTop' ? COLORS.moss[0] : COLORS.moss[1]);
    const above = samples[(y - 1) * width + x];
    if (y > 0 && above?.cell && chance(x, y, salt + 6) < 0.5) setPixel(buffer, x, y - 1, COLORS.moss[0]);
  });
}

type Wear = 'missing' | 'cracked' | 'whole';

export function wearOf(cell: HexCell, salt: number): Wear {
  const roll = hashCell(cell.q, cell.r, salt) / 0x100000000;
  if (roll < WEAR.missing) return 'missing';
  return roll < WEAR.missing + WEAR.cracked ? 'cracked' : 'whole';
}

/** A crack is a line through the cell's middle, in one of three directions. */
function isOnCrack(point: Vec3, cell: HexCell, salt: number): boolean {
  const local = hexLocal(point, cell);
  const angle = ((hashCell(cell.q, cell.r, salt + 3) % 3) * Math.PI) / 3 + 0.4;
  return Math.abs(local.x * Math.sin(angle) - local.y * Math.cos(angle)) < HEX_SIZE * 0.09;
}

/** The dome flattened from its top, keeping areas: a regular honeycomb there wraps the half-sphere evenly. */
function flattened(point: Vec3): { x: number; y: number } {
  const up = Math.min(1, Math.max(0, (point.c - SHAPE.ringHeight) / SHAPE.glassRadius));
  const scale = Math.sqrt(2 / (1 + up)) / SHAPE.glassRadius;
  return { x: point.a * scale, y: point.b * scale };
}

/** The pointy-top hex cell of the honeycomb a glass point is on (axial coordinates, rounded through cube ones). */
export function hexCellOf(point: Vec3): HexCell {
  const { x, y } = flattened(point);
  const q = ((Math.sqrt(3) / 3) * x - y / 3) / HEX_SIZE;
  const r = ((2 / 3) * y) / HEX_SIZE;
  return roundHex(q, r);
}

function roundHex(q: number, r: number): HexCell {
  const s = -q - r;
  let [rq, rr, rs] = [Math.round(q), Math.round(r), Math.round(s)];
  const [dq, dr, ds] = [Math.abs(rq - q), Math.abs(rr - r), Math.abs(rs - s)];
  if (dq > dr && dq > ds) rq = -rr - rs;
  else if (dr > ds) rr = -rq - rs;
  return { q: rq, r: rr };
}

/** A glass point relative to its hex cell's middle, on the flattened dome. */
function hexLocal(point: Vec3, cell: HexCell): { x: number; y: number } {
  const { x, y } = flattened(point);
  return { x: x - HEX_SIZE * Math.sqrt(3) * (cell.q + cell.r / 2), y: y - HEX_SIZE * 1.5 * cell.r };
}

/** How much a surface faces the light, through its normal seen on screen. */
function lit(hit: Hit): number {
  const { a, b, c } = hit.normal;
  const right = (a - b) / Math.SQRT2;
  const up = (-0.354 * (a + b) + 0.866 * c) / Math.hypot(0.354, 0.354, 0.866);
  const toward = (0.612 * (a + b) + 0.5 * c) / Math.hypot(0.612, 0.612, 0.5);
  return right * LIGHT.x - up * LIGHT.y + toward * LIGHT.z;
}

function glassShade(light: number): number {
  return COLORS.glass.find((shade) => light >= shade.minLight)?.rgb ?? COLORS.glassDarkest;
}

/** Rust streaks: from a few frame pixels, 2 to 4 pixels running down the dome. */
function paintRust(buffer: PixelBuffer, samples: readonly (Sample | null)[], width: number, salt: number): void {
  const onGlass = (x: number, y: number) => samples[y * width + x]?.cell != null;
  forEachPixel(width, buffer.height, (x, y) => {
    if (!samples[y * width + x]?.frame || chance(x, y, salt + 4) >= WEAR.rust) return;
    const [shortest, longest] = WEAR.rustLength;
    const length = shortest + (hashCell(x, y, salt + 5) % (longest - shortest + 1));
    for (let step = 0; step < length && y + step < buffer.height && onGlass(x, y + step); step++) {
      setPixel(buffer, x, y + step, step === 0 ? COLORS.rust[1] : COLORS.rust[0]);
    }
  });
}

function paintOutline(buffer: PixelBuffer, samples: readonly (Sample | null)[], width: number): void {
  const inside = (x: number, y: number) => x >= 0 && y >= 0 && x < width && y < buffer.height && samples[y * width + x] != null;
  forEachPixel(width, buffer.height, (x, y) => {
    if (isShapeEdge(inside, x, y)) setPixel(buffer, x, y, COLORS.outline);
  });
}

/** The shadow on the ground, a little to the bottom right: the light comes from the top left. */
function shadowOf(frame: TileFrame, geometry: Geometry) {
  const center = groundCenter(frame);
  const rx = SHAPE.ringRadius * geometry.halfWidth * Math.SQRT2 + 3;
  return { cx: center.x + 3, cy: center.y + 2, rx, ry: rx / 2 };
}

function chance(x: number, y: number, salt: number): number {
  return hashCell(x, y, salt) / 0x100000000;
}

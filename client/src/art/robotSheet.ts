import { isInt, isNonEmptyString, isRecord } from '../net/validate';

/** A robot's cell in the sheet, by its top-left corner (ADR 0006). */
export interface RobotCell {
  slug: string;
  x: number;
  y: number;
}

/** `robots.json`: the cells' size, and the robots in the sheet's order. The first one is the default robot. */
export interface RobotSheet {
  cell: number;
  robots: readonly [RobotCell, ...RobotCell[]];
}

export interface SheetSize {
  width: number;
  height: number;
}

/** Reads `robots.json`, checking each cell is inside the sheet's PNG. Null when it's malformed. */
export function parseRobotSheet(json: unknown, size: SheetSize): RobotSheet | null {
  if (!isRecord(json)) return null;
  const { cell, sprites } = json;
  if (!isInt(cell) || cell <= 0 || !Array.isArray(sprites)) return null;
  const robots: RobotCell[] = [];
  for (const sprite of sprites) {
    const robot = robotCell(sprite, cell, size);
    if (!robot || robots.some((other) => other.slug === robot.slug)) return null;
    robots.push(robot);
  }
  const [first, ...rest] = robots;
  return first ? { cell, robots: [first, ...rest] } : null;
}

function robotCell(value: unknown, cell: number, size: SheetSize): RobotCell | null {
  if (!isRecord(value) || !isNonEmptyString(value.slug) || !isInt(value.x) || !isInt(value.y)) return null;
  const { slug, x, y } = value;
  const inside = x >= 0 && y >= 0 && x + cell <= size.width && y + cell <= size.height;
  return inside ? { slug, x, y } : null;
}

/** The robot drawn for a slug: its own, or the first one for an unknown or empty slug. */
export function robotToDraw<T extends { slug: string }>(robots: readonly [T, ...T[]], slug: string): T {
  return robots.find((robot) => robot.slug === slug) ?? robots[0];
}

import { Rectangle, Texture } from 'pixi.js';
import { silent, type Log } from '../debug/log';
import { parseRobotSheet, robotToDraw, type RobotCell } from '../art/robotSheet';

/** Where the robot sheet is served from (ADR 0006). */
export const ROBOT_SHEET_URLS = { png: '/sprites/robots.png', json: '/sprites/robots.json' } as const;

export interface RobotTextures {
  /** Every robot's slug, the default one first. */
  slugs: readonly string[];
  /** The texture drawn for a player's robot: the default one for an unknown or empty slug. */
  textureOf(slug: string): Texture;
}

export interface RobotSheetSource {
  loadTexture(url: string): Promise<Texture>;
  fetchJson(url: string): Promise<unknown>;
}

export interface RobotTexturesDeps {
  source: RobotSheetSource;
  urls?: { png: string; json: string };
  now?: () => number;
  log?: Log;
}

/** Loads the robot sheet and cuts one crisp texture per robot. Rejects when a file fails or the JSON is malformed. */
export async function loadRobotTextures({ source, urls = ROBOT_SHEET_URLS, now = () => performance.now(), log = silent }: RobotTexturesDeps): Promise<RobotTextures> {
  const started = now();
  const [sheet, json] = await Promise.all([source.loadTexture(urls.png), source.fetchJson(urls.json)]);
  const parsed = parseRobotSheet(json, { width: sheet.width, height: sheet.height });
  if (!parsed) throw new Error(`loadRobotTextures: ${urls.json} is malformed`);
  sheet.source.scaleMode = 'nearest';
  const withTexture = (robot: RobotCell) => ({ slug: robot.slug, texture: cut(sheet, robot, parsed.cell) });
  const [first, ...rest] = parsed.robots;
  const robots = [withTexture(first), ...rest.map(withTexture)] as const;
  log('robot sheet loaded', { robots: robots.length, ms: Math.round(now() - started) });
  return {
    slugs: robots.map((robot) => robot.slug),
    textureOf(slug) {
      const robot = robotToDraw(robots, slug);
      if (robot.slug !== slug) log('default robot drawn: unknown slug', { slug, drawn: robot.slug });
      return robot.texture;
    },
  };
}

function cut(sheet: Texture, robot: RobotCell, cell: number): Texture {
  return new Texture({ source: sheet.source, frame: new Rectangle(robot.x, robot.y, cell, cell) });
}

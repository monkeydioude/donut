import '@fontsource/zen-maru-gothic/500.css';
import '@fontsource/zen-maru-gothic/700.css';
import { Application } from 'pixi.js';
import { FINGER_HOTSPOT, paintFinger } from './art/finger';
import { createGeneratedArt } from './art/tileArtSet';
import type { TileFrame } from './art/tileFrame';
import { paintTileOutline } from './art/tileOutline';
import { createDefaultConfig, type GameConfig } from './config';
import { createSeededRng } from './core/rng';
import { createLogger, type Log, type Logger } from './debug/log';
import { startPlayersSession, type CommandMenu, type MenuChoices, type PlayersSession } from './game/session';
import { createIsoProjection, type IsoProjection } from './iso/projection';
import { nearestCopy, type Grid } from './map/grid';
import { gridOf, type MapDescription } from './map/worldMap';
import { openGameSocket } from './net/gameSocket';
import { browserStore, loadGuest } from './net/guest';
import { fetchDescription } from './net/mapApi';
import { socketUrl } from './net/protocol';
import { cellAt, followPosition, tileOnCanvas, type WorldPlacement } from './render/camera';
import { createChunksView, type Bake } from './render/chunkView';
import { createWorldView, type WorldView } from './render/mapView';
import { createOverlay } from './render/overlay';
import { createPlayersView, type PlayersView } from './render/playersView';
import { createWorldGraphics, spriteGraphic, type WorldGraphics } from './render/tileGraphics';
import { createRegionBanner } from './ui/banner';
import { spriteCursor } from './ui/cursor';
import { createActionMenu } from './ui/menu';
import { showNotice } from './ui/notice';
import { listenToPointer } from './ui/pointer';

interface World {
  map: MapDescription;
  grid: Grid;
  projection: IsoProjection;
  graphics: WorldGraphics;
  view: WorldView;
}

const BORDERS = { hover: { rgb: 0xffffff, alpha: 220 }, selection: { rgb: 0xffd84a, alpha: 255 } } as const;

async function start(config: GameConfig, host: HTMLElement): Promise<void> {
  const logger = createLogger(config.debug);
  const log = logger.scope('boot');
  log('starting', config);
  host.style.cursor = spriteCursor(paintFinger(), FINGER_HOTSPOT, config.cursorScale);
  log('finger cursor set', { scale: config.cursorScale, hotspot: FINGER_HOTSPOT });
  const app = await createApp(config, host);
  log('pixi ready', { renderer: app.renderer.name, screen: { width: app.screen.width, height: app.screen.height }, resolution: app.renderer.resolution });
  let map: MapDescription;
  try {
    map = await fetchDescription(config.mapSlug, (url) => fetch(url), logger.scope('map'));
  } catch (error) {
    console.error(error);
    showNotice(host, 'Could not load the map. Is the back running? (make dev)');
    return;
  }
  const world = createWorld(config, map);
  log('world ready, waiting for its chunks', { size: `${map.width}x${map.height}`, wraps: map.wraps, regions: map.regions.length });
  app.stage.addChild(world.view.root);
  play(config, host, app, world, logger);
}

async function createApp(config: GameConfig, host: HTMLElement): Promise<Application> {
  const app = new Application();
  await app.init({
    resizeTo: window,
    background: config.background,
    antialias: false,
    roundPixels: true,
    resolution: window.devicePixelRatio,
    autoDensity: true,
  });
  host.appendChild(app.canvas);
  return app;
}

function createWorld(config: GameConfig, map: MapDescription): World {
  const rng = createSeededRng(config.seed);
  const projection = createIsoProjection(config.tile.width, config.tile.height);
  const artSet = createGeneratedArt({
    tileWidth: config.tile.width,
    tileHeight: config.tile.height,
    rockHeadroom: config.rockHeadroom,
    playerHeadroom: config.playerHeadroom,
  });
  const graphics = createWorldGraphics({ artSet, variantsPerKind: config.variantsPerKind, rng });
  const view = createWorldView();
  view.root.scale.set(config.zoom);
  return { map, grid: gridOf(map), projection, graphics, view };
}

/** Draws a chunk's ground into one texture at art resolution, crisp, then drops the container it came from. */
function bakeWith(app: Application): Bake {
  return (container) => {
    const bounds = container.getLocalBounds();
    const texture = app.renderer.generateTexture({
      target: container,
      resolution: 1,
      antialias: false,
      textureSourceOptions: { scaleMode: 'nearest' },
    });
    container.destroy({ children: true });
    return { texture, x: bounds.x, y: bounds.y };
  };
}

/** Every frame, the world is placed so your player, as drawn, is in the middle of the screen. */
function followCamera(app: Application, world: World, zoom: number, players: PlayersView, session: PlayersSession, log: Log): void {
  app.ticker.add(() => {
    const target = players.ownPosition();
    if (!target) return;
    const origin = followPosition(target, app.screen, zoom);
    world.view.root.position.set(origin.x, origin.y);
    session.followCamera();
  });
  app.renderer.on('resize', () => {
    log('viewport resized', { width: app.screen.width, height: app.screen.height });
    session.refresh();
  });
}

function play(config: GameConfig, host: HTMLElement, app: Application, world: World, logger: Logger): void {
  const placement = (): WorldPlacement => ({ origin: world.view.root.position, zoom: config.zoom });
  const frame: TileFrame = { tileWidth: config.tile.width, tileHeight: config.tile.height, headroom: 0 };
  const guest = loadGuest({
    store: browserStore(),
    newId: () => crypto.randomUUID(),
    randomInt: (max) => Math.floor(Math.random() * max),
    log: logger.scope('guest'),
  });
  const players = createPlayersView({
    layer: world.view.entities,
    projection: world.projection,
    looks: world.graphics.players,
    ticker: app.ticker,
    grid: world.grid,
    log: logger.scope('players'),
  });
  // a cell within the map, drawn at its copy nearest to your player
  const drawn = (cell: { col: number; row: number }) => {
    const anchor = players.ownCell();
    return anchor ? nearestCopy(world.grid, cell, anchor) : cell;
  };
  const session = startPlayersSession({
    map: world.map,
    socketUrl: socketUrl(window.location, world.map.slug, guest),
    openSocket: (url, handlers) => openGameSocket(url, handlers, logger.scope('socket')),
    listenToPointer: (handlers) =>
      listenToPointer(app.canvas, (point) => cellAt(point, placement(), world.projection, world.grid), handlers, logger.scope('pointer')),
    createMenu: (choices) =>
      commandMenu(host, app, choices, (cell) => tileOnCanvas(drawn(cell), placement(), world.projection, config.tile.width), logger.scope('menu')),
    players,
    chunks: createChunksView({
      ground: world.view.ground,
      entities: world.view.entities,
      projection: world.projection,
      graphics: world.graphics,
      grid: world.grid,
      bake: bakeWith(app),
      anchor: () => players.ownCell(),
      log: logger.scope('chunks'),
    }),
    overlay: createOverlay({
      layer: world.view.overlay,
      projection: world.projection,
      borders: {
        hover: spriteGraphic(paintTileOutline(frame, BORDERS.hover.rgb, BORDERS.hover.alpha), frame),
        selection: spriteGraphic(paintTileOutline(frame, BORDERS.selection.rgb, BORDERS.selection.alpha), frame),
      },
      ticker: app.ticker,
      pathStart: () => players.ownPosition(),
      anchor: () => players.ownCell(),
      grid: world.grid,
      log: logger.scope('overlay'),
    }),
    banner: createRegionBanner(host, logger.scope('banner')),
    onDisconnect: () => showNotice(host, 'Disconnected. Reload the page to come back.'),
    log: logger.scope('session'),
  });
  followCamera(app, world, config.zoom, players, session, logger.scope('camera'));
}

/** The action menu of a selected tile: Move and Cancel, beside the tile. */
function commandMenu(
  host: HTMLElement,
  app: Application,
  choices: MenuChoices,
  anchorOf: (cell: { col: number; row: number }) => { left: number; right: number; y: number },
  log: Log,
): CommandMenu {
  const menu = createActionMenu(host, () => app.screen, log);
  return {
    show: (cell, moveEnabled) =>
      menu.show(anchorOf(cell), [
        { label: 'Move', enabled: moveEnabled, choose: choices.onMove },
        { label: 'Cancel', enabled: true, choose: choices.onCancel },
      ]),
    hide: () => menu.hide(),
  };
}

start(createDefaultConfig(), document.body);

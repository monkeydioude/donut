import '@fontsource/zen-maru-gothic/500.css';
import '@fontsource/zen-maru-gothic/700.css';
import { Application, Assets, type Texture } from 'pixi.js';
import { paintCameraIcon } from './art/cameraIcon';
import { DONUT_MARK_RING, paintDonut } from './art/donut';
import { FINGER_HOTSPOT, paintFinger } from './art/finger';
import { paintOtherMark, paintYouMark } from './art/mapMarks';
import type { PixelBuffer } from './art/pixelBuffer';
import { paintPlayerShadow, paintYourRing } from './art/playerMarks';
import { createGeneratedArt } from './art/tileArtSet';
import type { TileFrame } from './art/tileFrame';
import { paintTileOutline } from './art/tileOutline';
import { createDefaultConfig, type GameConfig } from './config';
import { createSeededRng } from './core/rng';
import { browserStore, type KeyValueStore } from './core/store';
import { createLogger, type Log, type Logger } from './debug/log';
import { createCamera, initialCamera, type Camera } from './game/camera';
import { loadCameraLock, saveCameraLock } from './game/cameraSettings';
import { explore, noExploration, type Exploration } from './game/exploration';
import { startScreens, type MapWindow, type WorldmapHandlers } from './game/screens';
import { startPlayersSession, type CommandMenu, type MenuChoices, type PlayersSession } from './game/session';
import { browserPlatform, listenToControls } from './input/controls';
import type { Heights } from './iso/heights';
import { createIsoProjection, type IsoProjection } from './iso/projection';
import { emptyChunks, heightForDrawing, type HeldChunksRef } from './map/chunks';
import { nearestCopy, type ChunkKey, type Grid } from './map/grid';
import { gridOf, regionName, type MapDescription } from './map/worldMap';
import { openGameSocket } from './net/gameSocket';
import { loadGuest } from './net/guest';
import { fetchDescription, mapImageUrl } from './net/mapApi';
import { socketUrl } from './net/protocol';
import { cellAt, tileOnCanvas, type WorldPlacement } from './render/camera';
import { createChunksView, type Bake, type ChunksView } from './render/chunkView';
import { createWorldView, type WorldView } from './render/mapView';
import { createOverlay } from './render/overlay';
import { createPlayersView, type PlayerGraphics, type PlayersView } from './render/playersView';
import { loadRobotTextures, type RobotTextures } from './render/robotTextures';
import { createWorldGraphics, spriteGraphic, type WorldGraphics } from './render/tileGraphics';
import { createRegionBanner } from './ui/banner';
import { createCameraButton } from './ui/cameraButton';
import { spriteCursor, spriteDataUrl } from './ui/cursor';
import { createMapScreen, type MapArt } from './ui/mapScreen';
import { createChatbox } from './ui/chatbox';
import { createCoordinatesLabel } from './ui/coordinates';
import { createActionMenu } from './ui/menu';
import { showNotice } from './ui/notice';
import { listenToPointer } from './ui/pointer';

interface World {
  map: MapDescription;
  grid: Grid;
  projection: IsoProjection;
  /** How heights are drawn (ADR 0007). */
  heights: Heights;
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
  const [map, robots] = await Promise.allSettled([
    fetchDescription(config.mapSlug, (url) => fetch(url), logger.scope('map')),
    loadRobotTextures({ source: robotSheetSource(), log: logger.scope('robots') }),
  ]);
  if (map.status === 'rejected') {
    console.error(map.reason);
    showNotice(host, 'Could not load the map. Is the back running? (make dev)');
    return;
  }
  if (robots.status === 'rejected') {
    console.error(robots.reason);
    showNotice(host, 'Could not load the sprites.');
    return;
  }
  const world = createWorld(config, map.value);
  log('world ready, waiting for its chunks', { size: `${map.value.width}x${map.value.height}`, wraps: map.value.wraps, regions: map.value.regions.length });
  app.stage.addChild(world.view.root);
  play(config, host, app, world, robots.value, logger);
}

/** The robot sheet from the page's own host: the PNG through Pixi, the JSON through fetch. */
function robotSheetSource() {
  return {
    loadTexture: (url: string) => Assets.load<Texture>(url),
    fetchJson: async (url: string) => {
      const response = await fetch(url);
      if (!response.ok) throw new Error(`${url}: ${response.status}`);
      return (await response.json()) as unknown;
    },
  };
}

/** Your ring and the shadow, painted on a tile's ground; the robots from the sheet. */
function playerGraphics(frame: TileFrame, robots: RobotTextures): PlayerGraphics {
  return {
    shadow: spriteGraphic(paintPlayerShadow(frame), frame),
    ring: spriteGraphic(paintYourRing(frame), frame),
    robotOf: (slug) => robots.textureOf(slug),
  };
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
    levelPx: config.levelHeight,
  });
  const graphics = createWorldGraphics({ artSet, variantsPerKind: config.variantsPerKind, rng });
  const view = createWorldView();
  view.root.scale.set(config.camera.zoom.start);
  return { map, grid: gridOf(map), projection, heights: { projection, levelPx: config.levelHeight }, graphics, view };
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

/** Every frame, the world is placed and scaled where the camera says: on your player when it's locked. */
function driveCamera(app: Application, world: World, camera: Camera, session: PlayersSession, log: Log): void {
  app.ticker.add(() => {
    const placement = camera.place();
    if (!placement) return;
    world.view.root.position.set(placement.origin.x, placement.origin.y);
    world.view.root.scale.set(placement.zoom);
    session.followCamera();
  });
  app.renderer.on('resize', () => {
    log('viewport resized', { width: app.screen.width, height: app.screen.height });
    session.refresh();
  });
}

function play(config: GameConfig, host: HTMLElement, app: Application, world: World, robots: RobotTextures, logger: Logger): void {
  // where the world is drawn right now, as the camera last placed it
  const placement = (): WorldPlacement => ({ origin: world.view.root.position, zoom: world.view.root.scale.x });
  const frame: TileFrame = { tileWidth: config.tile.width, tileHeight: config.tile.height, headroom: 0 };
  const store = browserStore();
  let explored = noExploration;
  // the chunks held, kept by the session; the views read their heights
  const held: HeldChunksRef = { current: emptyChunks() };
  const heightOf = heightForDrawing(held, world.grid);
  const guest = loadGuest({
    store,
    newId: () => crypto.randomUUID(),
    randomInt: (max) => Math.floor(Math.random() * max),
    robots: robots.slugs,
    log: logger.scope('guest'),
  });
  const players = createPlayersView({
    layer: world.view.entities,
    heights: world.heights,
    heightAt: heightOf,
    graphics: playerGraphics(frame, robots),
    ticker: app.ticker,
    grid: world.grid,
    log: logger.scope('players'),
  });
  // a cell within the map, drawn at its copy nearest to your player
  const drawn = (cell: { col: number; row: number }) => {
    const anchor = players.ownCell();
    return anchor ? nearestCopy(world.grid, cell, anchor) : cell;
  };
  // bottom left, top to bottom: the hovered cell's coordinates, yours, then the chatbox
  const corner = bottomLeftCorner(host);
  const hoveredCoordinates = createCoordinatesLabel(corner, 'hud-coords hud-coords-hovered');
  const ownCoordinates = createCoordinatesLabel(corner, 'hud-coords');
  const session = startPlayersSession({
    map: world.map,
    socketUrl: socketUrl(window.location, world.map.slug, guest),
    openSocket: (url, handlers) => openGameSocket(url, handlers, logger.scope('socket')),
    listenToPointer: (handlers) =>
      listenToPointer(
        app.canvas,
        (point) => cellAt(point, placement(), world.heights, heightOf, world.grid),
        handlers,
        config.camera.dragThresholdPx,
        logger.scope('pointer'),
      ),
    createMenu: (choices) =>
      commandMenu(
        host,
        app,
        choices,
        (cell) => tileOnCanvas(drawn(cell), heightOf(cell), placement(), world.heights, config.tile.width),
        logger.scope('menu'),
      ),
    createChatbox: (onLine) => createChatbox(corner, onLine, logger.scope('chatbox')),
    players,
    chunks: exploring(
      createChunksView({
        water: world.view.water,
        ground: world.view.ground,
        entities: world.view.entities,
        heights: world.heights,
        graphics: world.graphics,
        grid: world.grid,
        held,
        bake: bakeWith(app),
        ticker: app.ticker,
        waterFrameMs: config.waterFrameMs,
        anchor: () => players.ownCell(),
        log: logger.scope('chunks'),
      }),
      (key) => (explored = explore(explored, key)),
    ),
    overlay: createOverlay({
      layer: world.view.overlay,
      heights: world.heights,
      heightAt: heightOf,
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
    held,
    onDisconnect: () => showNotice(host, 'Disconnected. Reload the page to come back.'),
    log: logger.scope('session'),
  });
  const camera = createGameCamera(config, app, world, players, store, logger.scope('camera'));
  const toggleLock = cameraIcon(host, config, camera, logger.scope('camera icon'));
  const controls = listenToControls({
    target: window,
    canvas: app.canvas,
    config: config.controls,
    platform: browserPlatform(),
    wheel: config.camera.wheel,
    dragThresholdPx: config.camera.dragThresholdPx,
    log: logger.scope('controls'),
  });
  startScreens({
    controls,
    worldmap: worldmapHandlers(camera, toggleLock, session),
    mapWindow: mapWindow(config, host, app, world, session, () => explored, logger.scope('map screen')),
    cancelSelection: () => session.cancel(),
    log: logger.scope('screens'),
  });
  driveCamera(app, world, camera, session, logger.scope('camera'));
  app.ticker.add(() => {
    hoveredCoordinates.show(session.hovered());
    ownCoordinates.show(session.whereabouts().you);
  });
}

/** The bottom left corner of the screen, stacking what's put in it from top to bottom. */
function bottomLeftCorner(host: HTMLElement): HTMLElement {
  const corner = document.createElement('div');
  corner.className = 'hud-corner';
  host.appendChild(corner);
  return corner;
}

/** The chunks view, also telling each chunk shown: what you explored (ADR 0005). */
function exploring(chunks: ChunksView, onShown: (key: ChunkKey) => void): ChunksView {
  return {
    show: (chunk) => {
      chunks.show(chunk);
      onShown(chunk.key);
    },
    forget: (key) => chunks.forget(key),
    follow: (anchor) => chunks.follow(anchor),
  };
}

/** The camera, starting with the saved lock and saving it each time it switches (ADR 0004). */
function createGameCamera(
  config: GameConfig,
  app: Application,
  world: World,
  players: PlayersView,
  store: KeyValueStore | null,
  log: Log,
): Camera {
  const locked = loadCameraLock(store);
  log('lock read back', { locked, storage: store !== null });
  return createCamera({
    rules: { projection: world.projection, zoom: config.camera.zoom, leashCells: config.camera.leashCells },
    start: initialCamera(locked, config.camera.zoom.start),
    player: () => players.ownPosition(),
    viewport: () => app.screen,
    saveLock: (next) => saveCameraLock(store, next),
    log,
  });
}

/** The camera icon, top right: it shows the lock, and clicking it switches it. Returns the switch, for Y too. */
function cameraIcon(host: HTMLElement, config: GameConfig, camera: Camera, log: Log): () => void {
  const icons = {
    locked: spriteDataUrl(paintCameraIcon(true), config.cursorScale) ?? '',
    free: spriteDataUrl(paintCameraIcon(false), config.cursorScale) ?? '',
  };
  const button = createCameraButton(host, icons, () => toggleLock(), log);
  const toggleLock = () => {
    camera.toggleLock();
    button.show(camera.locked());
  };
  button.show(camera.locked());
  return toggleLock;
}

/** The world map's controls (ADR 0004). The screens add opening the map. */
function worldmapHandlers(camera: Camera, toggleLock: () => void, session: PlayersSession): WorldmapHandlers {
  return {
    toggleLock,
    center: () => camera.center(),
    zoomIn: ({ at }) => camera.zoom(1, at),
    zoomOut: ({ at }) => camera.zoom(-1, at),
    pan: ({ delta }) => delta && camera.pan(delta),
    cancel: () => session.cancel(),
    chat: () => session.openChat(),
  };
}

/** The map screen (ADR 0005), following the session every frame while it's open. */
function mapWindow(
  config: GameConfig,
  host: HTMLElement,
  app: Application,
  world: World,
  session: PlayersSession,
  explored: () => Exploration,
  log: Log,
): MapWindow {
  const screen = createMapScreen({
    host,
    map: world.map,
    imageUrl: mapImageUrl(world.map.slug),
    art: mapArt(config.cursorScale),
    viewport: () => app.screen,
    log,
  });
  const view = () => {
    const { you, others, region } = session.whereabouts();
    return { you, others, regionName: region ? regionName(world.map, region) : null, explored: explored() };
  };
  app.ticker.add(() => {
    if (screen.isOpen()) screen.update(view());
  });
  return { open: () => screen.open(view()), close: () => screen.close() };
}

/** The map screen's sprites, scaled up like the cursor. */
function mapArt(scale: number): MapArt {
  const url = (sprite: PixelBuffer) => spriteDataUrl(sprite, scale) ?? '';
  return {
    donut: url(paintDonut()),
    you: { square: url(paintYouMark('square')), up: url(paintYouMark('up')), down: url(paintYouMark('down')) },
    other: url(paintOtherMark()),
    scale,
    ring: DONUT_MARK_RING,
  };
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

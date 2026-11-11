import '@fontsource/zen-maru-gothic/500.css';
import '@fontsource/zen-maru-gothic/700.css';
import { Application, Assets, type Texture, type Ticker } from 'pixi.js';
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
import { actionLabel } from './game/actions';
import { createCamera, initialCamera, type Camera } from './game/camera';
import { loadCameraLock, saveCameraLock } from './game/cameraSettings';
import { answerTypedLine } from './game/chatLine';
import { createExplorations, type Exploration } from './game/exploration';
import { logMapSwitch } from './game/gameLog';
import { startScreens, type MapWindow, type WorldmapHandlers } from './game/screens';
import { startPlayersSession, type CommandMenu, type MenuChoices, type PlayersSession } from './game/session';
import { arrive, initialTravel, receive, type TravelState } from './game/travel';
import { browserPlatform, listenToControls } from './input/controls';
import type { Heights } from './iso/heights';
import { createIsoProjection, type ScreenPoint, type TilePoint } from './iso/projection';
import { emptyChunks, heightForDrawing, type HeldChunksRef } from './map/chunks';
import { nearestCopy, type ChunkKey } from './map/grid';
import { gridOf, regionName, type MapDescription } from './map/worldMap';
import { openGameSocket, type GameSocket } from './net/gameSocket';
import { loadGuest } from './net/guest';
import { mapImageUrl } from './net/mapApi';
import { socketUrl, type ServerMessage } from './net/protocol';
import { cellAt, tileOnCanvas, type WorldPlacement } from './render/camera';
import { createChunksView, type Bake, type ChunksView } from './render/chunkView';
import { createWorldView, type WorldView } from './render/mapView';
import { createOverlay } from './render/overlay';
import { createPlayersView, type FrameTicker, type PlayerGraphics, type PlayersView } from './render/playersView';
import { loadRobotTextures, type RobotTextures } from './render/robotTextures';
import { createWorldGraphics, spriteGraphic, type WorldGraphics } from './render/tileGraphics';
import { createHeldBanner, createRegionBanner, type HeldBanner } from './ui/banner';
import { createCameraButton } from './ui/cameraButton';
import { createChatbox, type Chatbox } from './ui/chatbox';
import { createCoordinatesLabel } from './ui/coordinates';
import { spriteCursor, spriteDataUrl } from './ui/cursor';
import { createFade } from './ui/fade';
import { createMapScreen, type MapArt } from './ui/mapScreen';
import { createActionMenu, type ActionMenu } from './ui/menu';
import { showNotice } from './ui/notice';
import { listenToPointer } from './ui/pointer';

/** What lives for the whole game, whatever the map (ADR 0012). */
interface Game {
  config: GameConfig;
  host: HTMLElement;
  app: Application;
  logger: Logger;
  robots: RobotTextures;
  /** How heights are drawn (ADR 0007). */
  heights: Heights;
  /** The art's textures, painted once, shared by every map. */
  graphics: WorldGraphics;
  menu: ActionMenu;
  chatbox: Chatbox;
  banner: HeldBanner;
}

/** A map being played: its views, its session, its map screen. It's destroyed when the game goes to another map. */
interface MapPlay {
  view: WorldView;
  session: PlayersSession;
  players: PlayersView;
  mapWindow: MapWindow & { isOpen(): boolean };
  destroy(): void;
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
  let robots: RobotTextures;
  try {
    robots = await loadRobotTextures({ source: robotSheetSource(), log: logger.scope('robots') });
  } catch (error) {
    console.error(error);
    showNotice(host, 'Could not load the sprites.');
    return;
  }
  play(config, host, app, robots, logger);
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

/**
 * Plays: opens the game's socket, and builds each map the server puts you on (ADR 0012). A welcome for another map
 * fades the screen out, destroys the map shown, builds the new one, applies the messages that waited, and fades in.
 */
function play(config: GameConfig, host: HTMLElement, app: Application, robots: RobotTextures, logger: Logger): void {
  const log = logger.scope('game');
  const store = browserStore();
  const guest = loadGuest({
    store,
    newId: () => crypto.randomUUID(),
    randomInt: (max) => Math.floor(Math.random() * max),
    robots: robots.slugs,
    log: logger.scope('guest'),
  });
  const projection = createIsoProjection(config.tile.width, config.tile.height);
  const artSet = createGeneratedArt({ tileWidth: config.tile.width, tileHeight: config.tile.height, rockHeadroom: config.rockHeadroom, levelPx: config.levelHeight });
  // bottom left, top to bottom: the hovered cell's coordinates, yours, then the chatbox
  const corner = bottomLeftCorner(host);
  const hoveredCoordinates = createCoordinatesLabel(corner, 'hud-coords hud-coords-hovered');
  const ownCoordinates = createCoordinatesLabel(corner, 'hud-coords');
  const chatbox = createChatbox(
    corner,
    (text) => answerTypedLine(text, { addLine: (line) => chatbox.addLine(line), teleport: (map, to) => socket.teleport(map, to), log: logger.scope('chat') }),
    logger.scope('chatbox'),
  );
  const game: Game = {
    config,
    host,
    app,
    logger,
    robots,
    heights: { projection, levelPx: config.levelHeight },
    graphics: createWorldGraphics({ artSet, variantsPerKind: config.variantsPerKind, rng: createSeededRng(config.seed) }),
    menu: createActionMenu(host, () => app.screen, logger.scope('menu')),
    chatbox,
    banner: createHeldBanner(createRegionBanner(host, logger.scope('banner'))),
  };
  const fade = createFade(host, config.fadeMs, logger.scope('fade'));
  const explorations = createExplorations();
  let current: MapPlay | null = null;
  let travel: TravelState = initialTravel();

  const socket = openGameSocket(
    socketUrl(window.location, guest),
    { onMessage: (message) => onMessage(message), onClose: () => showNotice(host, 'Disconnected. Reload the page to come back.') },
    logger.scope('socket'),
  );
  const camera = createGameCamera(config, app, () => current?.players.ownPosition() ?? null, store, logger.scope('camera'));
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
  const screens = startScreens({
    controls,
    worldmap: worldmapHandlers(camera, toggleLock, () => current, chatbox),
    mapWindow: { open: () => current?.mapWindow.open(), close: () => current?.mapWindow.close() },
    cancelSelection: () => current?.session.cancel(),
    log: logger.scope('screens'),
  });
  driveCamera(app, () => current, camera, logger.scope('camera'));
  app.ticker.add(() => {
    hoveredCoordinates.show(current?.session.hovered() ?? null);
    ownCoordinates.show(current?.session.whereabouts().you ?? null);
  });

  const build = (map: MapDescription): MapPlay =>
    playMap(game, map, socket, { explored: () => explorations.of(map.slug), explore: (key) => explorations.explore(map.slug, key) });

  function onMessage(message: ServerMessage): void {
    const next = receive(travel, message);
    travel = next.state;
    const step = next.step;
    if (step.kind === 'apply') current?.session.receive(message);
    else if (step.kind === 'first') {
      log('first map', { map: step.map.slug });
      game.banner.hold();
      current = build(step.map);
      current.session.receive(message);
      fade.in(() => game.banner.release());
    } else if (step.kind === 'leave') leave(step.from, step.to);
  }

  // the screen goes dark, the map shown is destroyed, the new one built with the messages that waited, the camera
  // put back on your player, and back
  function leave(from: MapDescription, to: MapDescription): void {
    log('leaving for another map', { from: from.slug, to: to.slug });
    logMapSwitch(from, to).forEach((line) => chatbox.addLine(line));
    if (current?.mapWindow.isOpen()) screens.closeMap();
    current?.session.stop();
    game.banner.hold();
    fade.out(() => {
      const arrived = arrive(travel);
      if (!arrived) return;
      travel = arrived.state;
      const started = performance.now();
      current?.destroy();
      const next = build(arrived.map);
      current = next;
      camera.recenter();
      arrived.messages.forEach((message) => next.session.receive(message));
      log('map switched', { from: from.slug, to: arrived.map.slug, waited: arrived.messages.length - 1, ms: Math.round(performance.now() - started) });
      fade.in(() => game.banner.release());
    });
  }
}

/** A frame ticker whose callbacks all go when the map they belong to is destroyed. */
function scopedTicker(ticker: Ticker): FrameTicker & { stop(): void } {
  const added: (() => void)[] = [];
  return {
    add: (onFrame) => {
      added.push(onFrame);
      ticker.add(onFrame);
    },
    stop: () => added.splice(0).forEach((onFrame) => ticker.remove(onFrame)),
  };
}

/** What a map was explored, kept by the game for the session, per map (ADR 0012). */
interface ExplorationOf {
  explored(): Exploration;
  explore(key: ChunkKey): void;
}

/** Builds a map's views and session, on the game's socket: everything it draws is destroyed with it. */
function playMap(game: Game, map: MapDescription, socket: GameSocket, exploration: ExplorationOf): MapPlay {
  const { config, app, logger, heights } = game;
  const grid = gridOf(map);
  const ticker = scopedTicker(app.ticker);
  const view = createWorldView();
  view.root.scale.set(config.camera.zoom.start);
  app.stage.addChild(view.root);
  // where the map is drawn right now, as the camera last placed it
  const placement = (): WorldPlacement => ({ origin: view.root.position, zoom: view.root.scale.x });
  const frame: TileFrame = { tileWidth: config.tile.width, tileHeight: config.tile.height, headroom: 0 };
  // the chunks held, kept by the session; the views read their heights
  const held: HeldChunksRef = { current: emptyChunks() };
  const heightOf = heightForDrawing(held, grid);
  const players = createPlayersView({
    layer: view.entities,
    heights,
    heightAt: heightOf,
    graphics: playerGraphics(frame, game.robots),
    ticker,
    grid,
    log: logger.scope('players'),
  });
  // a cell within the map, drawn at its copy nearest to your player
  const drawn = (cell: TilePoint) => {
    const anchor = players.ownCell();
    return anchor ? nearestCopy(grid, cell, anchor) : cell;
  };
  const chunks = createChunksView({
    water: view.water,
    ground: view.ground,
    entities: view.entities,
    heights,
    graphics: game.graphics,
    grid,
    held,
    bake: bakeWith(app),
    ticker,
    waterFrameMs: config.waterFrameMs,
    anchor: () => players.ownCell(),
    log: logger.scope('chunks'),
  });
  const session = startPlayersSession({
    map,
    socket,
    listenToPointer: (handlers) =>
      listenToPointer(
        app.canvas,
        (point) => cellAt(point, placement(), heights, heightOf, grid, chunks.objectAt),
        handlers,
        config.camera.dragThresholdPx,
        logger.scope('pointer'),
      ),
    createMenu: (choices) => commandMenu(game.menu, choices, (cell) => tileOnCanvas(drawn(cell), heightOf(cell), placement(), heights, config.tile.width)),
    chatbox: game.chatbox,
    players,
    chunks: exploring(chunks, exploration.explore),
    overlay: createOverlay({
      layer: view.overlay,
      heights,
      heightAt: heightOf,
      borders: {
        hover: spriteGraphic(paintTileOutline(frame, BORDERS.hover.rgb, BORDERS.hover.alpha), frame),
        selection: spriteGraphic(paintTileOutline(frame, BORDERS.selection.rgb, BORDERS.selection.alpha), frame),
      },
      ticker,
      pathStart: () => players.ownPosition(),
      anchor: () => players.ownCell(),
      grid,
      log: logger.scope('overlay'),
    }),
    banner: game.banner,
    held,
    log: logger.scope('session'),
  });
  const screen = mapScreenOf(game, map, session, exploration.explored, ticker);
  return {
    view,
    session,
    players,
    mapWindow: screen,
    destroy() {
      session.stop();
      ticker.stop();
      chunks.clear();
      screen.destroy();
      app.stage.removeChild(view.root);
      view.root.destroy({ children: true });
    },
  };
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

/** Every frame, the map shown is placed and scaled where the camera says: on your player when it's locked. */
function driveCamera(app: Application, current: () => MapPlay | null, camera: Camera, log: Log): void {
  app.ticker.add(() => {
    const placement = camera.place();
    const shown = current();
    if (!placement || !shown) return;
    shown.view.root.position.set(placement.origin.x, placement.origin.y);
    shown.view.root.scale.set(placement.zoom);
    shown.session.followCamera();
  });
  app.renderer.on('resize', () => {
    log('viewport resized', { width: app.screen.width, height: app.screen.height });
    current()?.session.refresh();
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
    objectAt: (point) => chunks.objectAt(point),
    clear: () => chunks.clear(),
  };
}

/** The camera, starting with the saved lock and saving it each time it switches (ADR 0004). It follows your player on the map shown. */
function createGameCamera(config: GameConfig, app: Application, player: () => ScreenPoint | null, store: KeyValueStore | null, log: Log): Camera {
  const locked = loadCameraLock(store);
  log('lock read back', { locked, storage: store !== null });
  return createCamera({
    rules: { projection: createIsoProjection(config.tile.width, config.tile.height), zoom: config.camera.zoom, leashCells: config.camera.leashCells },
    start: initialCamera(locked, config.camera.zoom.start),
    player,
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

/** The world map's controls (ADR 0004), on the map shown. The screens add opening the map. */
function worldmapHandlers(camera: Camera, toggleLock: () => void, current: () => MapPlay | null, chatbox: Chatbox): WorldmapHandlers {
  return {
    toggleLock,
    center: () => camera.center(),
    zoomIn: ({ at }) => camera.zoom(1, at),
    zoomOut: ({ at }) => camera.zoom(-1, at),
    pan: ({ delta }) => delta && camera.pan(delta),
    cancel: () => current()?.session.cancel(),
    chat: () => chatbox.focus(),
  };
}

/** The map screen of a map (ADR 0005), following the session every frame while it's open. */
function mapScreenOf(
  game: Game,
  map: MapDescription,
  session: PlayersSession,
  explored: () => Exploration,
  ticker: FrameTicker,
): MapWindow & { isOpen(): boolean; destroy(): void } {
  const screen = createMapScreen({
    host: game.host,
    map,
    imageUrl: mapImageUrl(map.slug),
    art: mapArt(game.config.cursorScale),
    viewport: () => game.app.screen,
    log: game.logger.scope('map screen'),
  });
  const view = () => {
    const { you, others, region } = session.whereabouts();
    return { you, others, regionName: region ? regionName(map, region) : null, explored: explored() };
  };
  ticker.add(() => {
    if (screen.isOpen()) screen.update(view());
  });
  return { open: () => screen.open(view()), close: () => screen.close(), isOpen: () => screen.isOpen(), destroy: () => screen.destroy() };
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

/** The action menu of a selected tile, beside it: Move, Enter or Exit (ADR 0012), then Cancel. One menu for the game. */
function commandMenu(menu: ActionMenu, choices: MenuChoices, anchorOf: (cell: TilePoint) => { left: number; right: number; y: number }): CommandMenu {
  return {
    show: (cell, action, enabled) =>
      menu.show(anchorOf(cell), [
        { label: actionLabel(action), enabled, choose: choices.onAct },
        { label: 'Cancel', enabled: true, choose: choices.onCancel },
      ]),
    hide: () => menu.hide(),
  };
}

start(createDefaultConfig(), document.body);

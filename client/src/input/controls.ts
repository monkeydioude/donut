import { silent, type Log } from '../debug/log';
import type { ScreenPoint } from '../iso/projection';
import {
  keyOutcome,
  platformOf,
  pointerAction,
  type ActionBindings,
  type Button,
  type ControlsConfig,
  type ControlStates,
  type KeyPress,
  type Platform,
  type PointerInput,
  type StateName,
} from './bindings';
import { isClick, moveTo, press, type Press } from './drag';
import { addScroll, initialWheel, scrollPx, type WheelConfig, type WheelState } from './wheel';

/** What an action gets: the pointer on the canvas for the wheel and clicks, how far a drag went for a pan. */
export interface ActionInput {
  at?: ScreenPoint;
  delta?: ScreenPoint;
}

export type ActionHandlers<A extends string> = Readonly<Record<A, (input: ActionInput) => void>>;

export interface ControlsDeps {
  /** Where keys, the wheel and the buttons are listened to: the page's window. A left press counts on the canvas only. */
  target: Window;
  /** Pointer positions are given in CSS pixels from its top-left. */
  canvas: HTMLElement;
  config: ControlsConfig;
  platform: Platform;
  wheel: WheelConfig;
  dragThresholdPx: number;
  now?: () => number;
  log?: Log;
}

export interface Controls {
  /** Switches to a state: from now on, only its controls react. */
  enter<S extends StateName>(state: S, handlers: ActionHandlers<ControlStates[S]>): void;
  stop(): void;
}

interface CurrentState {
  name: StateName;
  bindings: ActionBindings<string>;
  handlers: ActionHandlers<string>;
}

/** A held button: which one, and its press. */
interface HeldButton {
  button: Button;
  press: Press;
}

/** The DOM's button numbers and `buttons` bits. */
const BUTTONS: Readonly<Record<Button, { number: number; bit: number }>> = {
  left: { number: 0, bit: 1 },
  right: { number: 2, bit: 2 },
};

function buttonOf(event: PointerEvent): Button | null {
  if (event.button === BUTTONS.left.number) return 'left';
  if (event.button === BUTTONS.right.number) return 'right';
  return null;
}

/** Keys, the wheel and the buttons' clicks and drags, routed to the current state's actions (ADR 0004). */
export function listenToControls(deps: ControlsDeps): Controls {
  const log = deps.log ?? silent;
  const now = deps.now ?? (() => performance.now());
  let current: CurrentState | null = null;
  let wheel: WheelState = initialWheel;
  let held: HeldButton | null = null;

  const canvasPoint = (event: MouseEvent): ScreenPoint => {
    const rect = deps.canvas.getBoundingClientRect();
    return { x: event.clientX - rect.left, y: event.clientY - rect.top };
  };
  const run = (action: string | null, input: ActionInput, via: string | null) => {
    if (!current || !action) return;
    if (via) log('action', { state: current.name, action, via });
    current.handlers[action]?.(input);
  };
  const runPointer = (pointer: PointerInput, input: ActionInput, via: string | null) =>
    current && run(pointerAction(current.bindings, pointer), input, via);
  const outcomeOf = (event: KeyboardEvent) =>
    current && !isTyping(event.target) ? keyOutcome(current.bindings, keyPress(event), deps.platform) : null;

  const onKeyDown = (event: KeyboardEvent) => {
    const outcome = outcomeOf(event);
    if (outcome?.block) event.preventDefault();
    if (outcome?.action) run(outcome.action, {}, event.code);
  };
  const onKeyUp = (event: KeyboardEvent) => {
    if (outcomeOf(event)?.block) event.preventDefault();
  };
  // the page never scrolls or zooms: Cmd/Ctrl + wheel and trackpad pinches zoom the game
  const onWheel = (event: WheelEvent) => {
    event.preventDefault();
    const next = addScroll(wheel, scrollPx(event.deltaY, event.deltaMode), now(), deps.wheel);
    wheel = next.state;
    if (next.step) runPointer({ wheel: next.step }, { at: canvasPoint(event) }, `wheel ${next.step}`);
  };
  // the left button presses on the world only: a drag from the menu or the camera icon doesn't pan
  const onPointerDown = (event: PointerEvent) => {
    const button = buttonOf(event);
    if (button === 'left' && event.target !== deps.canvas) return;
    if (button) held = { button, press: press(canvasPoint(event)) };
  };
  const onPointerMove = (event: PointerEvent) => {
    if (!held) return;
    if ((event.buttons & BUTTONS[held.button].bit) === 0) return void (held = null);
    const wasDragging = held.press.dragging;
    const next = moveTo(held.press, canvasPoint(event), deps.dragThresholdPx);
    held = { ...held, press: next.held };
    if (!next.delta) return;
    if (!wasDragging) log(`${held.button} drag: start`);
    runPointer({ button: held.button, gesture: 'drag' }, { delta: next.delta }, null);
  };
  const onPointerUp = (event: PointerEvent) => {
    if (!held || buttonOf(event) !== held.button) return;
    const at = canvasPoint(event);
    const { button, press: pressed } = held;
    const click = isClick(pressed, at, deps.dragThresholdPx);
    if (pressed.dragging) log(`${button} drag: end`);
    held = null;
    if (click) runPointer({ button, gesture: 'click' }, { at }, `${button} click`);
  };
  const onContextMenu = (event: MouseEvent) => event.preventDefault();

  // keys are caught before anything else on the page, and the wheel must be able to block the page's zoom
  const listeners: [string, EventListener, AddEventListenerOptions][] = [
    ['keydown', onKeyDown as EventListener, { capture: true }],
    ['keyup', onKeyUp as EventListener, { capture: true }],
    ['wheel', onWheel as EventListener, { passive: false }],
    ['pointerdown', onPointerDown as EventListener, {}],
    ['pointermove', onPointerMove as EventListener, {}],
    ['pointerup', onPointerUp as EventListener, {}],
    ['contextmenu', onContextMenu as EventListener, {}],
  ];
  listeners.forEach(([type, listener, options]) => deps.target.addEventListener(type, listener, options));

  return {
    enter(state, handlers) {
      log('state entered', state);
      current = { name: state, bindings: deps.config[state], handlers };
      wheel = initialWheel;
      held = null;
    },
    stop() {
      listeners.forEach(([type, listener, options]) => deps.target.removeEventListener(type, listener, options));
    },
  };
}

/** The browser's platform: macOS and iOS hold the modifier on Cmd, the others on Ctrl. */
export function browserPlatform(nav: Navigator = navigator): Platform {
  const data = (nav as Navigator & { userAgentData?: { platform?: string } }).userAgentData;
  return platformOf(data?.platform || nav.platform || nav.userAgent);
}

function keyPress(event: KeyboardEvent): KeyPress {
  return { code: event.code, key: event.key, meta: event.metaKey, ctrl: event.ctrlKey, repeat: event.repeat };
}

/** Keys typed in a text field are left alone. */
function isTyping(target: EventTarget | null): boolean {
  return target instanceof HTMLElement && (target.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(target.tagName));
}

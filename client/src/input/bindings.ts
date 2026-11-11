/**
 * An input that can trigger an action:
 * - a key by its position (`KeyboardEvent.code`): `KeyZ` is the key printed W on AZERTY;
 * - a key by its printed character (`KeyboardEvent.key`), for keys whose position varies between layouts (+, −);
 * - the wheel, up or down;
 * - the left or right button, clicked or dragged. A left click on a tile selects it: that one isn't a binding.
 * A key with `mod` needs the modifier: Cmd on macOS, Ctrl elsewhere.
 */
export type Binding =
  | { code: string; mod?: boolean }
  | { char: string; mod?: boolean }
  | { wheel: WheelDirection }
  | { button: Button; gesture: Gesture };

export type WheelDirection = 'up' | 'down';
export type Button = 'left' | 'right';
export type Gesture = 'click' | 'drag';

/** Each action of a state, with the inputs that trigger it. */
export type ActionBindings<A extends string> = Readonly<Record<A, readonly Binding[]>>;

/** The game's states, each with the actions its controls trigger (ADR 0004, ADR 0005). */
export interface ControlStates {
  worldmap: 'toggleLock' | 'center' | 'zoomIn' | 'zoomOut' | 'pan' | 'cancel' | 'openMap' | 'chat';
  map: 'close';
}

export type StateName = keyof ControlStates;

export type ControlsConfig = { readonly [S in StateName]: ActionBindings<ControlStates[S]> };

/** Which key holds the modifier: Cmd on macOS, Ctrl elsewhere. */
export type Platform = 'mac' | 'other';

/** A key press, as the bindings read it. */
export interface KeyPress {
  code: string;
  key: string;
  meta: boolean;
  ctrl: boolean;
  repeat: boolean;
}

/** A pointer input: the wheel's step, or a button's gesture. */
export type PointerInput = { wheel: WheelDirection } | { button: Button; gesture: Gesture };

/** What a key press does: whether the page's default action is blocked, and the action to run, if any. */
export interface KeyOutcome<A extends string> {
  block: boolean;
  action: A | null;
}

/**
 * A key bound in the state blocks the page's default (scroll, page zoom, a focused button). It runs its action
 * once: a held key's repeats are blocked but do nothing.
 */
export function keyOutcome<A extends string>(bindings: ActionBindings<A>, press: KeyPress, platform: Platform): KeyOutcome<A> {
  const action = findAction(bindings, (binding) => keyMatches(binding, press, platform));
  return { block: action !== null, action: press.repeat ? null : action };
}

/** The action a wheel step or a button's gesture triggers in the state, or null. */
export function pointerAction<A extends string>(bindings: ActionBindings<A>, input: PointerInput): A | null {
  return findAction(bindings, (binding) => pointerMatches(binding, input));
}

/** The platform of a browser, from its platform name (`navigator.platform`, or the user agent data's). */
export function platformOf(name: string): Platform {
  return /mac|iphone|ipad/i.test(name) ? 'mac' : 'other';
}

function findAction<A extends string>(bindings: ActionBindings<A>, matches: (binding: Binding) => boolean): A | null {
  const actions = Object.keys(bindings) as A[];
  return actions.find((action) => bindings[action].some(matches)) ?? null;
}

/** A key without the modifier doesn't match while Cmd or Ctrl is held: Cmd + Z isn't Z. Shift and Alt are ignored. */
function keyMatches(binding: Binding, press: KeyPress, platform: Platform): boolean {
  if (!('code' in binding) && !('char' in binding)) return false;
  const pressed = 'code' in binding ? binding.code === press.code : binding.char === press.key;
  if (!pressed) return false;
  if (binding.mod) return platform === 'mac' ? press.meta : press.ctrl;
  return !press.meta && !press.ctrl;
}

/** The wheel matches with or without a modifier: a trackpad pinch comes as Ctrl + wheel. */
function pointerMatches(binding: Binding, input: PointerInput): boolean {
  if ('wheel' in binding) return 'wheel' in input && binding.wheel === input.wheel;
  if ('button' in binding) return 'button' in input && binding.button === input.button && binding.gesture === input.gesture;
  return false;
}

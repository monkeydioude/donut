import { silent, type Log } from '../debug/log';
import { addLine, chatKeyOf, emptyHistory, scrollBy, visibleLines, type ChatHistory } from '../game/chatHistory';
import { commandsMatching, usageOf, type CommandHelp } from '../game/commands';
import type { LogLine } from '../game/gameLog';

export interface Chatbox {
  /** Adds a line at the bottom and shows it. Text only, never HTML. */
  addLine(line: LogLine): void;
  /** Gives the input the focus: the game's keys type here until Escape or a click elsewhere. */
  focus(): void;
}

/**
 * The chatbox (ADR 0010): a bar at the bottom of the screen, DOM over the canvas, with a few lines of history over an
 * input. Enter sends the line, Escape blurs, the arrows scroll the history while it has focus, and the wheel over it
 * scrolls it instead of zooming the game. While a command is typed, the commands it may be are listed over the input.
 */
export function createChatbox(host: HTMLElement, onLine: (text: string) => void, log: Log = silent): Chatbox {
  const root = document.createElement('div');
  root.className = 'chatbox';
  const history = document.createElement('div');
  history.className = 'chatbox-history';
  const input = document.createElement('input');
  input.className = 'chatbox-input';
  input.type = 'text';
  input.autocomplete = 'off';
  input.spellcheck = false;
  input.placeholder = 'Enter to type. Commands start with /';
  const commands = document.createElement('div');
  commands.className = 'chatbox-commands';
  commands.hidden = true;
  const entry = document.createElement('div');
  entry.className = 'chatbox-entry';
  entry.append(commands, input);
  root.append(history, entry);
  host.appendChild(root);
  let shown: ChatHistory = emptyHistory();

  const render = () => {
    history.replaceChildren(
      ...visibleLines(shown).map((line) => {
        const element = document.createElement('div');
        element.className = `chatbox-line chatbox-${line.kind}`;
        element.textContent = line.text;
        return element;
      }),
    );
  };
  // the commands the line may be, while the input has focus
  const listCommands = (focused: boolean) => {
    const matching = focused ? commandsMatching(input.value) : [];
    commands.replaceChildren(...matching.map(commandLine));
    commands.hidden = matching.length === 0;
  };
  const scroll = (lines: number) => {
    shown = scrollBy(shown, lines);
    render();
  };
  const send = () => {
    const text = input.value;
    input.value = '';
    listCommands(true);
    if (text.trim() === '') return;
    log('line sent', text);
    onLine(text);
  };

  input.addEventListener('keydown', (event) => {
    const key = chatKeyOf(event.key);
    if (!key) return;
    event.preventDefault();
    switch (key) {
      case 'send':
        return send();
      case 'blur':
        return input.blur();
      case 'up':
        return scroll(1);
      case 'down':
        return scroll(-1);
    }
  });
  input.addEventListener('input', () => listCommands(true));
  input.addEventListener('focus', () => {
    log('opened');
    listCommands(true);
  });
  input.addEventListener('blur', () => {
    log('closed');
    listCommands(false);
  });
  // the wheel scrolls the history, and never reaches the game's zoom
  root.addEventListener(
    'wheel',
    (event) => {
      event.preventDefault();
      event.stopPropagation();
      scroll(event.deltaY < 0 ? 1 : -1);
    },
    { passive: false },
  );

  return {
    addLine(line) {
      shown = addLine(shown, line);
      render();
    },
    focus() {
      input.focus();
    },
  };
}

/** A listed command: how it's typed, then what it does. Text only, never HTML. */
function commandLine(command: CommandHelp): HTMLElement {
  const line = textElement('div', 'chatbox-command', '');
  line.append(textElement('span', 'chatbox-command-usage', usageOf(command)), textElement('span', 'chatbox-command-does', command.does));
  return line;
}

function textElement(tag: 'div' | 'span', className: string, text: string): HTMLElement {
  const created = document.createElement(tag);
  created.className = className;
  created.textContent = text;
  return created;
}

import { silent, type Log } from '../debug/log';
import { decodeServerMessage, encodePlan, encodeWalk, type Cell, type ServerMessage } from './protocol';

export interface GameSocket {
  /** Asks the path from your player to a cell: the answer comes as a `planned` message. */
  plan(to: Cell): void;
  /** Asks the server to walk your player to a cell: it answers `walking`, then each step comes as `moved`. */
  walk(to: Cell): void;
  close(): void;
}

export interface GameSocketHandlers {
  onMessage(message: ServerMessage): void;
  /** The connection ended: the server closed it, the network dropped, or another tab took over. */
  onClose(): void;
}

/**
 * Opens the websocket of a map. Messages the client doesn't understand are dropped.
 * Logs only the URL's path: its query carries your secret guest id.
 */
export function openGameSocket(url: string, handlers: GameSocketHandlers, log: Log = silent): GameSocket {
  log('opening', new URL(url).pathname);
  const socket = new WebSocket(url);
  socket.addEventListener('open', () => log('open'));
  socket.addEventListener('message', (event) => {
    const message = typeof event.data === 'string' ? decodeServerMessage(event.data) : null;
    if (!message) {
      log('received a message it does not understand, dropped', event.data);
      return;
    }
    log(`← ${message.type}`, message);
    handlers.onMessage(message);
  });
  socket.addEventListener('error', () => log('error (the close event tells why)'));
  socket.addEventListener('close', (event) => {
    log('closed', { code: event.code, reason: event.reason, clean: event.wasClean });
    handlers.onClose();
  });
  const send = (text: string) => {
    if (socket.readyState !== WebSocket.OPEN) {
      log('not sent, socket not open', { readyState: socket.readyState, text });
      return;
    }
    log('→ send', JSON.parse(text));
    socket.send(text);
  };
  return {
    plan: (to) => send(encodePlan(to)),
    walk: (to) => send(encodeWalk(to)),
    close: () => {
      log('closing');
      socket.close();
    },
  };
}

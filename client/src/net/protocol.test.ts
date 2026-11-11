import { describe, expect, it } from 'vitest';
import { decodeServerMessage, encodePlan, encodeWalk, socketUrl } from './protocol';

const ann = { id: 'a', name: 'Ann', robot: 'dome-head', col: 1, row: 2 };

// ADR 0001 test 22, ADR 0002 test 18
describe('decodeServerMessage', () => {
  it('decodes every server message', () => {
    const messages = [
      { type: 'welcome', you: 'a', players: [ann], stepMs: 200 },
      { type: 'joined', player: ann },
      { type: 'moved', id: 'a', col: 3, row: 4 },
      { type: 'left', id: 'a' },
      { type: 'planned', col: 3, row: 2, path: [{ col: 2, row: 2 }, { col: 3, row: 2 }] },
      { type: 'planned', col: 3, row: 2, path: [] },
      { type: 'walking', path: [{ col: 2, row: 2 }] },
      { type: 'stopped', col: 1, row: 2 },
    ];
    for (const message of messages) {
      expect(decodeServerMessage(JSON.stringify(message))).toEqual(message);
    }
  });

  it('gives nothing for an unknown or malformed message', () => {
    const bad = [
      'not json',
      '[]',
      '{"type": "dance"}',
      '{"type": "rejected", "col": 1, "row": 2}',
      '{"type": "moved", "id": "a", "col": 1}',
      '{"type": "moved", "id": "a", "col": 1.5, "row": 2}',
      '{"type": "joined", "player": {"id": "a"}}',
      '{"type": "welcome", "you": "a", "players": [], "stepMs": 0}',
      '{"type": "welcome", "you": "a", "players": [{"id": "b"}], "stepMs": 200}',
      '{"type": "planned", "col": 1, "row": 2}',
      '{"type": "planned", "col": 1, "row": 2, "path": [{"col": 1}]}',
      '{"type": "walking", "path": null}',
      '{"type": "stopped"}',
      '{"type": "left"}',
    ];
    for (const text of bad) {
      expect(decodeServerMessage(text), text).toBeNull();
    }
  });
});

/** A chunk as the server sends it: grass with one void cell, a rock, region 3 everywhere. */
function serverChunk(): Record<string, unknown> {
  const ground = Array<number>(1024).fill(0);
  ground[1] = 1;
  return {
    type: 'chunk',
    cx: 2,
    cy: 1,
    version: 1,
    kinds: ['grass', 'void'],
    ground,
    height: Array<number>(1024).fill(0),
    region: Array<number>(1024).fill(3),
    objects: [{ id: 'r1', kind: 'rock', col: 70, row: 40 }],
  };
}

// ADR 0003 test 25
describe('chunk messages', () => {
  it('decode a chunk, its ground as kinds per cell', () => {
    const message = decodeServerMessage(JSON.stringify(serverChunk()));
    expect(message?.type).toBe('chunk');
    if (message?.type !== 'chunk') return;
    expect(message.chunk.key).toEqual({ cx: 2, cy: 1 });
    expect(message.chunk.version).toBe(1);
    expect(message.chunk.ground.slice(0, 3)).toEqual(['grass', 'void', 'grass']);
    expect(message.chunk.ground).toHaveLength(1024);
    expect(message.chunk.region[0]).toBe(3);
    expect(message.chunk.objects).toEqual([{ id: 'r1', kind: 'rock', col: 70, row: 40 }]);
  });

  // ADR 0007 test 26
  it('decode water, holes, bridges, the ocean and heights from -3 to 3', () => {
    const chunk = serverChunk();
    chunk.kinds = ['grass', 'water', 'hole', 'bridge', 'ocean'];
    chunk.ground = Array.from({ length: 1024 }, (_, i) => i % 4);
    chunk.height = Array.from({ length: 1024 }, (_, i) => (i % 7) - 3);
    const message = decodeServerMessage(JSON.stringify(chunk));
    if (message?.type !== 'chunk') throw new Error('not a chunk');
    expect(message.chunk.ground.slice(0, 4)).toEqual(['grass', 'water', 'hole', 'bridge']);
    expect(message.chunk.height.slice(0, 7)).toEqual([-3, -2, -1, 0, 1, 2, 3]);
  });

  it('decode a forget', () => {
    expect(decodeServerMessage('{"type": "forget", "cx": 2, "cy": 1}')).toEqual({ type: 'forget', cx: 2, cy: 1 });
  });

  const broken: Record<string, (chunk: Record<string, unknown>) => void> = {
    'a ground too short': (c) => (c.ground = Array(1023).fill(0)),
    'a height too long': (c) => (c.height = Array(1025).fill(0)),
    'an index outside the kinds': (c) => ((c.ground as number[])[5] = 2),
    'an unknown kind': (c) => (c.kinds = ['grass', 'lava']),
    'no kinds': (c) => (c.kinds = []),
    'a region over 255': (c) => ((c.region as number[])[0] = 256),
    'a height of 4 (ADR 0007)': (c) => ((c.height as number[])[0] = 4),
    'a height of -4 (ADR 0007)': (c) => ((c.height as number[])[0] = -4),
    'an object outside its chunk': (c) => (c.objects = [{ id: 'r1', kind: 'rock', col: 96, row: 40 }]),
    'an object of an unknown kind': (c) => (c.objects = [{ id: 't1', kind: 'tree', col: 70, row: 40 }]),
    'a negative chunk': (c) => (c.cx = -1),
    'version 0': (c) => (c.version = 0),
  };
  for (const [name, breakIt] of Object.entries(broken)) {
    it(`give nothing for ${name}`, () => {
      const chunk = serverChunk();
      breakIt(chunk);
      expect(decodeServerMessage(JSON.stringify(chunk))).toBeNull();
    });
  }

  it('give nothing for a forget without its chunk', () => {
    expect(decodeServerMessage('{"type": "forget", "cx": 2}')).toBeNull();
  });
});

describe('encodePlan and encodeWalk', () => {
  it('ask for a cell', () => {
    expect(JSON.parse(encodePlan({ col: 3, row: 4 }))).toEqual({ type: 'plan', col: 3, row: 4 });
    expect(JSON.parse(encodeWalk({ col: 3, row: 4 }))).toEqual({ type: 'walk', col: 3, row: 4 });
  });
});

// ADR 0006 test 26
describe('a player\'s robot', () => {
  const decodedRobot = (player: Record<string, unknown>) => {
    const message = decodeServerMessage(JSON.stringify({ type: 'joined', player }));
    return message?.type === 'joined' ? message.player.robot : null;
  };

  it('is decoded when it is a string', () => {
    expect(decodedRobot(ann)).toBe('dome-head');
  });

  it('is "" when it is missing or not a string', () => {
    const { robot: _, ...noRobot } = ann;
    expect(decodedRobot(noRobot)).toBe('');
    expect(decodedRobot({ ...ann, robot: 7 })).toBe('');
    expect(decodedRobot({ ...ann, robot: null })).toBe('');
  });

  it('is "" in a welcome too, and nothing else is kept from the player', () => {
    const message = decodeServerMessage(JSON.stringify({ type: 'welcome', you: 'a', players: [{ id: 'a', name: 'Ann', col: 1, row: 2, extra: 1 }], stepMs: 200 }));
    expect(message).toEqual({ type: 'welcome', you: 'a', players: [{ id: 'a', name: 'Ann', robot: '', col: 1, row: 2 }], stepMs: 200 });
  });
});

// ADR 0006 test 25
describe('socketUrl', () => {
  const guest = { id: 'a-b', name: 'Guest 12&', robot: 'dome-head' };

  it('opens the websocket on the page host, ws or wss like the page, with your robot', () => {
    expect(socketUrl({ protocol: 'http:', host: 'localhost:5180' }, 'meadow', guest)).toBe(
      'ws://localhost:5180/api/maps/meadow/ws?player=a-b&name=Guest+12%26&robot=dome-head',
    );
    expect(socketUrl({ protocol: 'https:', host: 'donut.example' }, 'meadow', guest)).toMatch(/^wss:\/\/donut\.example\//);
  });
});

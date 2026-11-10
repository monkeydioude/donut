// A short session against the separate back: join, a bad message, plan, walk, wait for the steps, leave.
const guest = '44444444-4444-4444-8444-444444444444';
const socket = new WebSocket(`ws://localhost:8401/api/maps/meadow/ws?player=${guest}&name=Logger`, { headers: { Origin: 'http://localhost:5180' } });
const wait = (ms) => new Promise((r) => setTimeout(r, ms));
let me;
socket.onmessage = (e) => { const m = JSON.parse(e.data); if (m.type === 'welcome') me = m.players.find((p) => p.id === m.you); };
socket.onopen = async () => {
  await wait(200);
  socket.send('{"type": "dance"}');
  socket.send(JSON.stringify({ type: 'plan', col: me.col, row: me.row + 3 }));
  await wait(200);
  socket.send(JSON.stringify({ type: 'walk', col: me.col, row: me.row + 3 }));
  await wait(1200);
  socket.close();
  await wait(300);
  process.exit(0);
};

package wstransport_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
	"donut/internal/presence/app"
	"donut/internal/presence/domain"
	wstransport "donut/internal/presence/transport/ws"
	"donut/pkg/errd"
	"donut/pkg/logs"
)

const (
	front = "http://localhost:5180"
	// guest ids: secrets each client keeps
	ann = "00000000-0000-0000-0000-00000000000a"
	bob = "00000000-0000-0000-0000-00000000000b"
)

// annPublic is the id the others know ann by.
var annPublic = domain.PublicID(ann)

// openWorld is a map without rocks; paths go straight along rows. Players spawn on its spawn cells in turn.
type openWorld struct {
	grid   core.Grid
	spawns []core.Cell
	joins  atomic.Int32
}

func (w *openWorld) Grid() core.Grid { return w.grid }

func (w *openWorld) IsWalkable(c core.Cell) bool { return w.grid.Contains(c) }

func (w *openWorld) SpawnPoint() (core.Cell, error) {
	return w.spawns[int(w.joins.Add(1)-1)%len(w.spawns)], nil
}

func (w *openWorld) Path(from, to core.Cell) []core.Cell {
	if from.Row != to.Row || from == to || !w.IsWalkable(to) {
		return nil
	}
	var path []core.Cell
	for step := 1; from.Col != to.Col; {
		if to.Col < from.Col {
			step = -1
		}
		from.Col += step
		path = append(path, from)
	}
	return path
}

// Chunk is grass everywhere, with a rock on the chunk's first cell.
func (w *openWorld) Chunk(key core.ChunkKey) (*core.ChunkContents, bool) {
	if !w.grid.HasChunk(key) {
		return nil, false
	}
	return &core.ChunkContents{
		Key: key, Version: 1, Kinds: []string{"grass"},
		Ground: make([]uint8, core.CellsPerChunk), Height: make([]int8, core.CellsPerChunk), Region: make([]uint8, core.CellsPerChunk),
		Objects: []core.PlacedObject{{ID: "r" + key.String(), Kind: "rock", Cell: key.Origin()}},
	}, true
}

// newServer serves the websocket of two maps. "meadow" is 5 x 5: one chunk, everyone spawns in its center.
// "world" is 1024 x 512 and wraps: players spawn on (40, 40), then (300, 40), then (70, 40), in turn.
// Rooms tick for real until the test ends.
func newServer(t *testing.T) *httptest.Server {
	return newLoggedServer(t, logs.Discard())
}

func newLoggedServer(t *testing.T, log *slog.Logger) *httptest.Server {
	t.Helper()
	maps := map[string]app.World{
		"meadow": &openWorld{grid: core.Grid{Width: 5, Height: 5}, spawns: []core.Cell{{Col: 2, Row: 2}}},
		"world": &openWorld{grid: core.Grid{Width: 1024, Height: 512, Wraps: true},
			spawns: []core.Cell{{Col: 40, Row: 40}, {Col: 300, Row: 40}, {Col: 70, Row: 40}}},
	}
	hub := app.NewHub(t.Context(), func(_ context.Context, slug string) (app.World, error) {
		if world, ok := maps[slug]; ok {
			return world, nil
		}
		return nil, fmt.Errorf("no map %q: %w", slug, errd.ErrNotFound)
	}, time.Now, log)
	mux := http.NewServeMux()
	mux.Handle("GET /api/maps/{slug}/ws", wstransport.NewHandler(hub, []string{"localhost:5180"}, log))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// lockedBuffer collects logs written from several goroutines.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Every action is logged, and the guest ids, which are secrets, never are.
func TestEveryActionIsLoggedWithoutTheGuestID(t *testing.T) {
	var out lockedBuffer
	log, err := logs.New("debug", &out)
	require.NoError(t, err)
	server := newLoggedServer(t, log)

	conn := join(t, server, ann, "Ann")
	write(t, conn, `{"type": "dance"}`)
	write(t, conn, `{"type": "plan", "col": 4, "row": 2}`)
	read(t, conn)
	write(t, conn, `{"type": "walk", "col": 3, "row": 2}`)
	read(t, conn) // walking
	read(t, conn) // moved
	require.NoError(t, conn.Close(websocket.StatusNormalClosure, ""))
	require.Eventually(t, func() bool { return strings.Contains(out.String(), "ws closed") }, 2*time.Second, 10*time.Millisecond)

	logged := out.String()
	for _, action := range []string{"ws connecting", "room opened", "ws open", "msg=join", "chunks sent", "ws message ignored",
		"plan: path found", "walk: path set", "msg=step", "step: arrived", "msg=leave", "ws closed"} {
		assert.Contains(t, logged, action)
	}
	assert.Contains(t, logged, annPublic)
	assert.NotContains(t, logged, ann, "the guest id is a secret")
}

func dial(t *testing.T, server *httptest.Server, slug, player, name, origin string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	return dialQuery(t, server, slug, url.Values{"player": {player}, "name": {name}}, origin)
}

func dialQuery(t *testing.T, server *httptest.Server, slug string, query url.Values, origin string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	address := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/maps/" + slug + "/ws?" + query.Encode()
	conn, res, err := websocket.Dial(context.Background(), address, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": {origin}},
	})
	if conn != nil {
		t.Cleanup(func() { conn.CloseNow() })
	}
	return conn, res, err
}

// join connects a player to "meadow", and reads its welcome and its only chunk.
func join(t *testing.T, server *httptest.Server, player, name string) *websocket.Conn {
	t.Helper()
	conn, _ := joinMap(t, server, "meadow", player, name, 1)
	return conn
}

// joinMap connects a player to a map, and reads its welcome and its first chunks.
func joinMap(t *testing.T, server *httptest.Server, slug, player, name string, chunks int) (*websocket.Conn, map[string]any) {
	t.Helper()
	conn, _, err := dial(t, server, slug, player, name, front)
	require.NoError(t, err)
	welcome := read(t, conn)
	assert.Equal(t, "welcome", welcome["type"])
	assert.Equal(t, float64(domain.StepInterval.Milliseconds()), welcome["stepMs"])
	for range chunks {
		assert.Equal(t, "chunk", read(t, conn)["type"])
	}
	return conn, welcome
}

func read(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var msg map[string]any
	require.NoError(t, wsjson.Read(ctx, conn, &msg))
	return msg
}

func readText(t *testing.T, conn *websocket.Conn) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, data, err := conn.Read(ctx)
	require.NoError(t, err)
	return string(data)
}

func write(t *testing.T, conn *websocket.Conn, text string) {
	t.Helper()
	require.NoError(t, conn.Write(context.Background(), websocket.MessageText, []byte(text)))
}

func cellJSON(col, row int) map[string]any {
	return map[string]any{"col": float64(col), "row": float64(row)}
}

func movedJSON(id string, col, row int) map[string]any {
	return map[string]any{"type": "moved", "id": id, "col": float64(col), "row": float64(row)}
}

// ADR 0002 test 15, ADR 0001 test 19, and ADR 0006 test 5: no robot is sent as ""
func TestPlayersSeeEachOtherJoinWalkAndLeave(t *testing.T) {
	server := newServer(t)
	bobConn := join(t, server, bob, "Bob")
	annConn := join(t, server, ann, "Ann")
	assert.Equal(t, map[string]any{
		"type": "joined", "player": map[string]any{"id": annPublic, "name": "Ann", "robot": "", "col": 2.0, "row": 2.0},
	}, read(t, bobConn))

	write(t, annConn, `{"type": "plan", "col": 4, "row": 2}`)
	path := []any{cellJSON(3, 2), cellJSON(4, 2)}
	assert.Equal(t, map[string]any{"type": "planned", "col": 4.0, "row": 2.0, "path": path}, read(t, annConn))

	write(t, annConn, `{"type": "walk", "col": 4, "row": 2}`)
	assert.Equal(t, map[string]any{"type": "walking", "path": path}, read(t, annConn))
	for _, conn := range []*websocket.Conn{annConn, bobConn} {
		assert.Equal(t, movedJSON(annPublic, 3, 2), read(t, conn), "bob got nothing from the plan: his next message is a step")
		assert.Equal(t, movedJSON(annPublic, 4, 2), read(t, conn))
	}

	require.NoError(t, annConn.Close(websocket.StatusNormalClosure, ""))
	assert.Equal(t, map[string]any{"type": "left", "id": annPublic}, read(t, bobConn))
}

// ADR 0002 test 10, over the wire: an empty path is [], not null
func TestNoPathIsAnEmptyList(t *testing.T) {
	conn := join(t, newServer(t), ann, "Ann")
	write(t, conn, `{"type": "plan", "col": 0, "row": 0}`)
	assert.JSONEq(t, `{"type":"planned","col":0,"row":0,"path":[]}`, readText(t, conn))
	write(t, conn, `{"type": "walk", "col": 9, "row": 9}`)
	assert.JSONEq(t, `{"type":"walking","path":[]}`, readText(t, conn))
}

// Security review: a guest id is a secret, or anyone could connect with it and play that player.
func TestNoMessageCarriesAGuestID(t *testing.T) {
	server := newServer(t)
	bobConn := join(t, server, bob, "Bob")
	annConn, _, err := dial(t, server, "meadow", ann, "Ann", front)
	require.NoError(t, err)

	texts := []string{readText(t, annConn), readText(t, annConn), readText(t, bobConn)} // ann's welcome and chunk, bob told ann joined
	write(t, annConn, `{"type": "walk", "col": 3, "row": 2}`)
	texts = append(texts, readText(t, annConn), readText(t, bobConn))
	for _, text := range texts {
		assert.NotContains(t, text, ann)
		assert.NotContains(t, text, bob)
	}
	assert.Contains(t, texts[0], annPublic)
}

// ADR 0001 test 20
func TestRefusedBeforeTheUpgrade(t *testing.T) {
	server := newServer(t)
	cases := []struct {
		name, slug, player, playerName, origin string
		status                                 int
	}{
		{"a bad player id", "meadow", "not-a-uuid", "Ann", front, http.StatusBadRequest},
		{"an empty name", "meadow", ann, "  ", front, http.StatusBadRequest},
		{"a name too long", "meadow", ann, strings.Repeat("a", 25), front, http.StatusBadRequest},
		{"an unknown map", "nowhere", ann, "Ann", front, http.StatusNotFound},
		{"a foreign origin", "meadow", ann, "Ann", "http://evil.example", http.StatusForbidden},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, res, err := dial(t, server, c.slug, c.player, c.playerName, c.origin)
			require.Error(t, err)
			require.NotNil(t, res)
			assert.Equal(t, c.status, res.StatusCode)
		})
	}
}

// ADR 0006 test 4
func TestAMalformedRobotIsRefusedBeforeTheUpgrade(t *testing.T) {
	server := newServer(t)
	for _, robot := range []string{"Dome", "dome head", "<b>", strings.Repeat("a", 33)} {
		query := url.Values{"player": {ann}, "name": {"Ann"}, "robot": {robot}}
		_, res, err := dialQuery(t, server, "meadow", query, front)
		require.Error(t, err, robot)
		require.NotNil(t, res, robot)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode, robot)
	}
}

// ADR 0006 test 6
func TestWelcomeAndJoinedCarryTheRobots(t *testing.T) {
	server := newServer(t)
	bobConn, _, err := dialQuery(t, server, "meadow", url.Values{"player": {bob}, "name": {"Bob"}, "robot": {"rusty-toy"}}, front)
	require.NoError(t, err)
	read(t, bobConn) // welcome
	read(t, bobConn) // chunk

	annConn, _, err := dialQuery(t, server, "meadow", url.Values{"player": {ann}, "name": {"Ann"}, "robot": {"dome-head"}}, front)
	require.NoError(t, err)

	robots := map[string]any{}
	for _, p := range read(t, annConn)["players"].([]any) {
		player := p.(map[string]any)
		robots[player["name"].(string)] = player["robot"]
	}
	assert.Equal(t, map[string]any{"Ann": "dome-head", "Bob": "rusty-toy"}, robots)
	joined := read(t, bobConn)
	assert.Equal(t, "joined", joined["type"])
	assert.Equal(t, "dome-head", joined["player"].(map[string]any)["robot"])
}

// ADR 0002 test 16, and ADR 0001 test 21
func TestBadMessages(t *testing.T) {
	server := newServer(t)
	conn := join(t, server, ann, "Ann")

	t.Run("bad JSON, unknown types and the old move are ignored, the connection stays", func(t *testing.T) {
		write(t, conn, `not json`)
		write(t, conn, `{"type": "dance"}`)
		write(t, conn, `{"type": "move", "col": 3, "row": 2}`)
		write(t, conn, `{"type": "plan", "col": 3}`)
		write(t, conn, `{"type": "plan", "col": 3, "row": 2}`)
		assert.Equal(t, "planned", read(t, conn)["type"], "the first answer is the valid plan's: nothing moved")
	})

	t.Run("a message over 1 KB closes the connection", func(t *testing.T) {
		write(t, conn, `{"type": "plan", "pad": "`+strings.Repeat("x", 1100)+`"}`)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _, err := conn.Read(ctx)
		assert.Equal(t, websocket.StatusMessageTooBig, websocket.CloseStatus(err))
	})
}

// ADR 0003 test 22
func TestAPlayerGetsTheWelcomeThenTheChunksAround(t *testing.T) {
	conn, _, err := dial(t, newServer(t), "world", ann, "Ann", front)
	require.NoError(t, err)

	assert.Equal(t, "welcome", read(t, conn)["type"])
	first := readText(t, conn)
	var chunk struct {
		Type    string `json:"type"`
		CX, CY  int
		Version int              `json:"version"`
		Kinds   []string         `json:"kinds"`
		Ground  []int            `json:"ground"`
		Height  []int            `json:"height"`
		Region  []int            `json:"region"`
		Objects []map[string]any `json:"objects"`
	}
	require.NoError(t, json.Unmarshal([]byte(first), &chunk))
	assert.Equal(t, "chunk", chunk.Type)
	assert.Equal(t, [2]int{1, 1}, [2]int{chunk.CX, chunk.CY}, "the player's own chunk first: (40, 40) is in chunk (1, 1)")
	assert.Equal(t, 1, chunk.Version)
	assert.Equal(t, []string{"grass"}, chunk.Kinds)
	assert.Len(t, chunk.Ground, 1024, "numbers, not base64")
	assert.Len(t, chunk.Height, 1024)
	assert.Len(t, chunk.Region, 1024)
	assert.Equal(t, []map[string]any{{"id": "r[1,1]", "kind": "rock", "col": 32.0, "row": 32.0}}, chunk.Objects)
	for range 8 {
		assert.Equal(t, "chunk", read(t, conn)["type"])
	}
}

// ADR 0003 test 23
func TestOnlyPlayersCloseEnoughSeeAWalk(t *testing.T) {
	server := newServer(t)
	annConn, _ := joinMap(t, server, "world", ann, "Ann", 9) // spawns on (40, 40)
	farConn, _ := joinMap(t, server, "world", bob, "Bob", 9) // spawns on (300, 40): far
	carl := "00000000-0000-0000-0000-00000000000c"
	nearConn, welcome := joinMap(t, server, "world", carl, "Carl", 9) // spawns on (70, 40): near ann
	assert.Len(t, welcome["players"], 2, "carl sees ann and himself, not bob")
	assert.Equal(t, "joined", read(t, annConn)["type"], "ann sees carl come")

	write(t, annConn, `{"type": "walk", "col": 41, "row": 40}`)
	assert.Equal(t, "walking", read(t, annConn)["type"])
	assert.Equal(t, movedJSON(annPublic, 41, 40), read(t, annConn))
	assert.Equal(t, movedJSON(annPublic, 41, 40), read(t, nearConn), "carl holds ann's chunk")

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, _, err := farConn.Read(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded, "bob is told nothing")
}

// Security review: compression doesn't get around the 1 KB limit. A message that compresses to a few bytes but
// inflates past 1 KB still closes the connection.
func TestACompressedMessageOverTheLimitClosesTheConnection(t *testing.T) {
	server := newServer(t)
	address := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/maps/meadow/ws?" + url.Values{"player": {ann}, "name": {"Ann"}}.Encode()
	conn, res, err := websocket.Dial(context.Background(), address, &websocket.DialOptions{
		HTTPHeader:      http.Header{"Origin": {front}},
		CompressionMode: websocket.CompressionNoContextTakeover,
	})
	require.NoError(t, err)
	t.Cleanup(func() { conn.CloseNow() })
	require.Contains(t, res.Header.Get("Sec-WebSocket-Extensions"), "permessage-deflate", "the server compresses")
	read(t, conn) // welcome
	read(t, conn) // the chunk

	write(t, conn, `{"type": "plan", "pad": "`+strings.Repeat("x", 100_000)+`"}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _, err = conn.Read(ctx)
	assert.Equal(t, websocket.StatusMessageTooBig, websocket.CloseStatus(err))
}

// ADR 0010 tests 8 and 9
func TestTeleportOverTheWire(t *testing.T) {
	server := newServer(t)
	bobConn := join(t, server, bob, "Bob")
	annConn := join(t, server, ann, "Ann")
	read(t, bobConn) // ann joined

	t.Run("done: teleported to you, moved to those who see both cells", func(t *testing.T) {
		write(t, annConn, `{"type": "teleport", "map": "meadow", "col": 4, "row": 4}`)
		assert.Equal(t, map[string]any{"type": "teleported", "col": 4.0, "row": 4.0, "walkable": true}, read(t, annConn))
		assert.Equal(t, movedJSON(annPublic, 4, 4), read(t, bobConn))
	})

	t.Run("refused, with why", func(t *testing.T) {
		write(t, annConn, `{"type": "teleport", "map": "mars", "col": 1, "row": 1}`)
		assert.Equal(t, map[string]any{"type": "refused", "request": "teleport", "reason": "unknown map: mars"}, read(t, annConn))
		write(t, annConn, `{"type": "teleport", "map": "meadow", "col": 9, "row": 9}`)
		assert.Equal(t, map[string]any{"type": "refused", "request": "teleport", "reason": "(9, 9) is outside the map"}, read(t, annConn))
	})

	t.Run("malformed ones are ignored", func(t *testing.T) {
		write(t, annConn, `{"type": "teleport", "col": 1, "row": 1}`)
		write(t, annConn, `{"type": "teleport", "map": "meadow", "col": 1.5, "row": 1}`)
		write(t, annConn, `{"type": "teleport", "map": 5, "col": 1, "row": 1}`)
		write(t, annConn, `{"type": "plan", "col": 4, "row": 3}`)
		assert.Equal(t, "planned", read(t, annConn)["type"], "the first answer is the valid plan's: nothing moved")
	})
}

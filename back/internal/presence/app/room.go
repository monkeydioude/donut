// Package app holds the players' use-cases: join a map, plan and walk paths on it, leave it, all told live
// to the players near enough to see it.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	core "donut/internal/core/domain"
	"donut/internal/presence/domain"
)

// Client is one connection of a player, as a room sees it.
type Client interface {
	// Send queues an event for the player. False when the queue is full: the player reads too slowly.
	Send(domain.Event) bool
	// Close ends the connection. It must not block: the room calls it while holding its lock.
	Close()
}

// World is the room's map: where players may stand, how they get somewhere, and what they are sent of each chunk.
type World interface {
	Grid() core.Grid
	IsWalkable(core.Cell) bool
	SpawnPoint() (core.Cell, error)
	// Path is the cells to step on from `from` to `to`, the start excluded. Nil when there is none.
	Path(from, to core.Cell) []core.Cell
	// Chunk is what players are sent of a chunk. False for a chunk outside the map.
	Chunk(core.ChunkKey) (*core.ChunkContents, bool)
}

// Room is the players on one map. The server walks their paths, a step at a time, and tells each player what happens
// on the chunks they hold (ADR 0003). It is safe for concurrent use.
type Room struct {
	mu      sync.Mutex
	world   World
	now     func() time.Time
	log     *slog.Logger
	members map[string]*member
	// lastCells is where players who left were, so they come back there.
	lastCells map[string]core.Cell
	// slow are the members whose queue was full, dropped once the current change is told.
	slow []string
}

type member struct {
	player domain.Player
	client Client
	// held is what the connection holds: the chunks sent, not forgotten yet.
	held domain.HeldChunks
	// path is the cells still to step on; nextStepAt when the next step may come.
	path       []core.Cell
	nextStepAt time.Time
	limiter    domain.RequestLimiter
}

func NewRoom(world World, now func() time.Time, log *slog.Logger) *Room {
	return &Room{world: world, now: now, log: log, members: map[string]*member{}, lastCells: map[string]core.Cell{}}
}

// Run ticks the room every `every` until ctx ends.
func (r *Room) Run(ctx context.Context, every time.Duration) {
	r.log.Debug("room loop started", "tick", every, "step", domain.StepInterval)
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			r.log.Debug("room loop stopped", "reason", ctx.Err())
			return
		case <-ticker.C:
			r.Tick()
		}
	}
}

// Join puts a player on the map, welcomes them and sends them the chunks around them; the players who see them learn
// they joined. A player already there (another tab, a refresh) keeps their place and walk: the old connection is
// closed, and the new one gets everything again.
func (r *Room) Join(id, name string, client Client) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.dropSlow()

	if existing, ok := r.members[id]; ok {
		r.log.Debug("join: player already here, old connection replaced", "player", id, "name", name, "cell", existing.player.Cell, "walking", len(existing.path))
		existing.client.Close()
		existing.client = client
		existing.held = domain.HeldChunks{}
		r.welcome(existing)
		return nil
	}
	cell, returning, err := r.spawnCell(id)
	if err != nil {
		r.log.Warn("join: no cell to spawn on", "player", id, "error", err)
		return fmt.Errorf("Join: %w", err)
	}
	joiner := &member{player: domain.Player{ID: id, Name: name, Cell: cell}, client: client, held: domain.HeldChunks{},
		limiter: domain.NewRequestLimiter(r.now())}
	r.members[id] = joiner
	r.log.Debug("join", "player", id, "name", name, "cell", cell, "chunk", core.ChunkOf(cell), "back_where_they_left", returning, "players", len(r.members))
	r.welcome(joiner)
	for _, other := range r.others(id) {
		if other.held.Sees(cell) {
			r.log.Debug("in view: joined", "player", other.player.ID, "sees", id)
			r.send(other, domain.Joined{Player: joiner.player})
		}
	}
	return nil
}

// welcome gives a member the chunks around them, then tells them who they are and who they see, then sends the chunks.
func (r *Room) welcome(m *member) {
	received := m.held.Receive(r.world.Grid(), m.player.Cell)
	r.send(m, domain.Welcome{You: m.player.ID, Players: r.seenBy(m), Step: domain.StepInterval})
	r.sendChunks(m, received)
}

// Plan tells the player the path from where they stand to `to`, without moving them.
func (r *Room) Plan(id string, client Client, to core.Cell) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.dropSlow()

	if asker := r.requester(id, client, "plan", to); asker != nil {
		path := r.findPath(asker, to, "plan")
		r.send(asker, domain.Planned{To: to, Path: path})
	}
}

// Walk replaces the player's walk with the path to `to`, and tells them which. The next ticks walk it.
func (r *Room) Walk(id string, client Client, to core.Cell) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.dropSlow()

	if walker := r.requester(id, client, "walk", to); walker != nil {
		replaced := len(walker.path)
		walker.path = r.findPath(walker, to, "walk")
		r.log.Debug("walk: path set", "player", id, "steps", len(walker.path), "replaced_steps", replaced,
			"next_step_in", max(walker.nextStepAt.Sub(r.now()), 0))
		r.send(walker, domain.Walking{Path: walker.path})
	}
}

// findPath runs the pathfinding for a request and logs what it found and how long it took. A target on a chunk the
// player doesn't hold has no path: they can't see it.
func (r *Room) findPath(m *member, to core.Cell, request string) []core.Cell {
	if !r.world.Grid().Contains(to) || !m.held.Sees(to) {
		r.log.Debug(request+": no path, target outside the player's chunks", "player", m.player.ID, "from", m.player.Cell, "to", to)
		return nil
	}
	start := time.Now()
	path := r.world.Path(m.player.Cell, to)
	r.log.Debug(request+": path found", "player", m.player.ID, "from", m.player.Cell, "to", to,
		"steps", len(path), "reachable", len(path) > 0, "path", path, "took", time.Since(start))
	return path
}

// Tick steps every walker whose next step is due, and tells who sees it.
func (r *Room) Tick() {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.dropSlow()

	now := r.now()
	for _, id := range r.memberIDs() {
		if m, ok := r.members[id]; ok {
			r.step(m, now)
		}
	}
}

// Leave takes a player off the map when their connection ends. A replaced connection leaving changes nothing.
func (r *Room) Leave(id string, client Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.dropSlow()

	leaver, ok := r.members[id]
	if !ok || leaver.client != client {
		r.log.Debug("leave: ignored, connection already replaced or dropped", "player", id)
		return
	}
	r.log.Debug("leave", "player", id, "cell", leaver.player.Cell, "walk_dropped_steps", len(leaver.path), "players_left", len(r.members)-1)
	r.remove(leaver)
}

// requester is the member sending a request on this connection, nil when it's a replaced connection
// or the member sends too many.
func (r *Room) requester(id string, client Client, request string, to core.Cell) *member {
	m, ok := r.members[id]
	switch {
	case !ok || m.client != client:
		r.log.Debug(request+": ignored, connection replaced", "player", id, "to", to)
		return nil
	case !m.limiter.Allow(r.now()):
		r.log.Debug(request+": ignored, too many requests", "player", id, "to", to, "limit_per_second", domain.MaxRequestsPerSecond)
		return nil
	}
	r.log.Debug(request+": asked", "player", id, "from", m.player.Cell, "to", to)
	return m
}

func (r *Room) step(walker *member, now time.Time) {
	if len(walker.path) == 0 || now.Before(walker.nextStepAt) {
		return
	}
	next := walker.path[0]
	if err := domain.CheckStep(r.world.Grid(), walker.player.Cell, next, r.world.IsWalkable); err != nil {
		r.log.Debug("step: walk stopped", "player", walker.player.ID, "cell", walker.player.Cell, "next", next,
			"reason", err, "dropped_steps", len(walker.path))
		walker.path = nil
		r.send(walker, domain.Stopped{Cell: walker.player.Cell})
		return
	}
	from := walker.player.Cell
	walker.player.Cell, walker.path, walker.nextStepAt = next, walker.path[1:], now.Add(domain.StepInterval)
	r.log.Debug("step", "player", walker.player.ID, "from", from, "to", next, "steps_left", len(walker.path))
	if len(walker.path) == 0 {
		r.log.Debug("step: arrived", "player", walker.player.ID, "cell", next)
	}
	r.send(walker, domain.Moved{ID: walker.player.ID, Cell: next})
	if core.ChunkOf(from) != core.ChunkOf(next) {
		r.followWalker(walker)
	}
	for id, other := range r.members {
		if id != walker.player.ID {
			r.tellStep(other, walker, from)
		}
	}
}

// followWalker moves a walker's chunks with them: the chunks coming into range and the players on them, then the
// players on the chunks going out of range and those chunks.
func (r *Room) followWalker(walker *member) {
	grid, at := r.world.Grid(), walker.player.Cell
	received := walker.held.Receive(grid, at)
	r.sendChunks(walker, received)
	for _, other := range r.standingOn(received, walker.player.ID) {
		r.log.Debug("in view: joined", "player", walker.player.ID, "sees", other.player.ID, "cell", other.player.Cell)
		r.send(walker, domain.Joined{Player: other.player})
	}
	forgotten := walker.held.Forget(grid, at)
	for _, other := range r.standingOn(forgotten, walker.player.ID) {
		r.log.Debug("out of view: left", "player", walker.player.ID, "lost_sight_of", other.player.ID, "cell", other.player.Cell)
		r.send(walker, domain.Left{ID: other.player.ID})
	}
	if len(forgotten) > 0 {
		r.log.Debug("chunks forgotten", "player", walker.player.ID, "chunks", forgotten, "held", len(walker.held))
	}
	for _, key := range forgotten {
		r.send(walker, domain.ChunkForgotten{Key: key})
	}
}

// tellStep tells an observer about a walker's step from `from`, depending on which of the two cells they see.
func (r *Room) tellStep(observer, walker *member, from core.Cell) {
	sawFrom, seesTo := observer.held.Sees(from), observer.held.Sees(walker.player.Cell)
	switch {
	case sawFrom && seesTo:
		r.send(observer, domain.Moved{ID: walker.player.ID, Cell: walker.player.Cell})
	case seesTo:
		r.log.Debug("in view: joined", "player", observer.player.ID, "sees", walker.player.ID, "cell", walker.player.Cell)
		r.send(observer, domain.Joined{Player: walker.player})
	case sawFrom:
		r.log.Debug("out of view: left", "player", observer.player.ID, "lost_sight_of", walker.player.ID, "cell", walker.player.Cell)
		r.send(observer, domain.Left{ID: walker.player.ID})
	}
}

// sendChunks sends a member the contents of chunks they just received.
func (r *Room) sendChunks(m *member, keys []core.ChunkKey) {
	if len(keys) == 0 {
		return
	}
	r.log.Debug("chunks sent", "player", m.player.ID, "chunks", keys, "held", len(m.held))
	for _, key := range keys {
		if contents, ok := r.world.Chunk(key); ok {
			r.send(m, domain.ChunkShown{Chunk: contents})
		}
	}
}

// spawnCell is where a joining player appears, and whether it's where they left.
func (r *Room) spawnCell(id string) (core.Cell, bool, error) {
	if cell, ok := r.lastCells[id]; ok && r.world.IsWalkable(cell) {
		return cell, true, nil
	}
	cell, err := r.world.SpawnPoint()
	return cell, false, err
}

// seenBy is the players a member sees, them included, by id.
func (r *Room) seenBy(m *member) []domain.Player {
	players := []domain.Player{}
	for _, other := range r.members {
		if m.held.Sees(other.player.Cell) {
			players = append(players, other.player)
		}
	}
	slices.SortFunc(players, func(a, b domain.Player) int { return strings.Compare(a.ID, b.ID) })
	return players
}

// standingOn is the members but `except` standing on these chunks, by id.
func (r *Room) standingOn(keys []core.ChunkKey, except string) []*member {
	var on []*member
	for _, other := range r.others(except) {
		if slices.Contains(keys, core.ChunkOf(other.player.Cell)) {
			on = append(on, other)
		}
	}
	return on
}

// others is every member but `except`, by id.
func (r *Room) others(except string) []*member {
	others := make([]*member, 0, len(r.members))
	for _, id := range r.memberIDs() {
		if id != except {
			others = append(others, r.members[id])
		}
	}
	return others
}

// memberIDs in a fixed order, so walkers step in the same order every tick.
func (r *Room) memberIDs() []string {
	ids := make([]string, 0, len(r.members))
	for id := range r.members {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// remove takes a member off the map: the players who saw them learn they left.
func (r *Room) remove(m *member) {
	delete(r.members, m.player.ID)
	r.lastCells[m.player.ID] = m.player.Cell
	for _, other := range r.members {
		if other.held.Sees(m.player.Cell) {
			r.send(other, domain.Left{ID: m.player.ID})
		}
	}
}

func (r *Room) send(m *member, event domain.Event) {
	if !m.client.Send(event) {
		r.log.Debug("send: queue full", "player", m.player.ID, "event", fmt.Sprintf("%T", event))
		r.slow = append(r.slow, m.player.ID)
	}
}

// dropSlow disconnects the members too slow to read. Telling the others may find more slow ones.
func (r *Room) dropSlow() {
	for len(r.slow) > 0 {
		id := r.slow[0]
		r.slow = r.slow[1:]
		if m, ok := r.members[id]; ok {
			r.log.Warn("slow client disconnected", "player", id, "cell", m.player.Cell)
			m.client.Close()
			r.remove(m)
		}
	}
}

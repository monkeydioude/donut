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

// World is the room's map: where players may stand, how they get somewhere, what they are sent of each chunk, and
// its doors (ADR 0012). It never changes once the room opened, so it may be read without the room's lock.
type World interface {
	Grid() core.Grid
	IsWalkable(core.Cell) bool
	SpawnPoint() (core.Cell, error)
	// Path is the cells to step on from `from` to `to`, the start excluded. Nil when there is none.
	Path(from, to core.Cell) []core.Cell
	// Chunk is what players are sent of a chunk. False for a chunk outside the map.
	Chunk(core.ChunkKey) (*core.ChunkContents, bool)
	// Info is what a player is told of the map on arriving.
	Info() core.MapInfo
	// DomeAt is the dome covering a cell, and DomeLeadingTo the dome leading to a map; false when there's none.
	DomeAt(core.Cell) (core.Dome, bool)
	DomeLeadingTo(slug string) (core.Dome, bool)
	// Edge is a city's edge: its walkable cells touching its wall. Empty on a map without walls.
	Edge() []core.Cell
	// IsOutside: in a city, a click on this cell means leaving. Never on a wrapping map.
	IsOutside(core.Cell) bool
}

// Traveler is a player going from one room to another (ADR 0012): who, on which connection, where to, and whether
// they asked a teleport, which they are told once there.
type Traveler struct {
	Player   domain.Player
	Client   Client
	To       domain.Destination
	Teleport bool
	// From is the map they left, for the logs.
	From string
}

// Room is the players on one map. The server walks their paths, a step at a time, and tells each player what happens
// on the chunks they hold (ADR 0003). It is safe for concurrent use.
type Room struct {
	mu sync.Mutex
	// slug names the room's map.
	slug    string
	world   World
	now     func() time.Time
	log     *slog.Logger
	members map[string]*member
	// lastCells is where players who left were, so they come back there.
	lastCells map[string]core.Cell
	// slow are the members whose queue was full, dropped once the current change is told.
	slow []string
	// leaving are the players gone through a door or teleported to another map, handed to `travel` once the lock is
	// released: a room never holds another room's lock (ADR 0012).
	leaving []Traveler
	travel  func(Traveler)
}

type member struct {
	player domain.Player
	client Client
	// held is what the connection holds: the chunks sent, not forgotten yet.
	held domain.HeldChunks
	// path is the cells still to step on; nextStepAt when the next step may come.
	path       []core.Cell
	nextStepAt time.Time
	// then is where the walk leads once it ends: through a door to another map. Nil for a plain walk.
	then *domain.Destination
}

// NewRoom opens the room of a map. `travel` takes the players going to another map, after the room's lock is released;
// nil drops them.
func NewRoom(slug string, world World, now func() time.Time, log *slog.Logger, travel func(Traveler)) *Room {
	if travel == nil {
		travel = func(t Traveler) { log.Warn("travel: nowhere to go, dropped", "player", t.Player.ID, "to", t.To) }
	}
	return &Room{slug: slug, world: world, now: now, log: log, members: map[string]*member{}, lastCells: map[string]core.Cell{}, travel: travel}
}

// Slug names the room's map. World is its map, read without the lock: it never changes.
func (r *Room) Slug() string { return r.slug }
func (r *Room) World() World { return r.world }

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

// Join puts a player on the map's spawn, welcomes them and sends them the chunks around them; the players who see them
// learn they joined. A player already there (another tab, a refresh) keeps their place and walk: see reconnect.
func (r *Room) Join(who domain.Profile, client Client) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.dropSlow()

	if existing, ok := r.members[who.ID]; ok {
		r.reconnect(existing, who, client)
		return nil
	}
	cell, err := r.world.SpawnPoint()
	if err != nil {
		r.log.Warn("join: no cell to spawn on", "player", who.ID, "error", err)
		return fmt.Errorf("Join: %w", err)
	}
	r.place(who, client, cell, "join: at the spawn")
	return nil
}

// Return puts a player back where they left this map (ADR 0012): where they stand when they're here on another
// connection, else the cell they left from. False, with nothing done, when there's none or it isn't walkable anymore.
func (r *Room) Return(who domain.Profile, client Client) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.dropSlow()

	if existing, ok := r.members[who.ID]; ok {
		r.reconnect(existing, who, client)
		return true
	}
	cell, remembered := r.lastCells[who.ID]
	if !remembered || !r.world.IsWalkable(cell) {
		r.log.Debug("return: nowhere walkable to come back to", "player", who.ID, "remembered", remembered, "cell", cell)
		return false
	}
	r.place(who, client, cell, "return: where they left")
	return true
}

// reconnect moves a member to a new connection: the old one is closed, and the new one gets everything again. They
// keep their place, walk, name and robot.
func (r *Room) reconnect(existing *member, who domain.Profile, client Client) {
	r.log.Debug("join: player already here, old connection replaced", "player", who.ID, "name", who.Name, "robot", who.Robot,
		"kept_robot", existing.player.Robot, "cell", existing.player.Cell, "walking", len(existing.path))
	existing.client.Close()
	existing.client = client
	existing.held = domain.HeldChunks{}
	r.welcome(existing)
}

// place stands a new member on a cell, welcomes them, and tells the players who see them.
func (r *Room) place(who domain.Profile, client Client, cell core.Cell, why string) {
	joiner := &member{player: domain.Player{ID: who.ID, Name: who.Name, Robot: who.Robot, Cell: cell}, client: client, held: domain.HeldChunks{}}
	r.members[who.ID] = joiner
	r.log.Debug(why, "player", who.ID, "name", who.Name, "robot", who.Robot, "cell", cell, "chunk", core.ChunkOf(cell), "players", len(r.members))
	r.welcome(joiner)
	r.tellArrival(joiner)
}

// tellArrival: the players who see a newcomer's cell learn they came.
func (r *Room) tellArrival(m *member) {
	for _, other := range r.others(m.player.ID) {
		if other.held.Sees(m.player.Cell) {
			r.log.Debug("in view: joined", "player", other.player.ID, "sees", m.player.ID)
			r.send(other, domain.Joined{Player: m.player})
		}
	}
}

// Admit takes a traveler from another map (ADR 0012): they stand on their cell, are welcomed with this map and its
// chunks, and told they were teleported if they asked; the players who see them learn they came. The same player
// already here on another connection is replaced.
func (r *Room) Admit(t Traveler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.dropSlow()

	id := t.Player.ID
	if existing, ok := r.members[id]; ok {
		r.log.Debug("admit: player already here, replaced by the traveler", "player", id)
		existing.client.Close()
		r.remove(existing)
	}
	arriving := &member{player: t.Player, client: t.Client, held: domain.HeldChunks{}}
	r.members[id] = arriving
	r.log.Debug("travel: arrived", "player", id, "from", t.From, "cell", t.Player.Cell, "teleport", t.Teleport, "players", len(r.members))
	r.welcome(arriving)
	if t.Teleport {
		r.send(arriving, domain.Teleported{Cell: t.Player.Cell, Walkable: r.world.IsWalkable(t.Player.Cell)})
	}
	r.tellArrival(arriving)
}

// Remember keeps where a player who isn't here would come back: one who arrived here as their connection ended.
func (r *Room) Remember(id string, c core.Cell) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastCells[id] = c
}

// welcome gives a member the chunks around them, then tells them the map, who they are and who they see, then sends
// the chunks.
func (r *Room) welcome(m *member) {
	received := m.held.Receive(r.world.Grid(), m.player.Cell)
	r.send(m, domain.Welcome{You: m.player.ID, Players: r.seenBy(m), Step: domain.StepInterval, Map: r.world.Info()})
	r.sendChunks(m, received)
}

// Plan tells the player the path from where they stand to `to`, without moving them, and what going there does
// (ADR 0012): a dome's cell is entered from beside it, a click outside a city leaves it from its edge.
func (r *Room) Plan(id string, client Client, to core.Cell) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.dropSlow()

	if asker := r.requester(id, client, "plan", to); asker != nil {
		action, path := r.planFor(asker, to)
		r.send(asker, domain.Planned{To: to, Path: path, Action: action})
	}
}

// Walk replaces the player's walk with the path to `to`, and tells them which. The next ticks walk it.
func (r *Room) Walk(id string, client Client, to core.Cell) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.dropSlow()

	if walker := r.requester(id, client, "walk", to); walker != nil {
		replaced := len(walker.path)
		walker.path, walker.then = r.findPath(walker, to, "walk"), nil
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

// Tick steps every walker whose next step is due, and tells who sees it. Those arriving at a door go through it.
func (r *Room) Tick() {
	defer r.sendTravelers()
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

// requester is the member sending a request on this connection, nil when it's a replaced connection, or one that left
// for another map. The session counts requests (ADR 0012).
func (r *Room) requester(id string, client Client, request string, to core.Cell) *member {
	m, ok := r.members[id]
	if !ok || m.client != client {
		r.log.Debug(request+": ignored, connection replaced or gone", "player", id, "to", to)
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
		walker.path, walker.then = nil, nil
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
	r.arrived(walker, from)
	if len(walker.path) == 0 && walker.then != nil {
		r.depart(walker, *walker.then, false)
	}
}

// Teleport puts the player on a cell of this map, walkable or not, and drops their walk (ADR 0010). A cell outside
// the map is refused with why. Then everything goes as after a step: their chunks follow them, and the others see
// them go or come. A teleport to another map goes through the hub (ADR 0012).
func (r *Room) Teleport(id string, client Client, to core.Cell) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.dropSlow()

	mover := r.requester(id, client, "teleport", to)
	if mover == nil {
		return
	}
	if err := domain.CheckTeleport(r.world.Grid(), to); err != nil {
		r.log.Debug("teleport: refused", "player", id, "map", r.slug, "to", to, "reason", err)
		r.send(mover, domain.Refused{Request: "teleport", Reason: err.Error()})
		return
	}
	from, walkable := mover.player.Cell, r.world.IsWalkable(to)
	r.log.Debug("teleport: done", "player", id, "from", from, "to", to, "walkable", walkable, "dropped_steps", len(mover.path))
	mover.player.Cell, mover.path, mover.then = to, nil, nil
	r.send(mover, domain.Teleported{Cell: to, Walkable: walkable})
	r.arrived(mover, from)
}

// arrived follows a member who just moved from `from`: their chunks when the chunk changed, then what each other
// player sees of the move.
func (r *Room) arrived(mover *member, from core.Cell) {
	if core.ChunkOf(from) != core.ChunkOf(mover.player.Cell) {
		r.followWalker(mover)
	}
	for id, other := range r.members {
		if id != mover.player.ID {
			r.tellStep(other, mover, from)
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

package app

import (
	"slices"

	core "donut/internal/core/domain"
	"donut/internal/presence/domain"
)

// Arrival is where a walk through a door leads, from the cell the walk ends on: a map and a cell on it. False when
// there's no way through. The hub works it out from the other map, which never changes (ADR 0012).
type Arrival func(from core.Cell) (domain.Destination, bool)

// Enter walks the player to beside the dome on `to`, then into the city it leads to (ADR 0012). Standing beside it
// already, they go in at once. With no way to it, the walk stops, as for a walk to nowhere.
func (r *Room) Enter(id string, client Client, to core.Cell, arrival Arrival) {
	defer r.sendTravelers()
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.dropSlow()

	walker := r.requester(id, client, "enter", to)
	if walker == nil {
		return
	}
	dome, ok := r.world.DomeAt(to)
	if !ok || !walker.held.Sees(to) {
		r.refuse(walker, "enter", domain.NoDomeAt(to))
		return
	}
	side, path, ok := r.pathBeside(walker, dome)
	r.goThrough(walker, "enter", side, path, ok, arrival)
}

// Exit walks the player to the edge of their city closest to `clicked`, then out onto the world beside its dome (ADR
// 0012). Standing there already, they go out at once.
func (r *Room) Exit(id string, client Client, clicked core.Cell, arrival Arrival) {
	defer r.sendTravelers()
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.dropSlow()

	walker := r.requester(id, client, "exit", clicked)
	if walker == nil {
		return
	}
	edge, ok := domain.ClosestEdge(r.world.Edge(), r.world.Grid(), clicked)
	if !ok {
		r.refuse(walker, "exit", domain.NoWayOut(r.world.Info().Name))
		return
	}
	path, reached := r.pathTo(walker, edge)
	r.goThrough(walker, "exit", edge, path, reached, arrival)
}

// goThrough sends a walker through a door from `from`: at once when they stand there, else at the end of the path.
// No path stops their walk; no way through the door is refused.
func (r *Room) goThrough(walker *member, request string, from core.Cell, path []core.Cell, reached bool, arrival Arrival) {
	if !reached {
		r.log.Debug(request+": no way to the door", "player", walker.player.ID, "from", walker.player.Cell)
		walker.path, walker.then = nil, nil
		r.send(walker, domain.Walking{})
		return
	}
	destination, ok := arrival(from)
	if !ok {
		r.refuse(walker, request, domain.NoWayOut(r.world.Info().Name))
		return
	}
	if len(path) == 0 {
		r.depart(walker, destination, false)
		return
	}
	walker.path, walker.then = path, &destination
	r.log.Debug(request+": walking to the door", "player", walker.player.ID, "door", from, "steps", len(path), "then", destination)
	r.send(walker, domain.Walking{Path: path})
}

// TeleportAway sends the player to a cell of another map, which the hub checked (ADR 0012): they leave this room now.
func (r *Room) TeleportAway(id string, client Client, to domain.Destination) {
	defer r.sendTravelers()
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.dropSlow()

	if mover := r.requester(id, client, "teleport", to.Cell); mover != nil {
		r.depart(mover, to, true)
	}
}

// Refuse tells the player a request they made wasn't done, and why: one the hub refused before this room could
// (ADR 0012).
func (r *Room) Refuse(id string, client Client, request string, why error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.dropSlow()

	if asker := r.requester(id, client, request, core.Cell{}); asker != nil {
		r.refuse(asker, request, why)
	}
}

func (r *Room) refuse(m *member, request string, why error) {
	r.log.Debug(request+": refused", "player", m.player.ID, "reason", why)
	r.send(m, domain.Refused{Request: request, Reason: why.Error()})
}

// planFor is what a click on `to` does, and the path it takes: beside a dome to enter it, to a city's edge to leave
// it, or to the cell itself. For a door one stands at already, the path is the cell one stands on.
func (r *Room) planFor(m *member, to core.Cell) (domain.Action, []core.Cell) {
	if dome, ok := r.world.DomeAt(to); ok && m.held.Sees(to) {
		_, path, reached := r.pathBeside(m, dome)
		return domain.ActionEnter, doorPath(m, path, reached)
	}
	if r.world.IsOutside(to) {
		edge, ok := domain.ClosestEdge(r.world.Edge(), r.world.Grid(), to)
		if !ok {
			return domain.ActionExit, nil
		}
		path, reached := r.pathTo(m, edge)
		return domain.ActionExit, doorPath(m, path, reached)
	}
	return domain.ActionMove, r.findPath(m, to, "plan")
}

// doorPath is a planned path to a door: the cell one stands on when one goes through at once, nil without a way.
func doorPath(m *member, path []core.Cell, reached bool) []core.Cell {
	switch {
	case !reached:
		return nil
	case len(path) == 0:
		return []core.Cell{m.player.Cell}
	}
	return path
}

// pathBeside is the way to the cell beside a dome one goes in from: the closest by straight distance that a path
// reaches, or the one standing on. False when none is reached.
func (r *Room) pathBeside(m *member, dome core.Dome) (core.Cell, []core.Cell, bool) {
	grid, at := r.world.Grid(), m.player.Cell
	sides := dome.Around(grid)
	for _, side := range sides {
		if side == at {
			return side, nil, true
		}
	}
	sortByDistance(grid, sides, at)
	for _, side := range sides {
		if path, ok := r.pathTo(m, side); ok {
			return side, path, true
		}
	}
	return core.Cell{}, nil, false
}

// pathTo is the way from where the member stands to a walkable cell: empty when they stand on it, false when none.
func (r *Room) pathTo(m *member, to core.Cell) ([]core.Cell, bool) {
	if m.player.Cell == to {
		return nil, true
	}
	if !r.world.IsWalkable(to) {
		return nil, false
	}
	path := r.world.Path(m.player.Cell, to)
	return path, len(path) > 0
}

// depart takes a member off this map toward another: the players who saw them learn they left, and once the lock
// is released, the hub takes them on (ADR 0012). Their walk ends here.
func (r *Room) depart(m *member, to domain.Destination, teleport bool) {
	delete(r.members, m.player.ID)
	for _, other := range r.members {
		if other.held.Sees(m.player.Cell) {
			r.send(other, domain.Left{ID: m.player.ID})
		}
	}
	r.log.Debug("travel: left", "player", m.player.ID, "cell", m.player.Cell, "to", to, "teleport", teleport, "players_left", len(r.members))
	traveler := Traveler{Player: m.player, Client: m.client, To: to, Teleport: teleport, From: r.slug}
	traveler.Player.Cell = to.Cell
	r.leaving = append(r.leaving, traveler)
}

// sendTravelers hands the players who left for another map to the hub. It runs once the room's lock is released: the
// hub takes the other room's lock.
func (r *Room) sendTravelers() {
	r.mu.Lock()
	travelers := r.leaving
	r.leaving = nil
	r.mu.Unlock()
	for _, t := range travelers {
		r.travel(t)
	}
}

// sortByDistance orders cells by their straight distance from `at`, the short way round on a wrapping grid.
func sortByDistance(grid core.Grid, cells []core.Cell, at core.Cell) {
	distance := func(c core.Cell) int {
		d := grid.Offset(at, c)
		return d.Col*d.Col + d.Row*d.Row
	}
	slices.SortStableFunc(cells, func(a, b core.Cell) int { return distance(a) - distance(b) })
}

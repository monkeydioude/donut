package app

import (
	"context"
	"sync"
	"time"

	core "donut/internal/core/domain"
	"donut/internal/presence/domain"
)

// Session is a guest's connection: it counts their requests, and passes them to the room of the map they're on,
// which changes as they travel (ADR 0012). Its methods are called from the connection's reading loop, and moveTo
// from a room handing it a traveler.
type Session struct {
	hub    *Hub
	id     string
	client Client

	mu      sync.Mutex
	room    *Room
	limiter domain.RequestLimiter
	closed  bool
}

func newSession(hub *Hub, id string, client Client, now time.Time) *Session {
	return &Session{hub: hub, id: id, client: client, limiter: domain.NewRequestLimiter(now)}
}

// settle places the guest with `place`, which gives their room, before any request: a traveler handed to the session
// meanwhile waits until it's placed.
func (s *Session) settle(place func() (*Room, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	room, err := place()
	if err != nil {
		return err
	}
	s.room = room
	return nil
}

// Plan, Walk and Teleport within the map, as before (ADR 0002, ADR 0010).
func (s *Session) Plan(to core.Cell) {
	if room := s.allow("plan", to); room != nil {
		room.Plan(s.id, s.client, to)
	}
}

func (s *Session) Walk(to core.Cell) {
	if room := s.allow("walk", to); room != nil {
		room.Walk(s.id, s.client, to)
	}
}

// Teleport puts the guest on a cell of a map, walkable or not: another map is a move to its room, checked before
// anything moves (ADR 0012).
func (s *Session) Teleport(ctx context.Context, toMap string, to core.Cell) {
	room := s.allow("teleport", to)
	if room == nil {
		return
	}
	if toMap == room.Slug() {
		room.Teleport(s.id, s.client, to)
		return
	}
	if _, err := s.hub.destination(ctx, toMap, to); err != nil {
		room.Refuse(s.id, s.client, "teleport", err)
		return
	}
	room.TeleportAway(s.id, s.client, domain.Destination{Map: toMap, Cell: to})
}

// Enter walks the guest beside the dome on `to`, then into its city (ADR 0012).
func (s *Session) Enter(ctx context.Context, to core.Cell) {
	room := s.allow("enter", to)
	if room == nil {
		return
	}
	dome, ok := room.World().DomeAt(to)
	if !ok {
		room.Refuse(s.id, s.client, "enter", domain.NoDomeAt(to))
		return
	}
	city, err := s.hub.Room(ctx, dome.LeadsTo)
	if err != nil {
		room.Refuse(s.id, s.client, "enter", domain.UnknownMap(dome.LeadsTo))
		return
	}
	room.Enter(s.id, s.client, to, func(from core.Cell) (domain.Destination, bool) {
		cell, ok := domain.ArrivalInCity(city.World().Edge(), city.World().Grid(), dome, room.World().Grid(), from)
		return domain.Destination{Map: city.Slug(), Cell: cell}, ok
	})
}

// Exit walks the guest to their city's edge closest to `clicked`, then out beside its dome on the world (ADR 0012).
func (s *Session) Exit(ctx context.Context, clicked core.Cell) {
	room := s.allow("exit", clicked)
	if room == nil {
		return
	}
	city := room.World().Info()
	if city.Wraps {
		room.Refuse(s.id, s.client, "exit", domain.ErrNotInCity)
		return
	}
	world, dome, ok := s.domeOf(ctx, city)
	if !ok {
		room.Refuse(s.id, s.client, "exit", domain.StandsNowhere(city.Name))
		return
	}
	room.Exit(s.id, s.client, clicked, func(from core.Cell) (domain.Destination, bool) {
		cell, ok := domain.ArrivalBesideDome(world.World().Grid(), dome, world.World().IsWalkable, city.Grid(), from)
		return domain.Destination{Map: world.Slug(), Cell: cell}, ok
	})
}

// domeOf is the room of the map a city stands on, and the dome leading to the city there. False when it stands
// nowhere: no dome leads to it, as on a world older than ADR 0012.
func (s *Session) domeOf(ctx context.Context, city core.MapInfo) (*Room, core.Dome, bool) {
	if city.StandsOn == nil {
		return nil, core.Dome{}, false
	}
	world, err := s.hub.Room(ctx, city.StandsOn.Map)
	if err != nil {
		return nil, core.Dome{}, false
	}
	dome, ok := world.World().DomeLeadingTo(city.Slug)
	return world, dome, ok
}

// Leave takes the guest off their map: the connection ended.
func (s *Session) Leave() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.room.Leave(s.id, s.client)
	s.hub.dropLive(s)
}

// allow is the room to pass a request to, nil when the connection ended or sends too many: 10 per second, on every
// map (ADR 0002, ADR 0012).
func (s *Session) allow(request string, to core.Cell) *Room {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case s.closed:
		return nil
	case !s.limiter.Allow(s.hub.now()):
		s.hub.log.Debug(request+": ignored, too many requests", "player", s.id, "to", to, "limit_per_second", domain.MaxRequestsPerSecond)
		return nil
	}
	return s.room
}

// moveTo admits a traveler of this session into a room, which becomes theirs. False when the connection ended, or
// another replaced it: they aren't admitted.
func (s *Session) moveTo(room *Room, t Traveler) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.client != t.Client {
		return false
	}
	room.Admit(t)
	s.room = room
	return true
}

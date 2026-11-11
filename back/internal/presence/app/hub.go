package app

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	core "donut/internal/core/domain"
	"donut/internal/presence/domain"
)

// LoadWorld gives a map, ready for its room. An unknown map is errd.ErrNotFound.
type LoadWorld func(ctx context.Context, slug string) (World, error)

// Hub holds a room per map, opened when its first player comes, and which map each guest is on (ADR 0012). It moves
// players from room to room, one connection each. Rooms tick until the hub's context ends.
type Hub struct {
	ctx   context.Context
	mu    sync.Mutex
	rooms map[string]*Room
	load  LoadWorld
	now   func() time.Time
	log   *slog.Logger
	// home is the map a new guest joins.
	home string

	guestsMu sync.Mutex
	// maps is the map each guest was last on: where they are, or left from. A server restart forgets it.
	maps map[string]string
	// live is each connected guest's session.
	live map[string]*Session
}

// NewHub opens rooms with `load`; new guests join the map `home`.
func NewHub(ctx context.Context, load LoadWorld, now func() time.Time, home string, log *slog.Logger) *Hub {
	return &Hub{ctx: ctx, rooms: map[string]*Room{}, load: load, now: now, home: home, log: log.With("component", "presence"),
		maps: map[string]string{}, live: map[string]*Session{}}
}

// Room is the room of a map, with the map loaded once and its loop started.
func (h *Hub) Room(ctx context.Context, slug string) (*Room, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room, ok := h.rooms[slug]; ok {
		return room, nil
	}
	world, err := h.load(ctx, slug)
	if err != nil {
		h.log.DebugContext(ctx, "room not opened", "map", slug, "error", err)
		return nil, fmt.Errorf("Room: %w", err)
	}
	room := NewRoom(slug, world, h.now, h.log.With("map", slug), h.arrive)
	h.rooms[slug] = room
	h.log.InfoContext(ctx, "room opened", "map", slug, "rooms", len(h.rooms))
	go room.Run(h.ctx, domain.TickInterval)
	return room, nil
}

// Connect places a guest's connection (ADR 0012): back where they left, on the map they were last on; else, new or
// with that place gone, at the home map's spawn. The session gets their requests until they leave.
func (h *Hub) Connect(ctx context.Context, who domain.Profile, client Client) (*Session, error) {
	session := newSession(h, who.ID, client, h.now())
	h.setLive(session)
	if err := session.settle(func() (*Room, error) { return h.place(ctx, who, client) }); err != nil {
		h.dropLive(session)
		return nil, fmt.Errorf("Connect: %w", err)
	}
	return session, nil
}

// place puts a connecting guest back where they left, else at the home map's spawn, and gives their room.
func (h *Hub) place(ctx context.Context, who domain.Profile, client Client) (*Room, error) {
	if room, ok := h.placeBack(ctx, who, client); ok {
		return room, nil
	}
	home, err := h.Room(ctx, h.home)
	if err != nil {
		return nil, err
	}
	if err := home.Join(who, client); err != nil {
		return nil, err
	}
	h.setMap(who.ID, h.home)
	return home, nil
}

// placeBack puts a returning guest back where they left. False for a new guest, or when their map is gone or their
// cell isn't walkable anymore, logged with why.
func (h *Hub) placeBack(ctx context.Context, who domain.Profile, client Client) (*Room, bool) {
	slug, returning := h.mapOf(who.ID)
	if !returning {
		h.log.DebugContext(ctx, "connect: a new guest, at the home map's spawn", "player", who.ID, "home", h.home)
		return nil, false
	}
	room, err := h.Room(ctx, slug)
	if err != nil {
		h.log.WarnContext(ctx, "connect: the guest's map is gone, sent to the home map's spawn", "player", who.ID, "map", slug,
			"home", h.home, "error", err)
		return nil, false
	}
	if !room.Return(who, client) {
		h.log.DebugContext(ctx, "connect: where the guest left isn't walkable anymore, sent to the home map's spawn",
			"player", who.ID, "map", slug, "home", h.home)
		return nil, false
	}
	h.log.DebugContext(ctx, "connect: back where they left", "player", who.ID, "map", slug)
	return room, true
}

// arrive takes a traveler into the room of their map, opened before they left (ADR 0012). A connection that ended or
// was replaced on the way only leaves them there, for when they come back.
func (h *Hub) arrive(t Traveler) {
	h.mu.Lock()
	room, ok := h.rooms[t.To.Map]
	h.mu.Unlock()
	if !ok {
		h.log.Error("travel: the map's room isn't open, traveler dropped", "player", t.Player.ID, "to", t.To)
		return
	}
	h.setMap(t.Player.ID, t.To.Map)
	session := h.liveSession(t.Player.ID)
	if session == nil || !session.moveTo(room, t) {
		h.log.Debug("travel: the connection ended on the way, remembered there", "player", t.Player.ID, "to", t.To)
		room.Remember(t.Player.ID, t.To.Cell)
	}
}

// destination checks a cell of another map, opening its room: the map exists, and the cell is in it.
func (h *Hub) destination(ctx context.Context, slug string, cell core.Cell) (*Room, error) {
	room, err := h.Room(ctx, slug)
	if err != nil {
		return nil, domain.UnknownMap(slug)
	}
	if err := domain.CheckTeleport(room.World().Grid(), cell); err != nil {
		return nil, err
	}
	return room, nil
}

func (h *Hub) mapOf(id string) (string, bool) {
	h.guestsMu.Lock()
	defer h.guestsMu.Unlock()
	slug, ok := h.maps[id]
	return slug, ok
}

func (h *Hub) setMap(id, slug string) {
	h.guestsMu.Lock()
	defer h.guestsMu.Unlock()
	h.maps[id] = slug
}

func (h *Hub) liveSession(id string) *Session {
	h.guestsMu.Lock()
	defer h.guestsMu.Unlock()
	return h.live[id]
}

// setLive makes a session the guest's: a new connection replaces the old one, which the room closes.
func (h *Hub) setLive(s *Session) {
	h.guestsMu.Lock()
	defer h.guestsMu.Unlock()
	h.live[s.id] = s
}

// dropLive forgets a session that ended, unless another replaced it already.
func (h *Hub) dropLive(s *Session) {
	h.guestsMu.Lock()
	defer h.guestsMu.Unlock()
	if h.live[s.id] == s {
		delete(h.live, s.id)
	}
}

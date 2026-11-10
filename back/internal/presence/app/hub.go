package app

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"donut/internal/presence/domain"
)

// LoadWorld gives a map, ready for its room. An unknown map is errd.ErrNotFound.
type LoadWorld func(ctx context.Context, slug string) (World, error)

// Hub holds a room per map, opened when its first player comes. Rooms tick until the hub's context ends.
type Hub struct {
	ctx   context.Context
	mu    sync.Mutex
	rooms map[string]*Room
	load  LoadWorld
	now   func() time.Time
	log   *slog.Logger
}

func NewHub(ctx context.Context, load LoadWorld, now func() time.Time, log *slog.Logger) *Hub {
	return &Hub{ctx: ctx, rooms: map[string]*Room{}, load: load, now: now, log: log.With("component", "presence")}
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
	room := NewRoom(world, h.now, h.log.With("map", slug))
	h.rooms[slug] = room
	h.log.InfoContext(ctx, "room opened", "map", slug, "rooms", len(h.rooms))
	go room.Run(h.ctx, domain.TickInterval)
	return room, nil
}

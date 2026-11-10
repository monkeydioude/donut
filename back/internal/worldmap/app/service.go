// Package app holds the worldmap use-cases: list and describe maps, create the starter world, give a room its world.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"donut/internal/worldmap/domain"
	"donut/pkg/errd"
)

// Store is where maps are saved.
type Store interface {
	ListMaps(ctx context.Context) ([]domain.Summary, error)
	CountMaps(ctx context.Context) (int, error)
	DescriptionBySlug(ctx context.Context, slug string) (domain.Description, error)
	MapBySlug(ctx context.Context, slug string) (domain.Map, error)
	InsertMap(ctx context.Context, m domain.Map) (string, error)
}

type Service struct {
	store Store
	log   *slog.Logger
}

func NewService(store Store, log *slog.Logger) *Service {
	return &Service{store: store, log: log.With("component", "worldmap")}
}

func (s *Service) ListMaps(ctx context.Context) ([]domain.Summary, error) {
	maps, err := s.store.ListMaps(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListMaps: %w", err)
	}
	s.log.DebugContext(ctx, "maps listed", "count", len(maps))
	return maps, nil
}

// Description is a map's size and regions, without its cells. An unknown slug is errd.ErrNotFound.
func (s *Service) Description(ctx context.Context, slug string) (domain.Description, error) {
	d, err := s.store.DescriptionBySlug(ctx, slug)
	if err != nil {
		s.log.DebugContext(ctx, "map not described", "slug", slug, "error", err)
		return domain.Description{}, fmt.Errorf("Description: %w", err)
	}
	s.log.DebugContext(ctx, "map described", "slug", slug, "id", d.ID, "size", fmt.Sprintf("%dx%d", d.Width, d.Height), "wraps", d.Wraps, "regions", len(d.Regions))
	return d, nil
}

// World is a whole map, ready for a room: where one can walk, and what players are sent of each chunk.
// An unknown slug is errd.ErrNotFound.
func (s *Service) World(ctx context.Context, slug string) (domain.World, error) {
	start := time.Now()
	m, err := s.store.MapBySlug(ctx, slug)
	if err != nil {
		s.log.DebugContext(ctx, "world not loaded", "slug", slug, "error", err)
		return domain.World{}, fmt.Errorf("World: %w", err)
	}
	loaded := time.Now()
	world := domain.NewWorld(m)
	s.log.DebugContext(ctx, "world loaded", "slug", slug, "id", m.ID, "size", fmt.Sprintf("%dx%d", m.Width, m.Height), "wraps", m.Wraps,
		"chunks", len(m.Chunks), "regions", len(m.Regions), "objects", len(m.Objects),
		"read_took", loaded.Sub(start), "build_took", time.Since(loaded))
	return world, nil
}

// EnsureStarterMap generates and saves the starter world when there is no map at all. It tells if it created it.
// Two servers booting together may both try: the second one finds the slug taken and creates nothing.
func (s *Service) EnsureStarterMap(ctx context.Context, seed uint64) (bool, error) {
	count, err := s.store.CountMaps(ctx)
	if err != nil {
		return false, fmt.Errorf("EnsureStarterMap: %w", err)
	}
	if count > 0 {
		s.log.DebugContext(ctx, "starter map not needed", "maps", count)
		return false, nil
	}
	start := time.Now()
	m := domain.Generate(domain.StarterWorld(), rand.New(rand.NewPCG(seed, seed)))
	s.log.DebugContext(ctx, "starter world generated", "slug", m.Slug, "seed", seed, "size", fmt.Sprintf("%dx%d", m.Width, m.Height),
		"chunks", len(m.Chunks), "regions", len(m.Regions), "rocks", len(m.Objects), "took", time.Since(start))
	if err := m.Validate(); err != nil {
		return false, fmt.Errorf("EnsureStarterMap: %w", err)
	}
	if _, err := s.store.InsertMap(ctx, m); err != nil {
		if errors.Is(err, errd.ErrConflict) {
			s.log.InfoContext(ctx, "starter map created by another server meanwhile", "slug", m.Slug)
			return false, nil
		}
		return false, fmt.Errorf("EnsureStarterMap: %w", err)
	}
	s.log.InfoContext(ctx, "starter world created", "slug", m.Slug, "chunks", len(m.Chunks), "rocks", len(m.Objects), "took", time.Since(start))
	return true, nil
}

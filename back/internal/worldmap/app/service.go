// Package app holds the worldmap use-cases: list and describe maps, save the map files missing from the database,
// reset maps from their files, give a room its world.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"donut/internal/worldmap/domain"
	"donut/pkg/errd"
)

// Store is where maps are saved.
type Store interface {
	ListMaps(ctx context.Context) ([]domain.Summary, error)
	DescriptionBySlug(ctx context.Context, slug string) (domain.Description, error)
	MapBySlug(ctx context.Context, slug string) (domain.Map, error)
	InsertMap(ctx context.Context, m domain.Map) (string, error)
	ReplaceMap(ctx context.Context, m domain.Map) (id string, replaced bool, err error)
}

// ReadMapFile reads and checks the map file of a slug (ADR 0007): the server reads the ones built into it.
type ReadMapFile func(slug string) (domain.Map, error)

type Service struct {
	store  Store
	log    *slog.Logger
	images imageCache
}

func NewService(store Store, log *slog.Logger) *Service {
	return &Service{store: store, log: log.With("component", "worldmap"), images: imageCache{bySlug: map[string]MapImage{}}}
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

// EnsureMaps saves each map file of `slugs` the database doesn't have yet (ADR 0012), and gives the slugs it saved.
// Two servers booting together may both try: the second one finds the slug taken and saves nothing.
func (s *Service) EnsureMaps(ctx context.Context, slugs []string, read ReadMapFile) ([]string, error) {
	saved, err := s.store.ListMaps(ctx)
	if err != nil {
		return nil, fmt.Errorf("EnsureMaps: %w", err)
	}
	have := map[string]bool{}
	for _, m := range saved {
		have[m.Slug] = true
	}
	created := []string{}
	for _, slug := range slugs {
		if have[slug] {
			continue
		}
		ok, err := s.saveMapFile(ctx, slug, read)
		if err != nil {
			return created, fmt.Errorf("EnsureMaps: %w", err)
		}
		if ok {
			created = append(created, slug)
		}
	}
	s.log.DebugContext(ctx, "maps ensured", "files", slugs, "already_saved", len(saved), "created", created)
	return created, nil
}

// saveMapFile reads a map file and saves it as a new map. False when another server saved it meanwhile.
func (s *Service) saveMapFile(ctx context.Context, slug string, read ReadMapFile) (bool, error) {
	start := time.Now()
	m, err := s.readMapFile(ctx, slug, read)
	if err != nil {
		return false, err
	}
	if _, err := s.store.InsertMap(ctx, m); err != nil {
		if errors.Is(err, errd.ErrConflict) {
			s.log.InfoContext(ctx, "map created by another server meanwhile", "slug", m.Slug)
			return false, nil
		}
		return false, err
	}
	s.log.InfoContext(ctx, "map created from its file", "slug", m.Slug, "chunks", len(m.Chunks), "objects", len(m.Objects), "took", time.Since(start))
	return true, nil
}

// ResetMaps replaces each map of `slugs` with its file, one after the other, each in its own transaction. It stops at
// the first that fails: the maps before it are replaced, the ones after it untouched.
func (s *Service) ResetMaps(ctx context.Context, slugs []string, read ReadMapFile) error {
	for i, slug := range slugs {
		if err := s.ResetMap(ctx, slug, read); err != nil {
			s.log.WarnContext(ctx, "maps reset stopped", "failed", slug, "replaced", slugs[:i], "untouched", slugs[i+1:])
			return fmt.Errorf("ResetMaps: %w", err)
		}
	}
	return nil
}

// ResetMap replaces a map in the database with its file, in one transaction: its chunks, regions and rocks, the rocks
// with new ids. A map not in the database yet is saved. Rooms already open keep the map they loaded: run it with the
// server stopped.
func (s *Service) ResetMap(ctx context.Context, slug string, read ReadMapFile) error {
	start := time.Now()
	m, err := s.readMapFile(ctx, slug, read)
	if err != nil {
		return fmt.Errorf("ResetMap: %w", err)
	}
	id, replaced, err := s.store.ReplaceMap(ctx, m)
	if err != nil {
		return fmt.Errorf("ResetMap: %w", err)
	}
	s.log.InfoContext(ctx, "map reset from its file", "slug", slug, "id", id, "replaced", replaced,
		"chunks", len(m.Chunks), "regions", len(m.Regions), "rocks", len(m.Objects), "took", time.Since(start))
	return nil
}

func (s *Service) readMapFile(ctx context.Context, slug string, read ReadMapFile) (domain.Map, error) {
	start := time.Now()
	m, err := read(slug)
	if err != nil {
		s.log.WarnContext(ctx, "map file refused", "slug", slug, "error", err)
		return domain.Map{}, err
	}
	s.log.DebugContext(ctx, "map file read", "slug", slug, "size", fmt.Sprintf("%dx%d", m.Width, m.Height), "wraps", m.Wraps,
		"regions", len(m.Regions), "rocks", len(m.Objects), "took", time.Since(start))
	return m, nil
}

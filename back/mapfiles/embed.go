// Package mapfiles holds the starter maps, a folder each (ADR 0007), built into the binary so every server ships with
// them. `make world` generates the world's folder, which is then edited by hand.
package mapfiles

import (
	"embed"
	"fmt"
	"io/fs"

	"donut/internal/worldmap/domain"
	"donut/internal/worldmap/infra"
)

//go:embed world
var folders embed.FS

// Read is the map folder of a slug, read and checked. Its map.json must have that slug.
func Read(slug string) (domain.Map, error) {
	folder, err := fs.Sub(folders, slug)
	if err != nil {
		return domain.Map{}, fmt.Errorf("mapfiles.Read %q: %w", slug, err)
	}
	m, err := infra.DecodeMapFile(folder)
	if err != nil {
		return domain.Map{}, fmt.Errorf("mapfiles.Read %q: %w", slug, err)
	}
	if m.Slug != slug {
		return domain.Map{}, fmt.Errorf("mapfiles.Read %q: its map.json has the slug %q", slug, m.Slug)
	}
	return m, nil
}

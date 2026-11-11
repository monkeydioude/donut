// Package mapfiles holds the starter maps, a folder each (ADR 0007), built into the binary so every server ships with
// them. `make world` generates the world's folder and `make city` Doma-1's (ADR 0012); both are then edited by hand.
package mapfiles

import (
	"embed"
	"fmt"
	"io/fs"

	"donut/internal/worldmap/domain"
	"donut/internal/worldmap/infra"
)

//go:embed world doma-1
var folders embed.FS

// Slugs are the maps built into the binary, one per folder, sorted.
func Slugs() []string {
	entries, err := fs.ReadDir(folders, ".")
	if err != nil {
		panic(fmt.Sprintf("mapfiles.Slugs: the embedded folders can't be read: %v", err))
	}
	slugs := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			slugs = append(slugs, entry.Name())
		}
	}
	return slugs
}

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
	if err := infra.CheckDomes(m, folders); err != nil {
		return domain.Map{}, fmt.Errorf("mapfiles.Read %q: %w", slug, err)
	}
	return m, nil
}

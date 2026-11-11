package domain

import (
	core "donut/internal/core/domain"
)

// World is what a room needs of a map: where one can walk, what players are sent of each chunk, its description, its
// domes, and for a city its edge (ADR 0012). It is built once when the room opens and never changes, so it may be
// read without the room's lock.
type World struct {
	Terrain
	chunks map[core.ChunkKey]*core.ChunkContents
	info   core.MapInfo
	// domes are the map's domes, by each cell they cover.
	domes map[core.Cell]core.Dome
	edge  []core.Cell
	// localGround is each cell's ground, row by row, on a map that doesn't wrap only: a city's cells tell where its
	// edge is and which clicks mean leaving. Nil on a wrapping map.
	localGround []TileKind
}

func NewWorld(m Map) World {
	objects := objectsByChunk(m.Objects)
	chunks := make(map[core.ChunkKey]*core.ChunkContents, len(m.Chunks))
	for key, chunk := range m.Chunks {
		chunks[key] = chunk.contents(objects[key])
	}
	w := World{Terrain: NewTerrain(m), chunks: chunks, info: infoOf(m.Description), domes: domesOf(m.Objects), edge: []core.Cell{}}
	if !m.Wraps {
		w.localGround = groundCells(m)
		w.edge = w.findEdge()
	}
	return w
}

// Chunk is what players are sent of a chunk of the map: a chunk with no cells stored is all void.
// False for a chunk outside the map.
func (w World) Chunk(key core.ChunkKey) (*core.ChunkContents, bool) {
	if !w.grid.HasChunk(key) {
		return nil, false
	}
	if contents, ok := w.chunks[key]; ok {
		return contents, true
	}
	return NewChunk(key, TileVoid).contents(nil), true
}

// Info is what a player is told of the map on arriving.
func (w World) Info() core.MapInfo {
	return w.info
}

// DomeAt is the dome covering a cell; false when none does.
func (w World) DomeAt(c core.Cell) (core.Dome, bool) {
	dome, ok := w.domes[c]
	return dome, ok
}

// DomeLeadingTo is the dome leading to a map; false when none does.
func (w World) DomeLeadingTo(slug string) (core.Dome, bool) {
	for _, dome := range w.domes {
		if dome.LeadsTo == slug {
			return dome, true
		}
	}
	return core.Dome{}, false
}

// Edge is a city's edge: its walkable cells touching its wall (8 neighbors), row by row. One arrives there when
// entering, and walks there to leave (ADR 0012). Empty on a map without walls.
func (w World) Edge() []core.Cell {
	return w.edge
}

// IsOutside: in a city, a click on this cell means leaving: it's past the map's edge, void, or the wall. Never on a
// wrapping map, which has no outside.
func (w World) IsOutside(c core.Cell) bool {
	if w.grid.Wraps {
		return false
	}
	if !w.grid.Contains(c) {
		return true
	}
	kind := w.localGround[w.grid.Index(c)]
	return kind == TileVoid || kind == TileWall
}

func (w World) findEdge() []core.Cell {
	edge := []core.Cell{}
	for row := range w.grid.Height {
		for col := range w.grid.Width {
			c := core.Cell{Col: col, Row: row}
			if w.IsWalkable(c) && w.touchesWall(c) {
				edge = append(edge, c)
			}
		}
	}
	return edge
}

func (w World) touchesWall(c core.Cell) bool {
	for _, d := range stepDirections {
		if next, ok := w.grid.Normalize(core.Cell{Col: c.Col + d.Col, Row: c.Row + d.Row}); ok && w.localGround[w.grid.Index(next)] == TileWall {
			return true
		}
	}
	return false
}

// objectsByChunk: an object is sent with the chunk of its cell, a dome's whole footprint being in it.
func objectsByChunk(objects []Object) map[core.ChunkKey][]Object {
	byChunk := map[core.ChunkKey][]Object{}
	for _, object := range objects {
		key := core.ChunkOf(object.Cell())
		byChunk[key] = append(byChunk[key], object)
	}
	return byChunk
}

func infoOf(d Description) core.MapInfo {
	regions := make([]core.RegionInfo, len(d.Regions))
	for i, r := range d.Regions {
		regions[i] = core.RegionInfo{ID: r.ID, Name: r.Name, Biome: string(r.Biome)}
	}
	return core.MapInfo{ID: d.ID, Slug: d.Slug, Name: d.Name, Width: d.Width, Height: d.Height, Wraps: d.Wraps, Regions: regions, StandsOn: d.StandsOn}
}

func domesOf(objects []Object) map[core.Cell]core.Dome {
	domes := map[core.Cell]core.Dome{}
	for _, object := range objects {
		if object.Kind == ObjectDome {
			for _, c := range object.Footprint() {
				domes[c] = object.Dome()
			}
		}
	}
	return domes
}

// groundCells: each cell's ground, row by row, void in missing chunks.
func groundCells(m Map) []TileKind {
	grid := m.Grid()
	ground := make([]TileKind, grid.Width*grid.Height)
	for _, chunk := range m.Chunks {
		for i, kind := range chunk.Ground {
			if cell := chunk.cellAt(i); grid.Contains(cell) {
				ground[grid.Index(cell)] = kind
			}
		}
	}
	return ground
}

// Package domain holds the map rules: its chunks and regions, how a world is generated, where one can walk.
package domain

import (
	"fmt"

	core "donut/internal/core/domain"
	"donut/pkg/errd"
)

const (
	// MaxSize bounds a map's width and height, in cells.
	MaxSize = 4096
	// MinWrappingSize is the smallest side of a wrapping map: 5 chunks, so a chunk never shows up twice
	// in the 5 x 5 chunks a player holds.
	MinWrappingSize = 5 * core.ChunkSize
	// MaxRegions is how many regions a map may have: a cell holds its region's id in one byte, 0 for none.
	MaxRegions = 255
	// MinHeight and MaxHeight bound a cell's height, in levels (ADR 0007).
	MinHeight = -3
	MaxHeight = 3
)

// TileKind is what the ground of a cell is made of. Its value is its code in the stored chunks.
type TileKind uint8

const (
	// TileVoid is no ground at all: not drawn, not walkable.
	TileVoid TileKind = iota
	TileGrass
	// TileWater is a river or a lake: drawn, not walkable.
	TileWater
	// TileHole is a pit: drawn, not walkable.
	TileHole
	// TileBridge is a bridge's deck over the river: walkable.
	TileBridge
	// TileOcean is the sea (ADR 0008): drawn still, not walkable. Rivers and lakes are TileWater.
	TileOcean
)

var tileNames = [...]string{TileVoid: "void", TileGrass: "grass", TileWater: "water", TileHole: "hole", TileBridge: "bridge", TileOcean: "ocean"}

func (k TileKind) Known() bool {
	return int(k) < len(tileNames)
}

// Walkable: one may walk on this ground, when nothing stands on it.
func (k TileKind) Walkable() bool {
	return k == TileGrass || k == TileBridge
}

// String is the kind's name, as players are sent it.
func (k TileKind) String() string {
	if !k.Known() {
		return fmt.Sprintf("unknown(%d)", uint8(k))
	}
	return tileNames[k]
}

// ObjectKind is what stands on a cell.
type ObjectKind string

const ObjectRock ObjectKind = "rock"

func (k ObjectKind) Known() bool {
	return k == ObjectRock
}

// Object stands on one cell. Its ID is empty until saved.
type Object struct {
	ID   string     `db:"id"`
	Kind ObjectKind `db:"kind"`
	Col  int        `db:"col"`
	Row  int        `db:"row"`
}

func (o Object) Cell() core.Cell {
	return core.Cell{Col: o.Col, Row: o.Row}
}

// Summary is a map without its contents.
type Summary struct {
	ID     string `db:"id"`
	Slug   string `db:"slug"`
	Name   string `db:"name"`
	Width  int    `db:"width"`
	Height int    `db:"height"`
	Wraps  bool   `db:"wraps"`
}

func (s Summary) Grid() core.Grid {
	return core.Grid{Width: s.Width, Height: s.Height, Wraps: s.Wraps}
}

// Biome is how a region looks.
type Biome string

const BiomeMeadow Biome = "meadow"

func (b Biome) Known() bool {
	return b == BiomeMeadow
}

// Region is a named area of a map. Its cells hold its id, 1 to 255.
type Region struct {
	ID    int    `db:"id"`
	Name  string `db:"name"`
	Biome Biome  `db:"biome"`
}

// Description is a map without its cells: its size, its regions.
type Description struct {
	Summary
	Regions []Region
}

// Map is a whole map: its chunks, its regions and the objects on it. A chunk missing from Chunks is all void.
type Map struct {
	Description
	Chunks  map[core.ChunkKey]*Chunk
	Objects []Object
}

// NewMap is a map whose every cell has the same ground, no region, and nothing on it.
func NewMap(summary Summary, ground TileKind) Map {
	m := Map{Description: Description{Summary: summary, Regions: []Region{}}, Chunks: map[core.ChunkKey]*Chunk{}, Objects: []Object{}}
	grid := summary.Grid()
	for cy := range grid.ChunkRows() {
		for cx := range grid.ChunkColumns() {
			key := core.ChunkKey{CX: cx, CY: cy}
			m.Chunks[key] = NewChunk(key, ground)
		}
	}
	return m
}

// GroundAt is the ground of a cell within the map; void when its chunk is missing.
func (m Map) GroundAt(c core.Cell) TileKind {
	if chunk, ok := m.Chunks[core.ChunkOf(c)]; ok {
		return chunk.Ground[chunk.local(c)]
	}
	return TileVoid
}

// SetGround changes the ground of a cell within the map. Its chunk must exist.
func (m Map) SetGround(c core.Cell, kind TileKind) {
	chunk := m.Chunks[core.ChunkOf(c)]
	chunk.Ground[chunk.local(c)] = kind
}

// SetRegion puts a cell within the map in a region. Its chunk must exist.
func (m Map) SetRegion(c core.Cell, region uint8) {
	chunk := m.Chunks[core.ChunkOf(c)]
	chunk.Region[chunk.local(c)] = region
}

// HeightAt is the height of a cell within the map; 0 when its chunk is missing.
func (m Map) HeightAt(c core.Cell) int8 {
	if chunk, ok := m.Chunks[core.ChunkOf(c)]; ok {
		return chunk.Height[chunk.local(c)]
	}
	return 0
}

// SetHeight changes the height of a cell within the map. Its chunk must exist.
func (m Map) SetHeight(c core.Cell, height int8) {
	chunk := m.Chunks[core.ChunkOf(c)]
	chunk.Height[chunk.local(c)] = height
}

// RegionAt is the region id of a cell within the map, 0 for none or when its chunk is missing.
func (m Map) RegionAt(c core.Cell) uint8 {
	if chunk, ok := m.Chunks[core.ChunkOf(c)]; ok {
		return chunk.Region[chunk.local(c)]
	}
	return 0
}

// Validate refuses a map whose size, chunks, regions or objects don't fit together.
func (m Map) Validate() error {
	for _, check := range []func() error{m.validateSize, m.validateRegions, m.validateChunks, m.validateObjects} {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}

func (m Map) validateSize() error {
	for _, side := range []int{m.Width, m.Height} {
		if side < core.ChunkSize || side > MaxSize || side%core.ChunkSize != 0 {
			return invalid("size %dx%d: each side is a multiple of %d from %d to %d", m.Width, m.Height, core.ChunkSize, core.ChunkSize, MaxSize)
		}
		if m.Wraps && side < MinWrappingSize {
			return invalid("size %dx%d: a wrapping map is at least %d on each side", m.Width, m.Height, MinWrappingSize)
		}
	}
	return nil
}

func (m Map) validateRegions() error {
	if len(m.Regions) > MaxRegions {
		return invalid("%d regions, at most %d", len(m.Regions), MaxRegions)
	}
	seen := map[int]bool{}
	for _, region := range m.Regions {
		switch {
		case region.ID < 1 || region.ID > MaxRegions:
			return invalid("region id %d is outside 1 to %d", region.ID, MaxRegions)
		case seen[region.ID]:
			return invalid("two regions with id %d", region.ID)
		case region.Name == "":
			return invalid("region %d has no name", region.ID)
		case !region.Biome.Known():
			return invalid("region %d: unknown biome %q", region.ID, region.Biome)
		}
		seen[region.ID] = true
	}
	return nil
}

func (m Map) validateChunks() error {
	grid, regions := m.Grid(), m.regionIDs()
	for key, chunk := range m.Chunks {
		if chunk.Key != key || !grid.HasChunk(key) {
			return invalid("chunk %v is outside the map", key)
		}
		if err := chunk.validate(regions); err != nil {
			return err
		}
	}
	return nil
}

func (m Map) validateObjects() error {
	grid := m.Grid()
	taken := make(map[core.Cell]bool, len(m.Objects))
	for _, object := range m.Objects {
		cell := object.Cell()
		switch {
		case !object.Kind.Known():
			return invalid("unknown object kind %q", object.Kind)
		case !grid.Contains(cell):
			return invalid("object at %v is outside the map", cell)
		case m.GroundAt(cell) != TileGrass:
			return invalid("object at %v stands on %v, not grass", cell, m.GroundAt(cell))
		case taken[cell]:
			return invalid("two objects on %v", cell)
		}
		taken[cell] = true
	}
	return nil
}

// regionIDs are the ids a cell may hold: the regions', and 0 for none.
func (m Map) regionIDs() map[uint8]bool {
	ids := map[uint8]bool{0: true}
	for _, region := range m.Regions {
		ids[uint8(region.ID)] = true
	}
	return ids
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("invalid map: "+format+": %w", append(args, errd.ErrBadRequest)...)
}

package domain

import (
	core "donut/internal/core/domain"
)

// Chunk is 32 x 32 cells of a map. Each cell has a ground kind, a height and a region, row by row.
type Chunk struct {
	Key core.ChunkKey
	// Version goes up each time the chunk changes, from 1.
	Version int
	Ground  [core.CellsPerChunk]TileKind
	// Height is 0 everywhere for now: walking and drawing on heights come later (ADR 0003).
	Height [core.CellsPerChunk]int8
	// Region is the id of each cell's region, 0 for none.
	Region [core.CellsPerChunk]uint8
}

// NewChunk is a chunk at version 1 whose every cell has the same ground, a height of 0 and no region.
func NewChunk(key core.ChunkKey, ground TileKind) *Chunk {
	chunk := &Chunk{Key: key, Version: 1}
	for i := range chunk.Ground {
		chunk.Ground[i] = ground
	}
	return chunk
}

// local is a cell's position in the chunk's arrays. The cell must be in the chunk.
func (c *Chunk) local(cell core.Cell) int {
	origin := c.Key.Origin()
	return (cell.Row-origin.Row)*core.ChunkSize + cell.Col - origin.Col
}

// cellAt is the map cell at a position of the chunk's arrays.
func (c *Chunk) cellAt(i int) core.Cell {
	origin := c.Key.Origin()
	return core.Cell{Col: origin.Col + i%core.ChunkSize, Row: origin.Row + i/core.ChunkSize}
}

func (c *Chunk) validate(regions map[uint8]bool) error {
	if c.Version < 1 {
		return invalid("chunk %v: version %d, expected 1 or more", c.Key, c.Version)
	}
	for i := range c.Ground {
		switch {
		case !c.Ground[i].Known():
			return invalid("chunk %v, cell %v: unknown ground code %d", c.Key, c.cellAt(i), c.Ground[i])
		case !regions[c.Region[i]]:
			return invalid("chunk %v, cell %v: unknown region %d", c.Key, c.cellAt(i), c.Region[i])
		}
	}
	return nil
}

// contents is what players are sent of the chunk, with the objects standing on it.
func (c *Chunk) contents(objects []Object) *core.ChunkContents {
	out := &core.ChunkContents{
		Key:     c.Key,
		Version: c.Version,
		Kinds:   []string{},
		Ground:  make([]uint8, core.CellsPerChunk),
		Height:  append([]int8(nil), c.Height[:]...),
		Region:  append([]uint8(nil), c.Region[:]...),
		Objects: make([]core.PlacedObject, len(objects)),
	}
	palette := map[TileKind]uint8{}
	for i, kind := range c.Ground {
		index, ok := palette[kind]
		if !ok {
			index = uint8(len(out.Kinds))
			palette[kind] = index
			out.Kinds = append(out.Kinds, kind.String())
		}
		out.Ground[i] = index
	}
	for i, object := range objects {
		out.Objects[i] = core.PlacedObject{ID: object.ID, Kind: string(object.Kind), Cell: object.Cell()}
	}
	return out
}

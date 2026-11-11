package domain

import (
	"math"

	core "donut/internal/core/domain"
)

// HoleOptions are the pits dug in the ground (ADR 0007). Zero holes per chunk: none.
type HoleOptions struct {
	// PerChunk is how many holes a chunk gets, on average.
	PerChunk float64
	// Size is how many cells a hole has.
	Size Span
	// FromWater is how many steps from water of any sort, or a bridge, a hole stays, at least.
	FromWater int
}

func starterHoles() HoleOptions {
	return HoleOptions{PerChunk: 0.4, Size: Span{Min: 1, Max: 6}, FromWater: 3}
}

// digHoles digs blobs of cells into holes. A hole that can't start where it's drawn is skipped, and one that can't
// grow stays smaller: the map's holes may be fewer or smaller than asked.
func (l *land) digHoles(o HoleOptions, rng Rng) {
	if o.PerChunk <= 0 {
		return
	}
	count := int(math.Round(o.PerChunk * float64(l.grid.ChunkColumns()*l.grid.ChunkRows())))
	for range count {
		center := core.Cell{Col: int(rng.Float64() * float64(l.grid.Width)), Row: int(rng.Float64() * float64(l.grid.Height))}
		size := o.Size.pick(rng)
		for _, i := range l.holeBlob(center, size, o.FromWater, rng) {
			l.ground[i] = TileHole
		}
	}
}

// holeBlob grows a blob from its center, one straight neighbor of a cell of it at a time, up to `size` cells.
func (l *land) holeBlob(center core.Cell, size, fromWater int, rng Rng) []int {
	start, ok := l.index(center)
	if !ok {
		return nil
	}
	return l.growBlob(start, size, func(i int) bool { return l.canDig(i, fromWater) }, rng)
}

// canDig: the cell is grass, not kept, far enough from the water, and touches no other hole: holes don't merge.
func (l *land) canDig(i, fromWater int) bool {
	if !l.isGrass(i) || (l.toWater[i] >= 0 && int(l.toWater[i]) < fromWater) {
		return false
	}
	touches := false
	l.eachNeighbor(i, func(j int) { touches = touches || l.ground[j] == TileHole })
	return !touches
}

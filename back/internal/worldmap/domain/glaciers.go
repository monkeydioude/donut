package domain

import (
	"math"

	core "donut/internal/core/domain"
)

// GlacierOptions are the ice blobs at the inner ridge of the donut (ADR 0009): the row that faces the hole, the
// coldest. Zero PerChunk or HalfWidth: none.
type GlacierOptions struct {
	// HalfWidth is how many rows the cold band spans on each side of the ridge.
	HalfWidth int
	// PerChunk is how many glaciers a chunk of the band gets, on average. They are densest on the ridge.
	PerChunk float64
	// Size is how many cells a glacier has.
	Size Span
	// Clearance is how far from the spawn a glacier stays, at least.
	Clearance int
	Spawn     core.Cell
}

// starterGlaciers: about 4 glaciers per chunk of the 160 rows around the ridge, 48 cells from the spawn at least.
func starterGlaciers(spawn core.Cell) GlacierOptions {
	return GlacierOptions{HalfWidth: 80, PerChunk: 4, Size: Span{Min: 6, Max: 24}, Clearance: 48, Spawn: spawn}
}

// ridge is the row that faces the hole of the donut (ADR 0003).
func (l *land) ridge() int {
	return l.grid.Height / 2
}

// freezeRidge grows glaciers on the grass of the cold band, each from a random column and a row drawn as the ridge
// plus the difference of two draws: a triangle, densest on the ridge. A seed on the sea, water, a hole, a mountain
// or ice is skipped, and a glacier that can't grow stays smaller: the count is a ceiling.
func (l *land) freezeRidge(o GlacierOptions, rng Rng) {
	if o.PerChunk <= 0 || o.HalfWidth <= 0 {
		return
	}
	bandChunks := float64(l.grid.ChunkColumns()) * float64(2*o.HalfWidth) / float64(core.ChunkSize)
	for range int(math.Round(o.PerChunk * bandChunks)) {
		row := l.ridge() + int(math.Round((rng.Float64()-rng.Float64())*float64(o.HalfWidth)))
		start, ok := l.index(core.Cell{Col: int(rng.Float64() * float64(l.grid.Width)), Row: row})
		if !ok {
			continue
		}
		for _, i := range l.growBlob(start, o.Size.pick(rng), func(i int) bool { return l.canFreeze(i, o) }, rng) {
			l.ground[i] = TileGlacier
		}
	}
}

// canFreeze: the cell is grass, within the band, and away from the spawn.
func (l *land) canFreeze(i int, o GlacierOptions) bool {
	c := l.cellAt(i)
	if !l.isGrass(i) || abs(c.Row-l.ridge()) > o.HalfWidth {
		return false
	}
	return distance(l.grid, pointOf(c), pointOf(o.Spawn)) >= float64(o.Clearance)
}

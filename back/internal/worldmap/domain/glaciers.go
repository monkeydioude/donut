package domain

import (
	core "donut/internal/core/domain"
)

// GlacierOptions are the glacier ring at the inner ridge of the donut (ADR 0011): the rows that face the hole, the
// coldest, all the way round. Zero Edge: none.
type GlacierOptions struct {
	// Core is how many rows on each side of the ridge are all glacier; Edge how far its ragged edge reaches.
	Core, Edge int
	// Pass is how many columns on each side of the spawn's column stay open, on the whole ring's height.
	Pass  int
	Spawn core.Cell
	// RiseEvery is how many cells in from the ring's edge each level of ice takes.
	RiseEvery int
}

// starterGlaciers: the 40 rows on each side of the ridge, a ragged edge out to 64, a pass 96 columns wide at the
// spawn, and a level of ice every 4 cells in.
func starterGlaciers(spawn core.Cell) GlacierOptions {
	return GlacierOptions{Core: 40, Edge: 64, Pass: 48, Spawn: spawn, RiseEvery: 4}
}

// ridge is the row that faces the hole of the donut (ADR 0003).
func (l *land) ridge() int {
	return l.grid.Height / 2
}

// freezeRidge turns the ring into glacier, over land and sea: grass, water, the ocean and bridges freeze; holes stay,
// as crevasses, and so do mountains, rock standing in the ice. Then the ice rises: each glacier cell's height is its
// height before + 1 + ⌊(d − 1) / RiseEvery⌋, d its distance to the closest cell that isn't glacier, a hole or a
// mountain, capped at MaxHeight.
func (l *land) freezeRidge(o GlacierOptions, rng Rng) {
	if o.Edge <= 0 {
		return
	}
	ragged := fractalNoise(l.grid, edgeWavelengths, rng)
	for i := range l.ground {
		if l.inRing(i, o, ragged[i]) && canFreeze(l.ground[i]) {
			l.ground[i] = TileGlacier
		}
	}
	outside := func(i int) bool {
		kind := l.ground[i]
		return kind != TileGlacier && kind != TileHole && kind != TileMountain
	}
	l.raiseByDistance(TileGlacier, outside, func(steps int) int { return 1 + (steps-1)/o.RiseEvery })
}

// inRing: the cell is within the core, or within the edge where the noise (−1 to 1) lets it, and out of the pass. A
// cell r rows from the ridge, past the core, freezes when (r − Core) / (Edge − Core) < (noise + 1) / 2.
func (l *land) inRing(i int, o GlacierOptions, noise float64) bool {
	c := l.cellAt(i)
	if axisDistance(float64(c.Col), float64(o.Spawn.Col), l.grid.Width, l.grid.Wraps) <= float64(o.Pass) {
		return false
	}
	r := abs(c.Row - l.ridge())
	if r <= o.Core {
		return true
	}
	return r <= o.Edge && float64(r-o.Core)/float64(o.Edge-o.Core) < (noise+1)/2
}

// canFreeze: grass, water of any sort and bridges become glacier.
func canFreeze(kind TileKind) bool {
	return kind == TileGrass || isWater(kind) || kind == TileBridge
}

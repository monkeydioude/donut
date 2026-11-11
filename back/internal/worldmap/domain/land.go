package domain

import (
	"math"

	core "donut/internal/core/domain"
)

// bankReach is how far from the water, in steps, the water pulls the ground toward its level (ADR 0007, ADR 0008).
const bankReach = 4

// land is the ground being generated: each cell's kind and height, row by row, written into the chunks at the end.
type land struct {
	grid   core.Grid
	ground []TileKind
	height []int8
	// kept are cells left walkable: no rock or hole goes there, like a bridge's ends.
	kept []bool
	// fixed are cells whose height never moves again, like a bridge's deck and ends.
	fixed []bool
	// landmass is the landmass each cell is part of, −1 for the ocean; 0 everywhere on a map with no ocean.
	landmass []int16
	// ocean: the map has an ocean, so toOcean and nearestOcean are set.
	ocean bool
	// toOcean is how many steps each cell is from the ocean, diagonals counting as 1, and nearestOcean the ocean cell
	// it's closest to. Both −1 on a map with no ocean.
	toOcean      []int32
	nearestOcean []int32
	// toWater is how many steps each cell is from the closest water (ocean, lake or river), up to bankReach; −1 when
	// farther.
	toWater []int8
}

func newLand(grid core.Grid) *land {
	cells := grid.Width * grid.Height
	l := &land{
		grid: grid, ground: make([]TileKind, cells), height: make([]int8, cells), kept: make([]bool, cells), fixed: make([]bool, cells),
		landmass: make([]int16, cells), toOcean: make([]int32, cells), nearestOcean: make([]int32, cells),
		toWater: make([]int8, cells),
	}
	for i := range cells {
		l.ground[i] = TileGrass
		l.toOcean[i], l.nearestOcean[i], l.toWater[i] = -1, -1, -1
	}
	return l
}

// isWater: the kind is water of any sort, which never moves and pulls the ground around it.
func isWater(kind TileKind) bool {
	return kind == TileWater || kind == TileOcean
}

func (l *land) cellAt(i int) core.Cell {
	return core.Cell{Col: i % l.grid.Width, Row: i / l.grid.Width}
}

// index is the position of a cell in the land's arrays, across the seams on a wrapping map. False outside a bounded
// map.
func (l *land) index(c core.Cell) (int, bool) {
	c, ok := l.grid.Normalize(c)
	if !ok {
		return 0, false
	}
	return l.grid.Index(c), true
}

// eachNeighbor calls fn with each of the 8 cells around cell i, across the seams on a wrapping map.
func (l *land) eachNeighbor(i int, fn func(j int)) {
	c := l.cellAt(i)
	for _, d := range stepDirections {
		if j, ok := l.index(core.Cell{Col: c.Col + d.Col, Row: c.Row + d.Row}); ok {
			fn(j)
		}
	}
}

// eachWithin calls fn with each cell at most `reach` steps from cell i, diagonals counting as 1, itself left out.
func (l *land) eachWithin(i, reach int, fn func(j, steps int)) {
	c := l.cellAt(i)
	for dc := -reach; dc <= reach; dc++ {
		for dr := -reach; dr <= reach; dr++ {
			if j, ok := l.index(core.Cell{Col: c.Col + dc, Row: c.Row + dr}); ok && (dc != 0 || dr != 0) {
				fn(j, max(abs(dc), abs(dr)))
			}
		}
	}
}

// isGrass: the cell is plain grass, free to be dug or flooded.
func (l *land) isGrass(i int) bool {
	return l.ground[i] == TileGrass && !l.kept[i]
}

// growBlob grows a blob from its start, one straight neighbor of a cell of it at a time, up to `size` cells, on the
// cells `can` allows. Nothing when the start isn't allowed; a blob that can't grow stays smaller.
func (l *land) growBlob(start, size int, can func(i int) bool, rng Rng) []int {
	if !can(start) {
		return nil
	}
	blob := []int{start}
	in := map[int]bool{start: true}
	for tries := 0; len(blob) < size && tries < size*8; tries++ {
		from := l.cellAt(blob[int(rng.Float64()*float64(len(blob)))])
		d := stepDirections[int(rng.Float64()*4)]
		next, ok := l.index(core.Cell{Col: from.Col + d.Col, Row: from.Row + d.Row})
		if ok && !in[next] && can(next) {
			blob = append(blob, next)
			in[next] = true
		}
	}
	return blob
}

// measureToOcean counts each cell's steps to the ocean, and remembers the ocean cell it reached first.
func (l *land) measureToOcean() {
	var ring []int
	for i, kind := range l.ground {
		if kind == TileOcean {
			l.toOcean[i], l.nearestOcean[i] = 0, int32(i)
			ring = append(ring, i)
		}
	}
	l.ocean = len(ring) > 0
	for steps := int32(1); len(ring) > 0; steps++ {
		var next []int
		for _, i := range ring {
			l.eachNeighbor(i, func(j int) {
				if l.toOcean[j] < 0 {
					l.toOcean[j], l.nearestOcean[j] = steps, l.nearestOcean[i]
					next = append(next, j)
				}
			})
		}
		ring = next
	}
}

// measureStepsTo counts each cell's steps to the closest cell `from` allows, diagonals counting as 1, up to `reach`,
// into `steps`: −1 when farther.
func (l *land) measureStepsTo(from func(i int) bool, reach int8, steps []int8) {
	var ring []int
	for i := range steps {
		steps[i] = -1
		if from(i) {
			steps[i] = 0
			ring = append(ring, i)
		}
	}
	for k := int8(1); k <= reach && len(ring) > 0; k++ {
		var next []int
		for _, i := range ring {
			l.eachNeighbor(i, func(j int) {
				if steps[j] < 0 {
					steps[j] = k
					next = append(next, j)
				}
			})
		}
		ring = next
	}
}

// measureToWater counts each cell's steps to the closest water of any sort, up to bankReach.
func (l *land) measureToWater() {
	l.measureStepsTo(func(i int) bool { return isWater(l.ground[i]) }, bankReach, l.toWater)
}

// settleBanks keeps each cell within reach of water between L − (k − 1) and L + k for every water level L around
// it, k its steps to the closest water at that level, so banks are at L or L + 1 and the ground steps up and down
// between waters of different levels. Then it smooths the cliffs away. A cell that can't satisfy every water is left
// as it is.
func (l *land) settleBanks() {
	l.measureToWater()
	lowest, highest := make([]int8, len(l.ground)), make([]int8, len(l.ground))
	for i := range lowest {
		lowest[i], highest[i] = MinHeight, MaxGroundHeight
	}
	steps := make([]int8, len(l.ground))
	for level := int8(MinHeight); level <= MaxGroundHeight; level++ {
		l.measureStepsTo(func(i int) bool { return isWater(l.ground[i]) && l.height[i] == level }, bankReach, steps)
		for i, k := range steps {
			if k > 0 {
				lowest[i], highest[i] = max(lowest[i], level-k+1), min(highest[i], level+k)
			}
		}
	}
	for i := range l.height {
		if !isWater(l.ground[i]) && lowest[i] <= highest[i] {
			l.height[i] = min(max(l.height[i], lowest[i]), highest[i])
		}
	}
	l.smoothCliffs()
}

// smoothCliffs moves cells one level toward 0, pass after pass, until no two neighbors are more than MaxClimb
// apart. Of two neighbors too far apart, the one farther from 0 moves, both on a tie. Water and fixed cells never
// move.
func (l *land) smoothCliffs() {
	for {
		var moving []int
		for i := range l.height {
			if !isWater(l.ground[i]) && !l.fixed[i] && l.tooSteep(i) {
				moving = append(moving, i)
			}
		}
		if len(moving) == 0 {
			return
		}
		for _, i := range moving {
			l.height[i] -= sign(l.height[i])
		}
	}
}

// tooSteep: a neighbor of cell i is more than MaxClimb levels from it, and no farther from 0.
func (l *land) tooSteep(i int) bool {
	h := int(l.height[i])
	steep := false
	l.eachNeighbor(i, func(j int) {
		other := int(l.height[j])
		steep = steep || (abs(h-other) > MaxClimb && abs(h) >= abs(other))
	})
	return steep
}

func (l *land) writeInto(m Map) {
	for i := range l.ground {
		cell := l.cellAt(i)
		m.SetGround(cell, l.ground[i])
		m.SetHeight(cell, l.height[i])
	}
}

// point is a place on the map, between cells.
type point struct{ col, row float64 }

func pointOf(c core.Cell) point {
	return point{col: float64(c.Col), row: float64(c.Row)}
}

// wrapPoint brings a point back onto a wrapping map.
func wrapPoint(grid core.Grid, p point) point {
	if !grid.Wraps {
		return p
	}
	w, h := float64(grid.Width), float64(grid.Height)
	return point{col: math.Mod(math.Mod(p.col, w)+w, w), row: math.Mod(math.Mod(p.row, h)+h, h)}
}

// distance between two points, the short way round on a wrapping map.
func distance(grid core.Grid, a, b point) float64 {
	return math.Hypot(axisDistance(a.col, b.col, grid.Width, grid.Wraps), axisDistance(a.row, b.row, grid.Height, grid.Wraps))
}

// axisDistance is how far apart a and b are on an axis of `size`, the short way round when it wraps.
func axisDistance(a, b float64, size int, wraps bool) float64 {
	d := math.Abs(a - b)
	if wraps && d > float64(size)/2 {
		d = float64(size) - d
	}
	return d
}

func square(x float64) float64 {
	return x * x
}

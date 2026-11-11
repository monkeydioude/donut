package domain

import (
	core "donut/internal/core/domain"
)

// MountainOptions are the massifs around the summits (ADR 0011). Zero Reach: none.
type MountainOptions struct {
	// Level is the height a summit's cells have, at least.
	Level int8
	// MinSummit is how many cells a summit needs to seed a massif.
	MinSummit int
	// Reach is how far a massif grows from its summit, in steps, at most: its edge is ragged, from half of it.
	Reach int
	// FootLevel is the lowest ground a massif grows over.
	FootLevel int8
	// Clearance is how far from the spawn a massif stays, at least.
	Clearance int
	Spawn     core.Cell
}

// starterMountains: massifs around every summit of 4 cells or more at level 3, up to 12 cells out and down to level
// 1, 24 cells from the spawn at least.
func starterMountains(spawn core.Cell) MountainOptions {
	return MountainOptions{Level: MaxGroundHeight, MinSummit: 4, Reach: 12, FootLevel: 1, Clearance: 24, Spawn: spawn}
}

// edgeWavelengths shape the noise that makes the edges of massifs and of the glacier ring ragged.
var edgeWavelengths = []int{32, 16, 8}

// raiseMountains grows a massif around each summit and raises it: each mountain cell's height is its ground's plus
// its distance to the massif's edge, capped at MaxHeight.
func (l *land) raiseMountains(o MountainOptions, rng Rng) {
	if o.Reach <= 0 {
		return
	}
	ragged := fractalNoise(l.grid, edgeWavelengths, rng)
	for _, i := range l.growMassifs(l.summits(o), o, ragged) {
		l.ground[i] = TileMountain
	}
	l.raiseByDistance(TileMountain, func(i int) bool { return l.ground[i] != TileMountain }, func(steps int) int { return steps })
}

// summits are the connected groups of grass cells at the level or above, 8 neighbors, of at least MinSummit
// cells, in row order.
func (l *land) summits(o MountainOptions) [][]int {
	isSummit := func(i int) bool { return l.isGrass(i) && l.height[i] >= o.Level }
	seen := make([]bool, len(l.ground))
	var summits [][]int
	for i := range l.ground {
		if seen[i] || !isSummit(i) {
			continue
		}
		if summit := l.connected(i, seen, isSummit); len(summit) >= o.MinSummit {
			summits = append(summits, summit)
		}
	}
	return summits
}

// connected floods the cells `in` allows from a start, 8 neighbors, marking them seen.
func (l *land) connected(start int, seen []bool, in func(i int) bool) []int {
	seen[start] = true
	cells := []int{start}
	for at := 0; at < len(cells); at++ {
		l.eachNeighbor(cells[at], func(j int) {
			if !seen[j] && in(j) {
				seen[j] = true
				cells = append(cells, j)
			}
		})
	}
	return cells
}

// growMassifs floods out from the summits, 8 neighbors, over grass at the foot level or above and away from the
// spawn. A cell k cells away from the summit cell it was reached from, as the crow flies, joins when
// k ≤ Reach × (0.5 + 0.5 × noise), the noise from −1 to 1, so the edges are round and ragged. A massif only grows from
// its own cells, so it stays in one piece.
func (l *land) growMassifs(summits [][]int, o MountainOptions, ragged []float64) []int {
	can := func(i int) bool {
		return l.isGrass(i) && l.height[i] >= o.FootLevel && distance(l.grid, pointOf(l.cellAt(i)), pointOf(o.Spawn)) >= float64(o.Clearance)
	}
	from := make([]int, len(l.ground))
	for i := range from {
		from[i] = -1
	}
	var massif []int
	for _, summit := range summits {
		for _, i := range summit {
			if can(i) && from[i] < 0 {
				from[i] = i
				massif = append(massif, i)
			}
		}
	}
	for at := 0; at < len(massif); at++ {
		source := from[massif[at]]
		l.eachNeighbor(massif[at], func(j int) {
			away := distance(l.grid, pointOf(l.cellAt(j)), pointOf(l.cellAt(source)))
			if from[j] < 0 && can(j) && away <= float64(o.Reach)*(0.75+0.25*ragged[j]) {
				from[j] = source
				massif = append(massif, j)
			}
		})
	}
	return massif
}

// raiseByDistance raises each cell of a kind by rise(steps), steps being its distance (8 neighbors) to the closest
// cell `edge` allows, capped at MaxHeight. A cell farther than the reach counted gets that reach's rise.
func (l *land) raiseByDistance(kind TileKind, edge func(i int) bool, rise func(steps int) int) {
	const reach = 4 * (MaxHeight - MinHeight + 1)
	steps := make([]int8, len(l.ground))
	l.measureStepsTo(edge, reach, steps)
	for i, k := range steps {
		if l.ground[i] != kind {
			continue
		}
		if k < 0 {
			k = reach
		}
		l.height[i] = int8(min(MaxHeight, int(l.height[i])+rise(int(k))))
	}
}

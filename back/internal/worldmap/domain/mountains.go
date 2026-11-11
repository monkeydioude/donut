package domain

import (
	"math"
)

// MountainOptions are the packs of bare rock on the summits (ADR 0009). Zero Cover: none.
type MountainOptions struct {
	// Level is the height a summit's cells have, at least.
	Level int8
	// MinSummit is how many cells a summit needs to get a pack.
	MinSummit int
	// Cover is the share of a summit's cells the pack takes, about.
	Cover float64
	// Size is how many cells each blob of the pack has.
	Size Span
}

// starterMountains: packs over about 60% of every summit of 4 cells or more at level 3.
func starterMountains() MountainOptions {
	return MountainOptions{Level: MaxHeight, MinSummit: 4, Cover: 0.6, Size: Span{Min: 3, Max: 10}}
}

// packTriesPerCell bounds how many blobs a summit is tried for, per cell of it.
const packTriesPerCell = 4

// raiseMountains puts a pack of mountains on each summit: the connected groups of grass cells at the level or
// above. Blobs are grown on the summit's grass, from random cells of it, until the cover is reached. A cell keeps
// its height: the pack is the hilltop itself.
func (l *land) raiseMountains(o MountainOptions, rng Rng) {
	if o.Cover <= 0 {
		return
	}
	for _, summit := range l.summits(o) {
		l.raisePack(summit, o, rng)
	}
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

// raisePack grows blobs of mountain on a summit until about the cover of it is taken: each blob is cut to what's
// left to take, and the pack stops when less than a blob's smallest size is left. A blob that can't reach that size
// is dropped, so no mountain cell stands alone; a summit whose blobs can't grow stays under the cover.
func (l *land) raisePack(summit []int, o MountainOptions, rng Rng) {
	in := cellSet(summit)
	target := int(math.Round(o.Cover * float64(len(summit))))
	placed := 0
	for tries := 0; placed+o.Size.Min <= target && tries < packTriesPerCell*len(summit); tries++ {
		start := summit[int(rng.Float64()*float64(len(summit)))]
		blob := l.growBlob(start, min(o.Size.pick(rng), target-placed), func(i int) bool { return in[i] && l.isGrass(i) }, rng)
		if len(blob) < o.Size.Min {
			continue
		}
		for _, i := range blob {
			l.ground[i] = TileMountain
		}
		placed += len(blob)
	}
}

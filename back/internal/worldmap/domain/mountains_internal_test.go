package domain

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
)

// hillLand is a wrapping 64 x 64 land of grass at 0 with a hill around (32, 32): level 3 within 2.5 cells, 2 within
// 6, 1 within 14. A kept cell at (32, 40), on the slope.
func hillLand() *land {
	l := newLand(core.Grid{Width: 64, Height: 64, Wraps: true})
	for i := range l.ground {
		r := distance(l.grid, pointOf(l.cellAt(i)), pointOf(core.Cell{Col: 32, Row: 32}))
		switch {
		case r <= 2.5:
			l.height[i] = 3
		case r <= 6:
			l.height[i] = 2
		case r <= 14:
			l.height[i] = 1
		}
	}
	keep, _ := l.index(core.Cell{Col: 32, Row: 40})
	l.kept[keep] = true
	return l
}

func hillMassif(spawn core.Cell) MountainOptions {
	return MountainOptions{Level: 3, MinSummit: 4, Reach: 12, FootLevel: 1, Clearance: 24, Spawn: spawn}
}

// ADR 0011 test 3
func TestRaiseMountains(t *testing.T) {
	before := hillLand()
	l := hillLand()
	l.raiseMountains(hillMassif(core.Cell{Col: 0, Row: 0}), rand.New(rand.NewPCG(1, 1)))
	summit := pointOf(core.Cell{Col: 32, Row: 32})

	t.Run("grows over the hill's grass at level 1 or above, at most 12 cells from the summit", func(t *testing.T) {
		count := 0
		for i, kind := range l.ground {
			if kind != TileMountain {
				continue
			}
			count++
			require.GreaterOrEqual(t, before.height[i], int8(1), "the mountain at %v", l.cellAt(i))
			require.LessOrEqual(t, distance(l.grid, pointOf(l.cellAt(i)), summit), 12+2.5, "the mountain at %v", l.cellAt(i))
		}
		assert.Greater(t, count, 100, "a massif")
	})

	t.Run("a kept cell stays grass", func(t *testing.T) {
		i, _ := l.index(core.Cell{Col: 32, Row: 40})
		assert.Equal(t, TileGrass, l.ground[i])
	})

	t.Run("each height is the ground's plus the distance to the massif's edge, capped at +6", func(t *testing.T) {
		edge := stepsToEdge(l, func(i int) bool { return l.ground[i] != TileMountain })
		for i, kind := range l.ground {
			if kind == TileMountain {
				require.Equal(t, int8(min(MaxHeight, int(before.height[i])+edge[i])), l.height[i], "the mountain at %v", l.cellAt(i))
			} else {
				require.Equal(t, before.height[i], l.height[i], "the grass at %v", l.cellAt(i))
			}
		}
	})

	t.Run("none within 24 cells of the spawn", func(t *testing.T) {
		near := hillLand()
		near.raiseMountains(hillMassif(core.Cell{Col: 32, Row: 32}), rand.New(rand.NewPCG(1, 1)))
		for i, kind := range near.ground {
			if kind == TileMountain {
				require.GreaterOrEqual(t, distance(near.grid, pointOf(near.cellAt(i)), summit), 24.0, "the mountain at %v", near.cellAt(i))
			}
		}
	})
}

// stepsToEdge is each cell's distance, 8 neighbors, to the closest cell `edge` allows, worked out apart from the land's
// own measures.
func stepsToEdge(l *land, edge func(i int) bool) []int {
	steps := make([]int, len(l.ground))
	var ring []int
	for i := range steps {
		steps[i] = math.MaxInt
		if edge(i) {
			steps[i] = 0
			ring = append(ring, i)
		}
	}
	for k := 1; len(ring) > 0; k++ {
		var next []int
		for _, i := range ring {
			l.eachNeighbor(i, func(j int) {
				if steps[j] == math.MaxInt {
					steps[j] = k
					next = append(next, j)
				}
			})
		}
		ring = next
	}
	return steps
}

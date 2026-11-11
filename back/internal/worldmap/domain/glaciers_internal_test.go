package domain

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
)

// ringLand is a wrapping 64 x 128 land of grass at 0, its ridge on row 64, with a strip of ocean (columns 40 to 47)
// at −1, a lake, a river down column 30 with a bridge on row 70, a hole and a mountain at 2 inside the core.
func ringLand() *land {
	l := newLand(core.Grid{Width: 64, Height: 128, Wraps: true})
	set := func(col, row int, kind TileKind, height int8) {
		i, _ := l.index(core.Cell{Col: col, Row: row})
		l.ground[i], l.height[i] = kind, height
	}
	for row := range 128 {
		for col := 40; col < 48; col++ {
			set(col, row, TileOcean, -1)
		}
	}
	for col := 20; col < 23; col++ {
		for row := 60; row < 63; row++ {
			set(col, row, TileWater, 0)
		}
	}
	for row := 50; row < 79; row++ {
		set(30, row, TileWater, 0)
	}
	set(30, 70, TileBridge, 1)
	for _, col := range []int{29, 31} {
		i, _ := l.index(core.Cell{Col: col, Row: 70})
		l.kept[i] = true
	}
	set(24, 66, TileHole, 0)
	set(26, 60, TileMountain, 2)
	return l
}

// a ring 10 rows each side, its edge out to 20, a pass 6 columns each side of column 2: across the seam.
var testRing = GlacierOptions{Core: 10, Edge: 20, Pass: 6, Spawn: core.Cell{Col: 2, Row: 64}, RiseEvery: 4}

// ADR 0011 test 4
func TestFreezeRidge(t *testing.T) {
	before := ringLand()
	l := ringLand()
	l.freezeRidge(testRing, rand.New(rand.NewPCG(2, 2)))
	inPass := func(c core.Cell) bool { return axisDistance(float64(c.Col), 2, 64, true) <= 6 }
	at := func(col, row int) TileKind {
		i, _ := l.index(core.Cell{Col: col, Row: row})
		return l.ground[i]
	}

	t.Run("the core is glacier, holes and mountains outside the pass, and nothing freezes past its edge", func(t *testing.T) {
		for i, kind := range l.ground {
			c := l.cellAt(i)
			r := abs(c.Row - 64)
			if kind == TileGlacier {
				require.LessOrEqual(t, r, 20, "the glacier at %v", c)
				require.False(t, inPass(c), "the glacier at %v is in the pass", c)
			} else if r <= 10 && !inPass(c) {
				require.True(t, kind == TileHole || kind == TileMountain, "%v at %v", kind, c)
			}
		}
	})

	t.Run("the sea, the lake, the river and the bridge freeze; the hole and the mountain stay", func(t *testing.T) {
		for _, c := range [][2]int{{44, 64}, {21, 61}, {30, 58}, {30, 70}} {
			assert.Equal(t, TileGlacier, at(c[0], c[1]), "(%d, %d)", c[0], c[1])
		}
		assert.Equal(t, TileHole, at(24, 66))
		assert.Equal(t, TileMountain, at(26, 60))
	})

	t.Run("the pass stays open, across the seam too", func(t *testing.T) {
		for _, col := range []int{60, 63, 0, 2, 8} {
			assert.Equal(t, TileGrass, at(col, 64), "column %d", col)
		}
		assert.Equal(t, TileGlacier, at(9, 64))
	})

	t.Run("each glacier is its height before + 1 + (d − 1) / 4, d its distance to the ring's edge, capped at +6", func(t *testing.T) {
		edge := stepsToEdge(l, func(i int) bool {
			return l.ground[i] != TileGlacier && l.ground[i] != TileHole && l.ground[i] != TileMountain
		})
		seaEdge := 0
		for i, kind := range l.ground {
			if kind != TileGlacier {
				require.Equal(t, before.height[i], l.height[i], "%v at %v", kind, l.cellAt(i))
				continue
			}
			require.Equal(t, int8(min(MaxHeight, int(before.height[i])+1+(edge[i]-1)/4)), l.height[i], "the glacier at %v", l.cellAt(i))
			if before.ground[i] == TileOcean && edge[i] == 1 {
				require.Equal(t, int8(0), l.height[i], "the sea's edge at %v", l.cellAt(i))
				seaEdge++
			}
		}
		assert.Positive(t, seaEdge, "the frozen sea has an edge")
	})
}

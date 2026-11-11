package domain_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
	"donut/internal/worldmap/domain"
)

func cell(col, row int) core.Cell { return core.Cell{Col: col, Row: row} }

// ADR 0002 tests 1 and 2
func TestPathOnAnOpenMap(t *testing.T) {
	terrain := domain.NewTerrain(mapWithRocks(6, 6))

	assert.Equal(t, []core.Cell{cell(1, 0), cell(2, 0), cell(3, 0)}, terrain.Path(cell(0, 0), cell(3, 0)), "a straight line")
	assert.Equal(t, []core.Cell{cell(1, 1), cell(2, 2)}, terrain.Path(cell(0, 0), cell(2, 2)), "diagonal when shorter")
}

// ADR 0002 test 3
func TestPathGoesAroundRocks(t *testing.T) {
	wall := []core.Cell{cell(2, 0), cell(2, 1), cell(2, 2)}
	terrain := domain.NewTerrain(mapWithRocks(5, 5, wall...))

	path := terrain.Path(cell(0, 0), cell(4, 0))
	require.NotEmpty(t, path)
	assert.Equal(t, cell(4, 0), path[len(path)-1])
	for _, step := range path {
		assert.NotContains(t, wall, step)
	}
}

// ADR 0002 test 4
func TestPathDoesNotCutCorners(t *testing.T) {
	terrain := domain.NewTerrain(mapWithRocks(3, 3, cell(1, 0)))
	assert.Equal(t, []core.Cell{cell(0, 1), cell(1, 1)}, terrain.Path(cell(0, 0), cell(1, 1)))
}

// ADR 0002 test 5
func TestNoPath(t *testing.T) {
	walledOff := []core.Cell{cell(3, 0), cell(3, 1), cell(4, 1)}
	terrain := domain.NewTerrain(mapWithRocks(5, 5, append(walledOff, cell(1, 1))...))

	assert.Nil(t, terrain.Path(cell(0, 0), cell(1, 1)), "the target is a rock")
	assert.Nil(t, terrain.Path(cell(0, 0), cell(5, 0)), "the target is outside the map")
	assert.Nil(t, terrain.Path(cell(0, 0), cell(4, 0)), "the target is walled off")
	assert.Nil(t, terrain.Path(cell(0, 0), cell(0, 0)), "the target is the start")
}

// Security review: a search is bounded, so a request stays cheap on big maps.
func TestPathSearchIsBounded(t *testing.T) {
	terrain := domain.NewTerrain(mapWithRocks(8, 8))
	assert.Len(t, terrain.Path(cell(0, 0), cell(7, 0)), 7)
	assert.Nil(t, terrain.WithSearchLimit(3).Path(cell(0, 0), cell(7, 0)), "past the limit, there is no path")
}

// Security review: an unreachable target is refused without searching.
func TestUnreachableTargetsAreRefusedAtOnce(t *testing.T) {
	walls := []core.Cell{cell(254, 255), cell(254, 254), cell(255, 254)}
	terrain := domain.NewTerrain(mapWithRocks(256, 256, walls...)).WithSearchLimit(1 << 30)
	start := time.Now()
	assert.Nil(t, terrain.Path(cell(0, 0), cell(255, 255)))
	assert.Less(t, time.Since(start), 5*time.Millisecond, "the areas say it's walled off: no search")
}

func wrappingMap(width, height int, rocks ...core.Cell) domain.Map {
	m := mapWithRocks(width, height, rocks...)
	m.Wraps = true
	return m
}

// ADR 0003 test 7
func TestPathsAcrossTheSeams(t *testing.T) {
	t.Run("the cells across the seam are walkable neighbors", func(t *testing.T) {
		terrain := domain.NewTerrain(wrappingMap(64, 64))
		assert.True(t, terrain.Grid().AreNeighbors(cell(63, 5), cell(0, 5)))
		assert.Equal(t, []core.Cell{cell(0, 5), cell(1, 5)}, terrain.Path(cell(63, 5), cell(1, 5)), "2 steps, not 62")
		assert.Equal(t, []core.Cell{cell(0, 0)}, terrain.Path(cell(63, 63), cell(0, 0)), "one diagonal step across both seams")
		assert.False(t, terrain.IsWalkable(cell(64, 5)), "cells are told within the map")
	})

	t.Run("no corner cutting across the seam", func(t *testing.T) {
		terrain := domain.NewTerrain(wrappingMap(64, 64, cell(0, 5)))
		assert.Equal(t, []core.Cell{cell(63, 4), cell(0, 4)}, terrain.Path(cell(63, 5), cell(0, 4)))
	})

	t.Run("a wall down the middle doesn't cut the map in two", func(t *testing.T) {
		var wall []core.Cell
		for row := range 64 {
			wall = append(wall, cell(32, row))
		}
		path := domain.NewTerrain(wrappingMap(64, 64, wall...)).Path(cell(31, 5), cell(33, 5))
		assert.Len(t, path, 62, "around the donut, across the seam")
		assert.Nil(t, domain.NewTerrain(mapWithRocks(64, 64, wall...)).Path(cell(31, 5), cell(33, 5)), "a bounded map is cut")
	})

	t.Run("void isn't walked on", func(t *testing.T) {
		m := wrappingMap(64, 64)
		m.SetGround(cell(0, 5), domain.TileVoid)
		path := domain.NewTerrain(m).Path(cell(63, 5), cell(1, 5))
		assert.Equal(t, []core.Cell{cell(63, 6), cell(0, 6), cell(1, 6), cell(1, 5)}, path, "around it, cutting no corner")
	})
}

// ADR 0003 test 8, ADR 0002 test 6, and ADR 0007 test 6
func TestPathsOnRandomMaps(t *testing.T) {
	rng := seeded(3)
	for i := range 600 {
		m := domain.Generate(domain.GenerateOptions{Width: 8, Height: 8, Wraps: i%2 == 1, RockChance: 0.2}, rng)
		if i%3 > 0 {
			eachCell(m, func(c core.Cell) { m.SetHeight(c, int8(rng.IntN(3))) })
		}
		terrain := domain.NewTerrain(m)
		from, to := randomCell(rng.IntN(64)), randomCell(rng.IntN(64))
		if !terrain.IsWalkable(from) {
			continue
		}
		path := terrain.Path(from, to)
		best, reachable := shortestCost(terrain, from, to)
		if from == to || !reachable {
			assert.Nil(t, path, "map %d: %v to %v", i, from, to)
			continue
		}
		require.NotEmpty(t, path, "map %d: %v to %v is reachable", i, from, to)
		assert.InDelta(t, best, checkSteps(t, terrain, from, path), 1e-9, "map %d: %v to %v is shortest", i, from, to)
	}
}

func randomCell(n int) core.Cell { return cell(n%8, n/8) }

// checkSteps checks each step is a walkable neighbor at most 1 level up or down that cuts no corner, and gives the
// path's cost.
func checkSteps(t *testing.T, terrain domain.Terrain, from core.Cell, path []core.Cell) float64 {
	t.Helper()
	grid := terrain.Grid()
	cost := 0.0
	for _, step := range path {
		require.True(t, grid.AreNeighbors(from, step), "%v to %v is one step", from, step)
		require.True(t, canStep(terrain, from, step), "%v to %v is walkable, 1 level at most", from, step)
		if step.Col != from.Col && step.Row != from.Row {
			d := grid.Offset(from, step)
			require.True(t, goesRound(terrain, from, step, cell(d.Col, 0)) && goesRound(terrain, from, step, cell(0, d.Row)),
				"%v to %v cuts no corner", from, step)
		}
		cost += domain.StepCost(from, step)
		from = step
	}
	return cost
}

// canStep: b is walkable and at most 1 level from a.
func canStep(terrain domain.Terrain, a, b core.Cell) bool {
	return terrain.IsWalkable(b) && absInt(int(terrain.HeightAt(a))-int(terrain.HeightAt(b))) <= 1
}

// goesRound: from a, the cell `d` away and then b are both steps, across the seams on a wrapping map.
func goesRound(terrain domain.Terrain, a, b, d core.Cell) bool {
	beside, ok := terrain.Grid().Normalize(cell(a.Col+d.Col, a.Row+d.Row))
	return ok && canStep(terrain, a, beside) && canStep(terrain, beside, b)
}

// shortestCost is a plain Dijkstra on the 8 x 8 map, with the same step rules: the answer A* must find.
func shortestCost(terrain domain.Terrain, from, to core.Cell) (float64, bool) {
	cost := map[core.Cell]float64{from: 0}
	done := map[core.Cell]bool{}
	for {
		current, best := core.Cell{}, math.Inf(1)
		for c, v := range cost {
			if !done[c] && v < best {
				current, best = c, v
			}
		}
		if math.IsInf(best, 1) {
			return 0, false
		}
		if current == to {
			return best, true
		}
		done[current] = true
		for dc := -1; dc <= 1; dc++ {
			for dr := -1; dr <= 1; dr++ {
				next, ok := terrain.Grid().Normalize(cell(current.Col+dc, current.Row+dr))
				if !ok || next == current || !canStep(terrain, current, next) {
					continue
				}
				if dc != 0 && dr != 0 && (!goesRound(terrain, current, next, cell(dc, 0)) || !goesRound(terrain, current, next, cell(0, dr))) {
					continue
				}
				if c, ok := cost[next]; !ok || best+domain.StepCost(current, next) < c {
					cost[next] = best + domain.StepCost(current, next)
				}
			}
		}
	}
}

// drawnMap is a bounded map drawn in text, a character per cell: '0' to '3' grass at that height, 'w' water at -1,
// 'h' a hole, 'b' a bridge, 'o' the ocean, 'm' a mountain, 'g' a glacier and 'r' a rock, at 0.
func drawnMap(rows ...string) domain.Map {
	m := mapWithRocks(len(rows[0]), len(rows))
	kinds := map[rune]domain.TileKind{
		'w': domain.TileWater, 'h': domain.TileHole, 'b': domain.TileBridge, 'o': domain.TileOcean, 'm': domain.TileMountain, 'g': domain.TileGlacier,
	}
	for row, line := range rows {
		for col, char := range line {
			c := cell(col, row)
			switch {
			case char >= '0' && char <= '3':
				m.SetHeight(c, int8(char-'0'))
			case char == 'r':
				m.Objects = append(m.Objects, domain.Object{Kind: domain.ObjectRock, Col: col, Row: row})
			case char == 'w':
				m.SetGround(c, kinds[char])
				m.SetHeight(c, -1)
			default:
				m.SetGround(c, kinds[char])
			}
		}
	}
	return m
}

// ADR 0007 test 3
func TestStepsOnHeights(t *testing.T) {
	t.Run("one level up or down is a step", func(t *testing.T) {
		terrain := domain.NewTerrain(drawnMap("01"))
		assert.Equal(t, []core.Cell{cell(1, 0)}, terrain.Path(cell(0, 0), cell(1, 0)))
		assert.Equal(t, []core.Cell{cell(0, 0)}, terrain.Path(cell(1, 0), cell(0, 0)))
	})

	t.Run("two levels is a cliff, either way", func(t *testing.T) {
		terrain := domain.NewTerrain(drawnMap("02"))
		assert.Nil(t, terrain.Path(cell(0, 0), cell(1, 0)))
		assert.Nil(t, terrain.Path(cell(1, 0), cell(0, 0)))
	})

	for name, side := range map[string]string{"a cliff": "02", "a hole": "0h", "water": "0w", "a mountain": "0m", "a glacier": "0g"} {
		t.Run("no diagonal past "+name+" beside it", func(t *testing.T) {
			terrain := domain.NewTerrain(drawnMap(side, "01"))
			assert.Equal(t, []core.Cell{cell(0, 1), cell(1, 1)}, terrain.Path(cell(0, 0), cell(1, 1)))
		})
	}
}

// ADR 0009 test 2
func TestPathsGoAroundMountainsAndGlaciers(t *testing.T) {
	terrain := domain.NewTerrain(drawnMap(
		"0m0",
		"0g0",
		"000",
	))
	path := terrain.Path(cell(0, 0), cell(2, 0))
	assert.Len(t, path, 6, "down the left side, along the bottom and up the right: %v", path)
	assert.NotContains(t, path, cell(1, 0))
	assert.NotContains(t, path, cell(1, 1))
	assert.Nil(t, terrain.Path(cell(0, 0), cell(1, 0)), "a mountain is no target")
	assert.Nil(t, terrain.Path(cell(0, 0), cell(1, 1)), "nor a glacier")
}

// ADR 0007 test 4
func TestPathsOnTerrain(t *testing.T) {
	t.Run("a hill of terraces is climbed ring by ring", func(t *testing.T) {
		terrain := domain.NewTerrain(drawnMap(
			"0000000",
			"0111110",
			"0122210",
			"0123210",
			"0122210",
			"0111110",
			"0000000",
		))
		assert.Equal(t, []core.Cell{cell(1, 1), cell(2, 2), cell(3, 3)}, terrain.Path(cell(0, 0), cell(3, 3)))
	})

	t.Run("a plateau ringed by a cliff can't be reached", func(t *testing.T) {
		terrain := domain.NewTerrain(drawnMap(
			"00000",
			"02220",
			"02220",
			"02220",
			"00000",
		))
		assert.Nil(t, terrain.Path(cell(0, 0), cell(2, 2)))
		assert.Nil(t, terrain.Path(cell(0, 0), cell(1, 1)), "its edge either")
	})

	t.Run("the river is crossed over a bridge only", func(t *testing.T) {
		river := []string{"0000000000", "wwwwwwwbbw", "0000000000"}
		path := domain.NewTerrain(drawnMap(river...)).Path(cell(0, 0), cell(0, 2))
		require.NotEmpty(t, path)
		crossed := false
		for _, step := range path {
			if step.Row == 1 {
				assert.Contains(t, []int{7, 8}, step.Col, "%v is on the bridge", step)
				crossed = true
			}
		}
		assert.True(t, crossed)

		river[1] = "wwwwwwwwww"
		assert.Nil(t, domain.NewTerrain(drawnMap(river...)).Path(cell(0, 0), cell(0, 2)), "no bridge, no way across")
		river[1] = "oooooooooo"
		assert.Nil(t, domain.NewTerrain(drawnMap(river...)).Path(cell(0, 0), cell(0, 2)), "the ocean is never crossed (ADR 0008)")
	})
}

// ADR 0007 test 5
func TestAreasFollowSteps(t *testing.T) {
	t.Run("a cliff line splits the map, a ramp joins it", func(t *testing.T) {
		split := domain.NewTerrain(drawnMap("0022", "0022", "0022"))
		assert.Nil(t, split.Path(cell(0, 0), cell(3, 0)))
		ramp := domain.NewTerrain(drawnMap("0022", "0012", "0022"))
		assert.NotEmpty(t, ramp.Path(cell(0, 0), cell(3, 0)))
	})

	t.Run("a target behind a cliff is refused without searching", func(t *testing.T) {
		m := mapWithRocks(256, 256)
		m.SetHeight(cell(255, 255), 2)
		terrain := domain.NewTerrain(m).WithSearchLimit(1 << 30)
		start := time.Now()
		assert.Nil(t, terrain.Path(cell(0, 0), cell(255, 255)))
		assert.Less(t, time.Since(start), 5*time.Millisecond, "the areas say it's cut off: no search")
	})
}

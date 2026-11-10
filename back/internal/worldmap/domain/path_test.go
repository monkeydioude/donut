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

// ADR 0003 test 8, and ADR 0002 test 6
func TestPathsOnRandomMaps(t *testing.T) {
	rng := seeded(3)
	for i := range 400 {
		m := domain.Generate(domain.GenerateOptions{Width: 8, Height: 8, Wraps: i%2 == 1, RockChance: 0.3}, rng)
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

// checkSteps checks each step is a walkable neighbor that cuts no corner, and gives the path's cost.
func checkSteps(t *testing.T, terrain domain.Terrain, from core.Cell, path []core.Cell) float64 {
	t.Helper()
	grid := terrain.Grid()
	cost := 0.0
	for _, step := range path {
		require.True(t, grid.AreNeighbors(from, step), "%v to %v is one step", from, step)
		require.True(t, terrain.IsWalkable(step), "%v is walkable", step)
		if step.Col != from.Col && step.Row != from.Row {
			d := grid.Offset(from, step)
			require.True(t, walkableAt(terrain, from, cell(d.Col, 0)) && walkableAt(terrain, from, cell(0, d.Row)),
				"%v to %v cuts no corner", from, step)
		}
		cost += domain.StepCost(from, step)
		from = step
	}
	return cost
}

// walkableAt: the cell `d` away from c is walkable, across the seams on a wrapping map.
func walkableAt(terrain domain.Terrain, c, d core.Cell) bool {
	next, ok := terrain.Grid().Normalize(cell(c.Col+d.Col, c.Row+d.Row))
	return ok && terrain.IsWalkable(next)
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
				if !ok || next == current || !terrain.IsWalkable(next) {
					continue
				}
				if dc != 0 && dr != 0 && (!walkableAt(terrain, current, cell(dc, 0)) || !walkableAt(terrain, current, cell(0, dr))) {
					continue
				}
				if c, ok := cost[next]; !ok || best+domain.StepCost(current, next) < c {
					cost[next] = best + domain.StepCost(current, next)
				}
			}
		}
	}
}

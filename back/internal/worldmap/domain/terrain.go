package domain

import (
	"fmt"

	core "donut/internal/core/domain"
	"donut/pkg/errd"
)

// DefaultSearchLimit bounds how many cells a path search explores: past it, there is no path.
// It keeps a request cheap (about 10 ms at worst) on the biggest maps, where the room's lock is held meanwhile.
const DefaultSearchLimit = 8192

// MaxClimb is the most levels a step goes up or down: more is a cliff (ADR 0007).
const MaxClimb = 1

// Terrain is where one can walk on a map: on grass or a bridge, with nothing standing on it, one level up or down
// at most per step. On a wrapping map, walking goes across the seams.
type Terrain struct {
	grid core.Grid
	// walkable, heights and areas are per cell, row by row. areas numbers the walkable cells by connected area:
	// a path only links cells of one area.
	walkable    []bool
	heights     []int8
	areas       []int32
	searchLimit int
	spawn       core.Cell
	spawnErr    error
}

func NewTerrain(m Map) Terrain {
	t := Terrain{grid: m.Grid(), walkable: walkableCells(m), heights: cellHeights(m), searchLimit: DefaultSearchLimit}
	t.areas = t.labelAreas()
	t.spawn, t.spawnErr = t.closestToCenter()
	return t
}

// walkableCells: every cell whose ground is walkable, but those an object covers.
func walkableCells(m Map) []bool {
	grid := m.Grid()
	walkable := make([]bool, grid.Width*grid.Height)
	for _, chunk := range m.Chunks {
		for i, kind := range chunk.Ground {
			if cell := chunk.cellAt(i); kind.Walkable() && grid.Contains(cell) {
				walkable[grid.Index(cell)] = true
			}
		}
	}
	for _, object := range m.Objects {
		for _, cell := range object.Footprint() {
			if grid.Contains(cell) {
				walkable[grid.Index(cell)] = false
			}
		}
	}
	return walkable
}

// cellHeights: every cell's height, 0 in missing chunks.
func cellHeights(m Map) []int8 {
	grid := m.Grid()
	heights := make([]int8, grid.Width*grid.Height)
	for _, chunk := range m.Chunks {
		for i, height := range chunk.Height {
			if cell := chunk.cellAt(i); grid.Contains(cell) {
				heights[grid.Index(cell)] = height
			}
		}
	}
	return heights
}

// WithSearchLimit is the same terrain with another search limit.
func (t Terrain) WithSearchLimit(cells int) Terrain {
	t.searchLimit = cells
	return t
}

func (t Terrain) Grid() core.Grid {
	return t.grid
}

// IsWalkable: a cell within the map, on ground, with nothing on it.
func (t Terrain) IsWalkable(c core.Cell) bool {
	return t.grid.Contains(c) && t.walkable[t.grid.Index(c)]
}

// HeightAt is a cell's height within the map, 0 outside.
func (t Terrain) HeightAt(c core.Cell) int8 {
	if !t.grid.Contains(c) {
		return 0
	}
	return t.heights[t.grid.Index(c)]
}

// SpawnPoint is the walkable cell closest to the map's center. Ties go to the lowest row, then column.
func (t Terrain) SpawnPoint() (core.Cell, error) {
	return t.spawn, t.spawnErr
}

// sameArea: both cells are walkable and connected.
func (t Terrain) sameArea(a, b core.Cell) bool {
	return t.IsWalkable(a) && t.IsWalkable(b) && t.areas[t.grid.Index(a)] == t.areas[t.grid.Index(b)]
}

// neighbor is the cell one step `d` from c, across the seams on a wrapping map. False past a bounded map's edges.
func (t Terrain) neighbor(c, d core.Cell) (core.Cell, bool) {
	return t.grid.Normalize(core.Cell{Col: c.Col + d.Col, Row: c.Row + d.Row})
}

// labelAreas floods every walkable cell with the number of its area. Straight steps are enough:
// a diagonal step needs both straight ways round to be steps, so it never links two areas.
func (t Terrain) labelAreas() []int32 {
	areas := make([]int32, len(t.walkable))
	var next int32
	for row := range t.grid.Height {
		for col := range t.grid.Width {
			start := core.Cell{Col: col, Row: row}
			if t.IsWalkable(start) && areas[t.grid.Index(start)] == 0 {
				next++
				t.flood(areas, start, next)
			}
		}
	}
	return areas
}

func (t Terrain) flood(areas []int32, start core.Cell, area int32) {
	areas[t.grid.Index(start)] = area
	for queue := []core.Cell{start}; len(queue) > 0; queue = queue[1:] {
		for _, d := range stepDirections[:4] {
			next, ok := t.step(queue[0], d)
			if ok && areas[t.grid.Index(next)] == 0 {
				areas[t.grid.Index(next)] = area
				queue = append(queue, next)
			}
		}
	}
}

func (t Terrain) closestToCenter() (core.Cell, error) {
	centerCol, centerRow := float64(t.grid.Width-1)/2, float64(t.grid.Height-1)/2
	best, bestDistance, found := core.Cell{}, 0.0, false
	for row := range t.grid.Height {
		for col := range t.grid.Width {
			cell := core.Cell{Col: col, Row: row}
			distance := square(float64(col)-centerCol) + square(float64(row)-centerRow)
			if t.IsWalkable(cell) && (!found || distance < bestDistance) {
				best, bestDistance, found = cell, distance, true
			}
		}
	}
	if !found {
		return core.Cell{}, fmt.Errorf("SpawnPoint: no walkable cell: %w", errd.ErrConflict)
	}
	return best, nil
}

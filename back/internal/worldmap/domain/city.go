package domain

import (
	"math"

	core "donut/internal/core/domain"
)

// CityOptions describe a dome city's map (ADR 0012): a round floor under its dome, ringed by the dome's wall, with a
// park of grass in the middle. The corners past the wall are void.
type CityOptions struct {
	Slug string
	Name string
	// Size is the map's side, in cells: a multiple of LocalSide.
	Size int
	// Park is the side of the grass square in the middle, in cells.
	Park int
}

// city's wall heights: a cutaway, so the city behind the front of the ring stays in view.
const (
	backWallHeight  = 3
	sideWallHeight  = 2
	frontWallHeight = 1
)

// Doma1 is the first dome city: 32 × 32 cells, on 2 × 2 world cells (ADR 0012).
func Doma1() CityOptions {
	return CityOptions{Slug: "doma-1", Name: "Doma-1", Size: 2 * LocalSide, Park: 6}
}

// GenerateCity makes a city's map. Measured from its middle to each cell's middle: floor up to Size/2 − 1.5, the
// park's square in the middle in grass, a wall 1 cell thick up to Size/2 − 0.5, void past it. The floor is at 0;
// the wall at +3 on the back half of the ring (toward the top of the screen), +1 on the front half, +2 where they
// meet. Every cell but the void is in the city's only region.
func GenerateCity(opts CityOptions) Map {
	summary := Summary{Slug: opts.Slug, Name: opts.Name, Width: opts.Size, Height: opts.Size}
	m := NewMap(summary, TileVoid)
	m.Regions = []Region{{ID: 1, Name: opts.Name, Biome: BiomeDome}}
	floorReach, wallReach := float64(opts.Size)/2-1.5, float64(opts.Size)/2-0.5
	for row := range opts.Size {
		for col := range opts.Size {
			cell := core.Cell{Col: col, Row: row}
			reach := fromMiddle(opts.Size, cell)
			switch {
			case reach <= floorReach:
				m.SetGround(cell, cityGround(opts, cell))
			case reach <= wallReach:
				m.SetGround(cell, TileWall)
				m.SetHeight(cell, wallHeight(opts.Size, cell))
			default:
				continue
			}
			m.SetRegion(cell, 1)
		}
	}
	return m
}

// fromMiddle is how far a cell's middle is from the map's middle, in cells.
func fromMiddle(size int, c core.Cell) float64 {
	middle := float64(size) / 2
	return math.Hypot(float64(c.Col)+0.5-middle, float64(c.Row)+0.5-middle)
}

// cityGround is the park's grass on the square in the middle, the floor elsewhere.
func cityGround(opts CityOptions, c core.Cell) TileKind {
	low, high := (opts.Size-opts.Park)/2, (opts.Size+opts.Park)/2
	if c.Col >= low && c.Col < high && c.Row >= low && c.Row < high {
		return TileGrass
	}
	return TileFloor
}

// wallHeight: the cells of the ring above the map's middle on screen (col + row smaller) are the back half.
func wallHeight(size int, c core.Cell) int8 {
	middle := size - 1
	switch depth := c.Col + c.Row; {
	case depth < middle-1:
		return backWallHeight
	case depth > middle+1:
		return frontWallHeight
	}
	return sideWallHeight
}

package domain

import (
	core "donut/internal/core/domain"
)

// Rng gives floats in [0, 1). *rand.Rand of math/rand/v2 is one.
type Rng interface {
	Float64() float64
}

// GenerateOptions describe a generated map.
type GenerateOptions struct {
	Slug   string
	Name   string
	Width  int
	Height int
	Wraps  bool
	// RockChance is the chance in [0, 1] for each cell to get a rock.
	RockChance float64
	// RegionNames are the regions to cut the map into, ids from 1 in this order. None: no regions.
	RegionNames []string
}

// starterRegionNames are placeholders: they are stored with the world, to rename.
var starterRegionNames = []string{
	"Sprinkle Meadow", "Glaze Hills", "Cinnamon Vale", "Maple Hollow", "Honey Downs", "Sugar Fields",
	"Cocoa Glen", "Frosting Flats", "Crumb Valley", "Jelly Ridge", "Cruller Heath", "Powder Plains",
}

// StarterWorld is the world the server creates when the database has no map: the surface of the donut,
// 1024 cells around the ring and 512 around the tube, wrapping both ways (ADR 0003).
func StarterWorld() GenerateOptions {
	return GenerateOptions{
		Slug: "world", Name: "World", Width: 1024, Height: 512, Wraps: true,
		RockChance: 0.12, RegionNames: starterRegionNames,
	}
}

// Generate makes a grass map cut into regions, with rocks scattered on it. The same rng sequence gives the same map:
// the regions' points are drawn first, then each cell's rock, row by row.
func Generate(opts GenerateOptions, rng Rng) Map {
	summary := Summary{Slug: opts.Slug, Name: opts.Name, Width: opts.Width, Height: opts.Height, Wraps: opts.Wraps}
	m := NewMap(summary, TileGrass)
	if len(opts.RegionNames) > 0 {
		m.Regions = regionsOf(opts.RegionNames)
		cutIntoRegions(m, regionPoints(len(opts.RegionNames), summary.Grid(), rng))
	}
	m.Objects = scatterRocks(summary.Grid(), opts.RockChance, rng)
	return m
}

func regionsOf(names []string) []Region {
	regions := make([]Region, len(names))
	for i, name := range names {
		regions[i] = Region{ID: i + 1, Name: name, Biome: BiomeMeadow}
	}
	return regions
}

// point is a place on the map, between cells.
type point struct{ col, row float64 }

func regionPoints(count int, grid core.Grid, rng Rng) []point {
	points := make([]point, count)
	for i := range points {
		points[i] = point{col: rng.Float64() * float64(grid.Width), row: rng.Float64() * float64(grid.Height)}
	}
	return points
}

// cutIntoRegions puts each cell in the region of the closest point, distances wrapping on a wrapping map.
// Ties go to the region found first.
func cutIntoRegions(m Map, points []point) {
	grid := m.Grid()
	for row := range grid.Height {
		for col := range grid.Width {
			best, bestDistance := 0, -1.0
			for i, p := range points {
				distance := square(axisDistance(float64(col), p.col, grid.Width, grid.Wraps)) +
					square(axisDistance(float64(row), p.row, grid.Height, grid.Wraps))
				if bestDistance < 0 || distance < bestDistance {
					best, bestDistance = i, distance
				}
			}
			m.SetRegion(core.Cell{Col: col, Row: row}, uint8(best+1))
		}
	}
}

// axisDistance is how far apart a and b are on an axis of `size`, the short way round when it wraps.
func axisDistance(a, b float64, size int, wraps bool) float64 {
	d := a - b
	if d < 0 {
		d = -d
	}
	if wraps && d > float64(size)/2 {
		d = float64(size) - d
	}
	return d
}

func scatterRocks(grid core.Grid, chance float64, rng Rng) []Object {
	rocks := []Object{}
	for row := range grid.Height {
		for col := range grid.Width {
			if rng.Float64() < chance {
				rocks = append(rocks, Object{Kind: ObjectRock, Col: col, Row: row})
			}
		}
	}
	return rocks
}

func square(x float64) float64 {
	return x * x
}

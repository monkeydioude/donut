package domain

import (
	"fmt"

	core "donut/internal/core/domain"
)

// regionPoint is where a region is centered, and its id.
type regionPoint struct {
	at point
	id uint8
}

// cutIntoRegions names the map's regions and gives each cell one. With landmasses (ADR 0008), each landmass is cut
// among its own points, and the ocean among the oceans' points, so a region never spans two landmasses, nor land and
// sea. Without, the whole map is cut among as many points as names (ADR 0003).
func (l *land) cutIntoRegions(m *Map, masses []landmass, o LandmassOptions, names []string, center core.Cell, rng Rng) {
	if len(masses) == 0 {
		if len(names) > 0 {
			m.Regions = regionsOf(names)
			cutByClosest(*m, regionPoints(len(names), l.grid, rng))
		}
		return
	}
	perMass, count := landmassPoints(l, masses, o, rng)
	oceans := oceanPoints(l.grid, o.OceanRegions, count, center)
	count += len(oceans)
	m.Regions = regionsOf(regionNames(names, count))
	for i := range l.ground {
		points := oceans
		if l.landmass[i] >= 0 {
			points = perMass[l.landmass[i]]
		}
		m.SetRegion(l.cellAt(i), closestRegion(l.grid, pointOf(l.cellAt(i)), points))
	}
}

// landmassPoints draws each landmass's region points on its cells, ids from 1 in order, and counts them.
func landmassPoints(l *land, masses []landmass, o LandmassOptions, rng Rng) ([][]regionPoint, int) {
	perMass := make([][]regionPoint, len(masses))
	id := 0
	for k, m := range masses {
		count := o.RegionsPerIsland
		if m.continent {
			count = o.RegionsPerContinent
		}
		for range count {
			if len(m.cells) == 0 {
				break
			}
			id++
			cell := m.cells[int(rng.Float64()*float64(len(m.cells)))]
			perMass[k] = append(perMass[k], regionPoint{at: pointOf(l.cellAt(cell)), id: uint8(id)})
		}
	}
	return perMass, id
}

// oceanPoints are evenly spaced around the ring, between the continents, on the center's row, ids after `from`.
func oceanPoints(grid core.Grid, count, from int, center core.Cell) []regionPoint {
	var points []regionPoint
	for k := range count {
		at := point{col: float64(center.Col) + (float64(k)+0.5)*float64(grid.Width)/float64(count), row: float64(center.Row)}
		points = append(points, regionPoint{at: wrapPoint(grid, at), id: uint8(from + k + 1)})
	}
	return points
}

// closestRegion is the region of the closest point, the first on a tie; 0 with no points.
func closestRegion(grid core.Grid, at point, points []regionPoint) uint8 {
	best, bestDistance := uint8(0), -1.0
	for _, p := range points {
		if d := distance(grid, at, p.at); bestDistance < 0 || d < bestDistance {
			best, bestDistance = p.id, d
		}
	}
	return best
}

// regionNames are `count` names: the ones given, then placeholders.
func regionNames(names []string, count int) []string {
	out := make([]string, count)
	for i := range out {
		if i < len(names) {
			out[i] = names[i]
		} else {
			out[i] = fmt.Sprintf("Region %d", i+1)
		}
	}
	return out
}

func regionsOf(names []string) []Region {
	regions := make([]Region, len(names))
	for i, name := range names {
		regions[i] = Region{ID: i + 1, Name: name, Biome: BiomeMeadow}
	}
	return regions
}

func regionPoints(count int, grid core.Grid, rng Rng) []point {
	points := make([]point, count)
	for i := range points {
		points[i] = point{col: rng.Float64() * float64(grid.Width), row: rng.Float64() * float64(grid.Height)}
	}
	return points
}

// cutByClosest puts each cell in the region of the closest point, distances wrapping on a wrapping map.
// Ties go to the region found first.
func cutByClosest(m Map, points []point) {
	grid := m.Grid()
	regions := make([]regionPoint, len(points))
	for i, p := range points {
		regions[i] = regionPoint{at: p, id: uint8(i + 1)}
	}
	for row := range grid.Height {
		for col := range grid.Width {
			cell := core.Cell{Col: col, Row: row}
			m.SetRegion(cell, closestRegion(grid, pointOf(cell), regions))
		}
	}
}

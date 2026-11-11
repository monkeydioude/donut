package domain

import (
	core "donut/internal/core/domain"
)

// LandmassOptions shape the continents and islands in the ocean (ADR 0008). No continents nor islands: all land.
type LandmassOptions struct {
	Continents, Islands int
	// ContinentRadius and IslandRadius are their plain radius, in cells, before the coast noise.
	ContinentRadius, IslandRadius int
	// Jitter is how far a continent's seed strays from its even place around the ring, at most, either way.
	Jitter int
	// Coast is how much the coast noise stretches or shrinks a radius, at most: ±Coast × radius.
	Coast float64
	// CoastWavelengths are the coast noise's octaves, in cells (see ReliefOptions.Wavelengths).
	CoastWavelengths []int
	// Clearance is the open sea between two landmasses at their widest, at least.
	Clearance int
	// InlandRise is added to the relief noise (−1…1) of a cell InlandReach steps or more from the ocean, and in
	// proportion closer to it.
	InlandRise  float64
	InlandReach int
	// Regions per landmass, and over the ocean.
	RegionsPerContinent, RegionsPerIsland, OceanRegions int
}

// starterLandmasses: 3 continents of about 11% of the map each, 6 islands of about 1%, 40% land in all.
func starterLandmasses() LandmassOptions {
	return LandmassOptions{
		Continents: 3, Islands: 6, ContinentRadius: 270, IslandRadius: 80, Jitter: 40,
		Coast: 0.3, CoastWavelengths: []int{256, 128, 64, 32}, Clearance: 40,
		InlandRise: 0.6, InlandReach: 120,
		RegionsPerContinent: 3, RegionsPerIsland: 1, OceanRegions: 3,
	}
}

// landmass is a continent or an island: its seed, its plain radius, and the cells it got.
type landmass struct {
	seed      point
	radius    int
	continent bool
	cells     []int
}

// widest is the landmass's radius at the coast noise's peak.
func (m landmass) widest(o LandmassOptions) float64 {
	return float64(m.radius) * (1 + o.Coast)
}

// islandTries is how many places an island is tried at before it's dropped.
const islandTries = 200

// shapeLand places the landmasses and makes every other cell ocean, at −1. A cell is land when it's within some
// landmass's radius, stretched by the coast noise; it belongs to the landmass it's closest to, relative to its radius.
func (l *land) shapeLand(o LandmassOptions, center core.Cell, rng Rng) []landmass {
	if o.Continents+o.Islands == 0 {
		return nil
	}
	masses := continentSeeds(o, l.grid, center, rng)
	masses = islandSeeds(o, l.grid, masses, rng)
	noise := fractalNoise(l.grid, o.CoastWavelengths, rng)
	for i := range l.ground {
		at := pointOf(l.cellAt(i))
		best, bestShare := -1, 0.0
		for k, m := range masses {
			share := distance(l.grid, at, m.seed) / float64(m.radius)
			if share < 1+o.Coast*noise[i] && (best < 0 || share < bestShare) {
				best, bestShare = k, share
			}
		}
		if best < 0 {
			l.landmass[i], l.ground[i], l.height[i] = -1, TileOcean, -1
		} else {
			l.landmass[i] = int16(best)
			masses[best].cells = append(masses[best].cells, i)
		}
	}
	l.measureToOcean()
	return masses
}

// continentSeeds are evenly spaced around the ring, the first at the center, the others a quarter of the height
// above and below it in turn, each moved by up to the jitter.
func continentSeeds(o LandmassOptions, grid core.Grid, center core.Cell, rng Rng) []landmass {
	var masses []landmass
	for k := range o.Continents {
		seed := point{
			col: float64(center.Col) + float64(k)*float64(grid.Width)/float64(o.Continents),
			row: float64(center.Row) + continentRowOffset(k, grid.Height),
		}
		if k > 0 {
			seed.col += float64(Span{Min: -o.Jitter, Max: o.Jitter}.pick(rng))
			seed.row += float64(Span{Min: -o.Jitter, Max: o.Jitter}.pick(rng))
		}
		masses = append(masses, landmass{seed: wrapPoint(grid, seed), radius: o.ContinentRadius, continent: true})
	}
	return masses
}

func continentRowOffset(k, height int) float64 {
	switch {
	case k == 0:
		return 0
	case k%2 == 1:
		return -float64(height) / 4
	}
	return float64(height) / 4
}

// islandSeeds draws each island's seed at random, with the clearance from every landmass placed so far. An island
// that finds no place is dropped.
func islandSeeds(o LandmassOptions, grid core.Grid, masses []landmass, rng Rng) []landmass {
	for range o.Islands {
		for try := 0; try < islandTries; try++ {
			island := landmass{seed: point{col: rng.Float64() * float64(grid.Width), row: rng.Float64() * float64(grid.Height)}, radius: o.IslandRadius}
			if isClear(island, masses, o, grid) {
				masses = append(masses, island)
				break
			}
		}
	}
	return masses
}

// isClear: the landmass is at least the clearance from every other, both at their widest.
func isClear(m landmass, others []landmass, o LandmassOptions, grid core.Grid) bool {
	for _, other := range others {
		if distance(grid, m.seed, other.seed) < m.widest(o)+other.widest(o)+float64(o.Clearance) {
			return false
		}
	}
	return true
}

// inlandRise is what each cell's relief noise gains for being far from the ocean: nothing on a map without one.
func (l *land) inlandRise(o LandmassOptions) func(i int) float64 {
	if !l.ocean || o.InlandReach <= 0 {
		return func(int) float64 { return 0 }
	}
	return func(i int) float64 {
		return o.InlandRise * float64(min(l.toOcean[i], int32(o.InlandReach))) / float64(o.InlandReach)
	}
}

// holds: the cell is on landmass k.
func (l *land) holds(k int, c core.Cell) bool {
	i, ok := l.index(c)
	return ok && int(l.landmass[i]) == k
}

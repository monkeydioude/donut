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
	// RockChance is the chance in [0, 1] for each grass cell to get a rock.
	RockChance float64
	// RegionNames name the regions, ids from 1 in this order. With landmasses (ADR 0008), each continent's regions
	// come first, then each island's, then the oceans'; without, as many regions are cut over the whole map.
	// None: no regions.
	RegionNames []string
	// Relief, Landmasses, Water, Holes, Mountains and Glaciers shape the ground (ADR 0007, ADR 0008, ADR 0011). Their
	// zero values leave it flat grass.
	Relief     ReliefOptions
	Landmasses LandmassOptions
	Water      WaterOptions
	Holes      HoleOptions
	Mountains  MountainOptions
	Glaciers   GlacierOptions
	// Dome places a dome city on the spawn (ADR 0012).
	Dome DomeOptions
}

// Span is a whole number from Min to Max, both included.
type Span struct{ Min, Max int }

// pick is a number of the span, each as likely.
func (s Span) pick(rng Rng) int {
	return s.Min + int(rng.Float64()*float64(s.Max-s.Min+1))
}

// starterRegionNames are placeholders: they are stored with the world, to rename. The continents' regions come
// first, then the islands', then the three seas.
func starterRegionNames() []string {
	return []string{
		"Sprinkle Meadow", "Glaze Hills", "Cinnamon Vale", "Maple Hollow", "Honey Downs", "Sugar Fields",
		"Cocoa Glen", "Frosting Flats", "Crumb Valley", "Jelly Ridge", "Cruller Heath", "Powder Plains",
		"Caramel Shore", "Nougat Cliffs", "Praline Woods",
		"Glaze Sea", "Syrup Sea", "Custard Sea",
	}
}

// StarterWorld is the world the starter map file is generated from: the surface of the donut, 2048 cells around the
// ring and 1024 around the tube, wrapping both ways (ADR 0003): three continents and six islands in an ocean, with
// hills, lakes, rivers and holes (ADR 0007, ADR 0008), massifs around the summits and the glacier ring at the inner
// ridge (ADR 0011), and Doma-1's dome on the spawn (ADR 0012).
func StarterWorld() GenerateOptions {
	const width, height = 2048, 1024
	// the spawn is the walkable cell closest to the center: the first continent is centered on it, and a river
	// passes by it
	center := core.Cell{Col: width/2 - 1, Row: height/2 - 1}
	return GenerateOptions{
		Slug: "world", Name: "World", Width: width, Height: height, Wraps: true,
		RockChance: 0.12, RegionNames: starterRegionNames(),
		Relief: starterRelief(), Landmasses: starterLandmasses(), Water: starterWater(center), Holes: starterHoles(),
		Mountains: starterMountains(center), Glaciers: starterGlaciers(center),
		Dome: DomeOptions{LeadsTo: "doma-1"},
	}
}

// Generate makes a map from its options. The same rng sequence gives the same map: the landmasses are shaped first,
// then the relief, the lakes, the rivers, the dome, the bridges, the holes, the massifs, the glacier ring, each cell's
// rock, row by row, and the regions. The dome draws nothing from the rng.
func Generate(opts GenerateOptions, rng Rng) Map {
	summary := Summary{Slug: opts.Slug, Name: opts.Name, Width: opts.Width, Height: opts.Height, Wraps: opts.Wraps}
	m := NewMap(summary, TileGrass)
	center := core.Cell{Col: opts.Width/2 - 1, Row: opts.Height/2 - 1}
	ground := newLand(summary.Grid())
	masses := ground.shapeLand(opts.Landmasses, center, rng)
	ground.raise(opts.Relief, ground.inlandRise(opts.Landmasses), rng)
	lakes := ground.digLakes(opts.Water, masses, rng)
	rivers := ground.runRivers(opts.Water, lakes, rng)
	dome, hasDome := ground.placeDome(opts.Dome)
	ground.settleBanks()
	ground.buildBridges(rivers, opts.Water.Bridges)
	ground.digHoles(opts.Holes, rng)
	ground.raiseMountains(opts.Mountains, rng)
	ground.freezeRidge(opts.Glaciers, rng)
	ground.writeInto(m)
	m.Objects = scatterRocks(ground, opts.RockChance, rng)
	if hasDome {
		m.Objects = append(m.Objects, dome)
	}
	ground.cutIntoRegions(&m, masses, opts.Landmasses, opts.RegionNames, center, rng)
	return m
}

// scatterRocks draws a float for every cell, row by row, and puts a rock on the grass cells under the chance,
// leaving the kept cells free.
func scatterRocks(l *land, chance float64, rng Rng) []Object {
	rocks := []Object{}
	if chance <= 0 {
		return rocks
	}
	for i := range l.ground {
		if rng.Float64() < chance && l.ground[i] == TileGrass && !l.kept[i] {
			cell := l.cellAt(i)
			rocks = append(rocks, Object{Kind: ObjectRock, Col: cell.Col, Row: cell.Row})
		}
	}
	return rocks
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func sign(n int8) int8 {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	}
	return 0
}

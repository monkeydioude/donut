package domain

import (
	core "donut/internal/core/domain"
)

// WaterOptions are the lakes, rivers and bridges of each landmass (ADR 0008). Zero spans: none.
type WaterOptions struct {
	// Rivers per landmass, each from a source lake of its own.
	RiversPerContinent, RiversPerIsland Span
	// LakesPerContinent are the standalone lakes, with no river out. Islands get none.
	LakesPerContinent Span
	// LakeSize is how many cells a lake has.
	LakeSize Span
	// LakeFromOcean and LakeFromWater: how far a lake stays from the ocean, and from any other water, at least.
	LakeFromOcean, LakeFromWater int
	// Wobble is how far a river bends sideways, at most, as a share of its length.
	Wobble float64
	// RiverTries is how many courses a river is tried before it's dropped.
	RiverTries int
	Spawn      SpawnRiver
	Bridges    BridgeOptions
}

// SpawnRiver is the river that passes by the spawn: its lake is West columns before Near and Above rows above it,
// it flows east to the ocean, and bends by at most Wobble rows.
type SpawnRiver struct {
	Near        core.Cell
	West, Above Span
	Wobble      int
}

func starterWater(near core.Cell) WaterOptions {
	return WaterOptions{
		RiversPerContinent: Span{Min: 2, Max: 4}, RiversPerIsland: Span{Min: 0, Max: 1}, LakesPerContinent: Span{Min: 1, Max: 3},
		LakeSize: Span{Min: 6, Max: 20}, LakeFromOcean: 30, LakeFromWater: 20,
		Wobble: 0.125, RiverTries: 5,
		Spawn:   SpawnRiver{Near: near, West: Span{Min: 120, Max: 160}, Above: Span{Min: 20, Max: 25}, Wobble: 5},
		Bridges: BridgeOptions{Every: 60, MaxSpan: 3},
	}
}

// lakeTries is how many places a lake is tried at before it's skipped.
const lakeTries = 200

// spawnClearance is how far from the spawn a lake stays, so the spawn hardly moves.
const spawnClearance = 10

// lake is a flat blob of water on a landmass.
type lake struct {
	cells    []int
	center   int
	level    int8
	landmass int
	// source: a river starts here. spawn: it's the spawn river's lake, the first of all.
	source, spawn bool
}

// digLakes digs each landmass's lakes: the spawn river's first, then the source lakes of its rivers, then its
// standalone ones. A lake that finds no place is skipped.
func (l *land) digLakes(o WaterOptions, masses []landmass, rng Rng) []lake {
	var lakes []lake
	for k, m := range masses {
		rivers, standalone := o.RiversPerIsland, Span{}
		if m.continent {
			rivers, standalone = o.RiversPerContinent, o.LakesPerContinent
		}
		count := rivers.pick(rng)
		if l.holds(k, o.Spawn.Near) {
			if lk, ok := l.digSpawnLake(o, k, rng); ok {
				lakes = append(lakes, lk)
				count--
			}
		}
		for range max(count, 0) {
			if lk, ok := l.digRandomLake(o, k, masses[k], true, rng); ok {
				lakes = append(lakes, lk)
			}
		}
		for range standalone.pick(rng) {
			if lk, ok := l.digRandomLake(o, k, masses[k], false, rng); ok {
				lakes = append(lakes, lk)
			}
		}
	}
	return lakes
}

func (l *land) digSpawnLake(o WaterOptions, k int, rng Rng) (lake, bool) {
	for try := 0; try < lakeTries; try++ {
		at := core.Cell{Col: o.Spawn.Near.Col - o.Spawn.West.pick(rng), Row: o.Spawn.Near.Row - o.Spawn.Above.pick(rng)}
		if center, ok := l.index(at); ok {
			if lk, ok := l.digLake(o, k, center, true, rng); ok {
				lk.spawn = true
				return lk, true
			}
		}
	}
	return lake{}, false
}

func (l *land) digRandomLake(o WaterOptions, k int, m landmass, source bool, rng Rng) (lake, bool) {
	for try := 0; try < lakeTries && len(m.cells) > 0; try++ {
		center := m.cells[int(rng.Float64()*float64(len(m.cells)))]
		if lk, ok := l.digLake(o, k, center, source, rng); ok {
			return lk, true
		}
	}
	return lake{}, false
}

// digLake floods a blob around its center, flat: a source lake at the ground's level there, at least 0; a standalone
// one a level below it, −3 at the lowest. False when the center can't hold a lake, or the blob stays too small.
func (l *land) digLake(o WaterOptions, k, center int, source bool, rng Rng) (lake, bool) {
	if !l.canHoldLake(o, k, center) {
		return lake{}, false
	}
	cells := l.growBlob(center, o.LakeSize.pick(rng), func(i int) bool { return l.isGrass(i) && int(l.landmass[i]) == k }, rng)
	if len(cells) < o.LakeSize.Min {
		return lake{}, false
	}
	level := max(int(l.height[center]), 0)
	if !source {
		level = max(int(l.height[center])-1, MinHeight)
	}
	lk := lake{cells: cells, center: center, level: int8(level), landmass: k, source: source}
	for _, i := range cells {
		l.ground[i], l.height[i] = TileWater, lk.level
	}
	return lk, true
}

// canHoldLake: the cell is grass of the landmass, far enough from the ocean, from any other water and from the spawn.
func (l *land) canHoldLake(o WaterOptions, k, i int) bool {
	if !l.isGrass(i) || int(l.landmass[i]) != k || (l.ocean && int(l.toOcean[i]) < o.LakeFromOcean) {
		return false
	}
	if distance(l.grid, pointOf(l.cellAt(i)), pointOf(o.Spawn.Near)) < spawnClearance {
		return false
	}
	nearWater := false
	l.eachWithin(i, o.LakeFromWater, func(j, _ int) { nearWater = nearWater || isWater(l.ground[j]) })
	return !nearWater
}

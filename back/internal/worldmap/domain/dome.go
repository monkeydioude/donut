package domain

import (
	core "donut/internal/core/domain"
)

// DomeOptions place a dome city on a generated world (ADR 0012). Its zero value places none.
type DomeOptions struct {
	// LeadsTo is the slug of the city's map.
	LeadsTo string
}

// placeDome puts the dome on the spawn: the grass cell closest to the center is its back cell, moved back by one
// column or row when its footprint would cross a chunk border. The footprint and the ring of cells around it become
// grass at that cell's height, kept free of rocks and holes, and fixed: the smoothing moves the ground around them.
func (l *land) placeDome(opts DomeOptions) (Object, bool) {
	spawn, ok := l.closestGrassToCenter()
	if opts.LeadsTo == "" || !ok {
		return Object{}, false
	}
	back := fitInChunk(l.cellAt(spawn))
	dome := Object{Kind: ObjectDome, Col: back.Col, Row: back.Row, LeadsTo: opts.LeadsTo}
	level := l.height[spawn]
	for _, cell := range append(dome.Footprint(), dome.Around(l.grid)...) {
		if i, ok := l.index(cell); ok {
			l.ground[i], l.height[i], l.kept[i], l.fixed[i] = TileGrass, level, true, true
		}
	}
	return dome, true
}

// fitInChunk moves a dome's back cell back until its footprint is within the chunk of that cell.
func fitInChunk(back core.Cell) core.Cell {
	for core.ChunkOf(core.Cell{Col: back.Col + core.DomeCells - 1, Row: back.Row}) != core.ChunkOf(back) {
		back.Col--
	}
	for core.ChunkOf(core.Cell{Col: back.Col, Row: back.Row + core.DomeCells - 1}) != core.ChunkOf(back) {
		back.Row--
	}
	return back
}

// closestGrassToCenter is the grass cell closest to the map's center, by the spawn's rule (Terrain.SpawnPoint): ties
// go to the lowest row, then column. False on a map with no grass.
func (l *land) closestGrassToCenter() (int, bool) {
	centerCol, centerRow := float64(l.grid.Width-1)/2, float64(l.grid.Height-1)/2
	best, bestDistance := -1, 0.0
	for i, kind := range l.ground {
		cell := l.cellAt(i)
		distance := square(float64(cell.Col)-centerCol) + square(float64(cell.Row)-centerRow)
		if kind == TileGrass && (best < 0 || distance < bestDistance) {
			best, bestDistance = i, distance
		}
	}
	return best, best >= 0
}

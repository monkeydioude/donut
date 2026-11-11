package domain

import (
	"fmt"
	"math"

	core "donut/internal/core/domain"
)

// Destination is where a player goes through a door or a teleport: a map and a cell on it.
type Destination struct {
	Map  string
	Cell core.Cell
}

// Refusals of Enter and Exit (ADR 0012), in words for the player.
func NoDomeAt(c core.Cell) error {
	return Refusal{Reason: fmt.Sprintf("There's no dome at (%d, %d)", c.Col, c.Row)}
}

var ErrNotInCity = Refusal{Reason: "You're not in a city"}

func StandsNowhere(city string) error {
	return Refusal{Reason: fmt.Sprintf("%s stands nowhere on the world", city)}
}

func NoWayOut(city string) error {
	return Refusal{Reason: fmt.Sprintf("There's no way out of %s", city)}
}

// UnknownMap refuses a teleport to a map that doesn't exist (ADR 0010).
func UnknownMap(slug string) error { return Refusal{Reason: fmt.Sprintf("unknown map: %s", slug)} }

// vec is a direction on a map, in cells.
type vec struct{ col, row float64 }

// point is a place on a map, between cells: a cell's middle is (col + 0.5, row + 0.5).
type point struct{ col, row float64 }

// domeMiddle is the corner a dome's cells share.
func domeMiddle(d core.Dome) point {
	half := float64(core.DomeCells) / 2
	return point{col: float64(d.Back.Col) + half, row: float64(d.Back.Row) + half}
}

// cityMiddle is the middle of a map.
func cityMiddle(grid core.Grid) point {
	return point{col: float64(grid.Width) / 2, row: float64(grid.Height) / 2}
}

// directionOf is the way from a point to a cell's middle, the short way round on a wrapping grid.
func directionOf(grid core.Grid, from point, c core.Cell) vec {
	return vec{col: shortest(float64(c.Col)+0.5-from.col, grid.Width, grid.Wraps), row: shortest(float64(c.Row)+0.5-from.row, grid.Height, grid.Wraps)}
}

func shortest(d float64, size int, wraps bool) float64 {
	if !wraps {
		return d
	}
	s := float64(size)
	d = math.Mod(math.Mod(d, s)+s, s)
	if d > s/2 {
		d -= s
	}
	return d
}

// closestByDirection is the candidate whose direction from `center` is closest to `dir`: the smallest angle between
// them. The first in order wins a tie. False without candidates.
func closestByDirection(grid core.Grid, candidates []core.Cell, center point, dir vec) (core.Cell, bool) {
	best, bestCos := core.Cell{}, math.Inf(-1)
	for _, c := range candidates {
		if cos := cosine(directionOf(grid, center, c), dir); cos > bestCos {
			best, bestCos = c, cos
		}
	}
	return best, len(candidates) > 0
}

// cosine of the angle between two directions; 0 when one has no length.
func cosine(a, b vec) float64 {
	lengths := math.Hypot(a.col, a.row) * math.Hypot(b.col, b.row)
	if lengths == 0 {
		return 0
	}
	return (a.col*b.col + a.row*b.row) / lengths
}

// ArrivalInCity is where one entering a city through its dome appears (ADR 0012): the edge cell whose direction from
// the city's middle is closest to the direction of `from`, the cell they stood on, from the dome's middle. Coming from
// the dome's east, one appears on the city's east edge. False for a city without an edge.
func ArrivalInCity(edge []core.Cell, city core.Grid, dome core.Dome, world core.Grid, from core.Cell) (core.Cell, bool) {
	dir := directionOf(world, domeMiddle(dome), from)
	return closestByDirection(city, edge, cityMiddle(city), dir)
}

// ArrivalBesideDome is where one leaving a city appears on the world (ADR 0012): of the walkable cells touching its
// dome, the one whose direction from the dome's middle is closest to the direction of `from`, the edge cell they left
// by, from the city's middle. False when no cell beside the dome is walkable.
func ArrivalBesideDome(world core.Grid, dome core.Dome, walkable func(core.Cell) bool, city core.Grid, from core.Cell) (core.Cell, bool) {
	var free []core.Cell
	for _, c := range dome.Around(world) {
		if walkable(c) {
			free = append(free, c)
		}
	}
	dir := directionOf(city, cityMiddle(city), from)
	return closestByDirection(world, free, domeMiddle(dome), dir)
}

// ClosestEdge is the edge cell one walks to for leaving a city by a click on `clicked`: the closest to it, measured
// straight, once it's brought back into the map's box. The first in order wins a tie. False without an edge.
func ClosestEdge(edge []core.Cell, city core.Grid, clicked core.Cell) (core.Cell, bool) {
	target := core.Cell{Col: min(max(clicked.Col, 0), city.Width-1), Row: min(max(clicked.Row, 0), city.Height-1)}
	best, bestDistance := core.Cell{}, math.Inf(1)
	for _, c := range edge {
		if d := math.Hypot(float64(c.Col-target.Col), float64(c.Row-target.Row)); d < bestDistance {
			best, bestDistance = c, d
		}
	}
	return best, len(edge) > 0
}

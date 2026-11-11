package domain

import (
	"math"

	core "donut/internal/core/domain"
)

// station is a river's cross-section at one point of its course: its cells, their level, the flow's direction there
// and how far along the course it is, in cells.
type station struct {
	cells []int
	level int8
	dir   point
	along float64
}

// river is a course of stations, from its lake to the water it ends in.
type river struct {
	stations []station
	// bridgePhase is how far along the river its first bridge goes.
	bridgePhase float64
}

// stationsPerCell: stations are laid every half cell, so consecutive ones touch.
const stationsPerCell = 2

// runRivers lays a river from each source lake to its sink: the closest of the ocean and the lakes below it. A river
// that can't be laid is tried again with another bend, then dropped.
func (l *land) runRivers(o WaterOptions, lakes []lake, rng Rng) []river {
	var rivers []river
	for _, lk := range lakes {
		if !lk.source {
			continue
		}
		target, sinkLevel, ok := l.sinkOf(lk, lakes, o.Spawn)
		if !ok {
			continue
		}
		for try := 0; try < max(o.RiverTries, 1); try++ {
			course, ok := l.courseOf(lk, target, sinkLevel, o, rng)
			if ok {
				l.paintRiver(course)
				rivers = append(rivers, river{stations: course, bridgePhase: l.bridgePhase(lk, course, o)})
				break
			}
		}
	}
	return rivers
}

// sinkOf is the cell a lake's river heads for and the water's level there: the spawn river's is the ocean east of
// its lake; any other's the closest of the ocean and the lakes of its landmass at least one level below it.
func (l *land) sinkOf(lk lake, lakes []lake, spawn SpawnRiver) (int, int8, bool) {
	if lk.spawn {
		return l.oceanEastOf(lk.center)
	}
	from := pointOf(l.cellAt(lk.center))
	target, level, best := -1, int8(0), math.Inf(1)
	consider := func(cell int, cellLevel int8) {
		if d := distance(l.grid, from, pointOf(l.cellAt(cell))); d < best {
			target, level, best = cell, cellLevel, d
		}
	}
	if l.ocean {
		consider(int(l.nearestOcean[lk.center]), -1)
	}
	for _, other := range lakes {
		if other.landmass == lk.landmass && other.center != lk.center && other.level < lk.level {
			consider(other.center, other.level)
		}
	}
	return target, level, target >= 0
}

// oceanEastOf is the first ocean cell east of a cell, on its row.
func (l *land) oceanEastOf(from int) (int, int8, bool) {
	c := l.cellAt(from)
	for d := 1; d < l.grid.Width; d++ {
		if i, ok := l.index(core.Cell{Col: c.Col + d, Row: c.Row}); ok && l.ground[i] == TileOcean {
			return i, -1, true
		}
	}
	return -1, 0, false
}

// courseOf lays out a river's stations from its lake to a target, without painting them: a straight course bent by
// a smooth wobble, its level going from the lake's down to the sink's one at a time, its width from 1 to 3. It ends
// at the first water it meets. False when it meets water higher than itself, or would be too steep.
func (l *land) courseOf(lk lake, target int, sinkLevel int8, o WaterOptions, rng Rng) ([]station, bool) {
	start := l.cellAt(l.closestTo(lk.cells, target))
	offset := l.grid.Offset(start, l.cellAt(target))
	length := math.Hypot(float64(offset.Col), float64(offset.Row))
	drop := int(lk.level) - int(sinkLevel)
	if length < 2 || length < 3*float64(drop) {
		return nil, false
	}
	along := point{col: float64(offset.Col) / length, row: float64(offset.Row) / length}
	across := point{col: -along.row, row: along.col}
	amplitude := o.Wobble * length
	if lk.spawn {
		amplitude = min(amplitude, float64(o.Spawn.Wobble))
	}
	bend := newWobble(amplitude, rng)
	at := func(t float64) point {
		return point{col: float64(start.Col) + along.col*length*t + across.col*bend.at(t), row: float64(start.Row) + along.row*length*t + across.row*bend.at(t)}
	}
	source, mine := cellSet(lk.cells), map[int]bool{}
	var stations []station
	steps := int(math.Ceil(length * stationsPerCell))
	for s := 0; s <= steps; s++ {
		t := float64(s) / float64(steps)
		st := station{level: int8(int(lk.level) - int(math.Round(float64(drop)*t))), dir: directionAt(at, t, steps), along: length * t}
		ends := false
		for _, c := range crossSection(at(t), across, 1+min(2, int(3*t))) {
			i, ok := l.index(c)
			if !ok {
				ends = true
				continue
			}
			if source[i] || mine[i] {
				continue
			}
			switch l.meetsWater(i, st.level, source, mine) {
			case meetsHigher:
				return nil, false
			case meetsLower:
				ends = true
			}
			if isWater(l.ground[i]) {
				continue
			}
			st.cells = append(st.cells, i)
			mine[i] = true
		}
		if len(st.cells) > 0 {
			stations = append(stations, st)
		}
		if ends {
			break
		}
	}
	return stations, len(stations) > 0
}

// closestTo is the cell of the lake closest to a target.
func (l *land) closestTo(cells []int, target int) int {
	to := pointOf(l.cellAt(target))
	best, bestDistance := cells[0], math.Inf(1)
	for _, i := range cells {
		if d := distance(l.grid, pointOf(l.cellAt(i)), to); d < bestDistance {
			best, bestDistance = i, d
		}
	}
	return best
}

// crossSection are the cells across a river at a point: `width` of them, side by side along `across`.
func crossSection(at, across point, width int) []core.Cell {
	cells := make([]core.Cell, 0, width)
	for k := range width {
		o := float64(k) - float64(width-1)/2
		cells = append(cells, core.Cell{Col: int(math.Round(at.col + across.col*o)), Row: int(math.Round(at.row + across.row*o))})
	}
	return cells
}

// directionAt is the course's direction at step s of `steps`, from the previous station's point.
func directionAt(at func(float64) point, t float64, steps int) point {
	before := max(t-1/float64(steps), 0)
	if before == t {
		before, t = t, t+1/float64(steps)
	}
	from, to := at(before), at(t)
	d := point{col: to.col - from.col, row: to.row - from.row}
	length := math.Hypot(d.col, d.row)
	if length == 0 {
		return point{col: 1}
	}
	return point{col: d.col / length, row: d.row / length}
}

type waterMeeting int

const (
	meetsNone waterMeeting = iota
	// meetsLower: the cell touches water at the river's level or below: the river ends there, joining it.
	meetsLower
	// meetsHigher: water near the cell is too high for the river to flow next to it without a cliff.
	meetsHigher
)

// meetsWater looks at the water around a cell of the river, the river's own and its lake left out: water k steps away
// more than k levels from the river makes a cliff; water next to it, no higher, is where the river ends.
func (l *land) meetsWater(i int, level int8, source, mine map[int]bool) waterMeeting {
	meeting := meetsNone
	if isWater(l.ground[i]) {
		if l.height[i] > level {
			return meetsHigher
		}
		meeting = meetsLower
	}
	l.eachWithin(i, 2, func(j, steps int) {
		if source[j] || mine[j] || !isWater(l.ground[j]) {
			return
		}
		switch {
		case abs(int(l.height[j])-int(level)) > steps:
			meeting = meetsHigher
		case steps == 1 && l.height[j] <= level && meeting == meetsNone:
			meeting = meetsLower
		}
	})
	return meeting
}

// paintRiver floods the stations' cells, each at its level.
func (l *land) paintRiver(stations []station) {
	for _, st := range stations {
		for _, i := range st.cells {
			l.ground[i], l.height[i] = TileWater, st.level
		}
	}
}

// bridgePhase is how far along a river its first bridge goes: by the spawn for the spawn's river, else half the
// spacing in.
func (l *land) bridgePhase(lk lake, stations []station, o WaterOptions) float64 {
	if !lk.spawn {
		return float64(o.Bridges.Every) / 2
	}
	best, bestOffset := 0.0, math.Inf(1)
	for _, st := range stations {
		if d := axisDistance(float64(l.cellAt(st.cells[0]).Col), float64(o.Spawn.Near.Col), l.grid.Width, l.grid.Wraps); d < bestOffset {
			best, bestOffset = st.along, d
		}
	}
	return best
}

// wobble is a smooth sideways bend, zero at both ends of a course: three harmonics with random weights, scaled so the
// bend never goes past its amplitude.
type wobble struct {
	amplitude float64
	weights   [3]float64
}

func newWobble(amplitude float64, rng Rng) wobble {
	w := wobble{amplitude: amplitude}
	total := 0.0
	for k := range w.weights {
		w.weights[k] = rng.Float64()*2 - 1
		total += math.Abs(w.weights[k])
	}
	if total > 0 {
		for k := range w.weights {
			w.weights[k] /= total
		}
	}
	return w
}

func (w wobble) at(t float64) float64 {
	bend := 0.0
	for k, weight := range w.weights {
		bend += weight * math.Sin(float64(k+1)*math.Pi*t)
	}
	return w.amplitude * bend
}

func cellSet(cells []int) map[int]bool {
	set := make(map[int]bool, len(cells))
	for _, i := range cells {
		set[i] = true
	}
	return set
}

package domain

import (
	"math"

	core "donut/internal/core/domain"
)

// BridgeOptions are the bridges over the rivers (ADR 0008). Zero Every: none.
type BridgeOptions struct {
	// Every is how many cells of river between two bridges, about.
	Every int
	// MaxSpan is the most water cells a deck crosses.
	MaxSpan int
}

// bridgeAxisCos: a bridge goes where the river flows within 30° of a column or a row.
var bridgeAxisCos = math.Cos(30 * math.Pi / 180)

// buildBridges lays a bridge about every `Every` cells along each river, where the river runs along an axis: the deck
// crosses it along the other one, one level above the water, with the bank cell at each end at the deck's level. The
// end cells and the ones past them are kept free, so the bridge can always be walked onto.
func (l *land) buildBridges(rivers []river, o BridgeOptions) {
	if o.Every <= 0 {
		return
	}
	for _, r := range rivers {
		next := r.bridgePhase
		for _, st := range r.stations {
			if st.along >= next && l.buildBridge(st, o.MaxSpan) {
				next = st.along + float64(o.Every)
			}
		}
	}
	l.smoothCliffs()
}

// buildBridge lays a bridge at a station when the flow is along an axis, the deck crosses at most MaxSpan water cells
// all at one level, and both banks are grass that can stand at the deck's level.
func (l *land) buildBridge(st station, maxSpan int) bool {
	across, ok := deckDirection(st.dir)
	if !ok {
		return false
	}
	center := st.cells[len(st.cells)/2]
	deckLevel := int(l.height[center]) + 1
	if !isWater(l.ground[center]) || deckLevel > MaxHeight {
		return false
	}
	deck := []int{center}
	var ends, past []int
	for _, out := range []int{-1, 1} {
		c := l.cellAt(center)
		for d := 1; ; d++ {
			i, ok := l.index(core.Cell{Col: c.Col + across.Col*out*d, Row: c.Row + across.Row*out*d})
			if !ok {
				return false
			}
			if !isWater(l.ground[i]) {
				if !l.isGrass(i) || !l.canStand(i, deckLevel) {
					return false
				}
				ends = append(ends, i)
				if j, ok := l.index(core.Cell{Col: c.Col + across.Col*out*(d+1), Row: c.Row + across.Row*out*(d+1)}); ok {
					past = append(past, j)
				}
				break
			}
			if l.height[i] != l.height[center] || len(deck) >= maxSpan {
				return false
			}
			deck = append(deck, i)
		}
	}
	for _, i := range deck {
		l.ground[i], l.height[i], l.fixed[i] = TileBridge, int8(deckLevel), true
	}
	for _, i := range ends {
		l.height[i], l.kept[i], l.fixed[i] = int8(deckLevel), true, true
	}
	for _, i := range past {
		l.kept[i] = true
	}
	return true
}

// canStand: a bank cell can be set at the deck's level without a cliff to the water around it.
func (l *land) canStand(i, deckLevel int) bool {
	ok := true
	l.eachNeighbor(i, func(j int) { ok = ok && (!isWater(l.ground[j]) || abs(int(l.height[j])-deckLevel) <= MaxClimb) })
	return ok
}

// deckDirection is the axis a deck runs along: across the flow, when the flow is close enough to the other axis.
func deckDirection(flow point) (core.Cell, bool) {
	switch {
	case math.Abs(flow.col) >= bridgeAxisCos:
		return core.Cell{Row: 1}, true
	case math.Abs(flow.row) >= bridgeAxisCos:
		return core.Cell{Col: 1}, true
	}
	return core.Cell{}, false
}

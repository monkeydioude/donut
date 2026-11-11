package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
	"donut/internal/presence/domain"
)

// squareCity is a 10 x 10 city whose edge is the ring of cells one in from its border: 1 to 8 on each side.
func squareCity() (core.Grid, []core.Cell) {
	var edge []core.Cell
	for row := 1; row <= 8; row++ {
		for col := 1; col <= 8; col++ {
			if col == 1 || col == 8 || row == 1 || row == 8 {
				edge = append(edge, core.Cell{Col: col, Row: row})
			}
		}
	}
	return core.Grid{Width: 10, Height: 10}, edge
}

var torus = core.Grid{Width: 2048, Height: 1024, Wraps: true}

// ADR 0012 test 15: one appears in the city on the side they came from.
func TestArrivalInCity(t *testing.T) {
	city, edge := squareCity()
	dome := core.Dome{Back: core.Cell{Col: 100, Row: 100}, LeadsTo: "city"}
	cases := map[string]struct {
		from core.Cell
		side func(core.Cell) bool
	}{
		"from the east, the east edge":     {core.Cell{Col: 102, Row: 101}, func(c core.Cell) bool { return c.Col == 8 }},
		"from the west, the west edge":     {core.Cell{Col: 99, Row: 100}, func(c core.Cell) bool { return c.Col == 1 }},
		"from the north, the north edge":   {core.Cell{Col: 100, Row: 99}, func(c core.Cell) bool { return c.Row == 1 }},
		"from the south-east, that corner": {core.Cell{Col: 102, Row: 102}, func(c core.Cell) bool { return c == core.Cell{Col: 8, Row: 8} }},
		"from the north-west, that corner": {core.Cell{Col: 99, Row: 99}, func(c core.Cell) bool { return c == core.Cell{Col: 1, Row: 1} }},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			arrival, ok := domain.ArrivalInCity(edge, city, dome, torus, c.from)
			require.True(t, ok)
			assert.True(t, c.side(arrival), "arrived on %v", arrival)
		})
	}

	t.Run("across the world's seam", func(t *testing.T) {
		seam := core.Dome{Back: core.Cell{Col: 2046, Row: 100}, LeadsTo: "city"}
		arrival, ok := domain.ArrivalInCity(edge, city, seam, torus, core.Cell{Col: 0, Row: 101})
		require.True(t, ok)
		assert.Equal(t, 8, arrival.Col, "column 0 is east of column 2047")
	})

	t.Run("a city without an edge has no arrival", func(t *testing.T) {
		_, ok := domain.ArrivalInCity(nil, city, dome, torus, core.Cell{Col: 102, Row: 101})
		assert.False(t, ok)
	})
}

// ADR 0012 test 16: one appears beside the dome on the side they left by, walkable.
func TestArrivalBesideDome(t *testing.T) {
	city, _ := squareCity()
	dome := core.Dome{Back: core.Cell{Col: 100, Row: 100}, LeadsTo: "city"}
	everywhere := func(core.Cell) bool { return true }

	arrival, ok := domain.ArrivalBesideDome(torus, dome, everywhere, city, core.Cell{Col: 1, Row: 5})
	require.True(t, ok)
	assert.Equal(t, 99, arrival.Col, "the west edge gives the dome's west side")

	arrival, ok = domain.ArrivalBesideDome(torus, dome, everywhere, city, core.Cell{Col: 5, Row: 8})
	require.True(t, ok)
	assert.Equal(t, 102, arrival.Row, "the south edge gives the dome's south side")

	t.Run("that side's cell not walkable: the next closest by direction", func(t *testing.T) {
		blocked := core.Cell{Col: 99, Row: 101}
		walkable := func(c core.Cell) bool { return c != blocked }
		first, _ := domain.ArrivalBesideDome(torus, dome, everywhere, city, core.Cell{Col: 1, Row: 5})
		require.Equal(t, blocked, first)
		arrival, ok := domain.ArrivalBesideDome(torus, dome, walkable, city, core.Cell{Col: 1, Row: 5})
		require.True(t, ok)
		assert.NotEqual(t, blocked, arrival)
		assert.Equal(t, 99, arrival.Col, "still on the west")
	})

	t.Run("nothing walkable beside the dome: no arrival", func(t *testing.T) {
		_, ok := domain.ArrivalBesideDome(torus, dome, func(core.Cell) bool { return false }, city, core.Cell{Col: 1, Row: 5})
		assert.False(t, ok)
	})
}

// ADR 0012 test 16: one walks to the edge cell closest to the click, a click past the map included.
func TestClosestEdge(t *testing.T) {
	city, edge := squareCity()
	for clicked, want := range map[core.Cell]core.Cell{
		{Col: -3, Row: 5}:  {Col: 1, Row: 5},
		{Col: 5, Row: 40}:  {Col: 5, Row: 8},
		{Col: 9, Row: 9}:   {Col: 8, Row: 8},
		{Col: 0, Row: 0}:   {Col: 1, Row: 1},
		{Col: 50, Row: -9}: {Col: 8, Row: 1},
	} {
		got, ok := domain.ClosestEdge(edge, city, clicked)
		require.True(t, ok)
		assert.Equal(t, want, got, "a click on %v", clicked)
	}
	_, ok := domain.ClosestEdge(nil, city, core.Cell{})
	assert.False(t, ok)
}

func TestDoorRefusals(t *testing.T) {
	assert.EqualError(t, domain.NoDomeAt(core.Cell{Col: 5, Row: 7}), "There's no dome at (5, 7)")
	assert.EqualError(t, domain.ErrNotInCity, "You're not in a city")
	assert.EqualError(t, domain.StandsNowhere("Doma-1"), "Doma-1 stands nowhere on the world")
	assert.EqualError(t, domain.NoWayOut("Doma-1"), "There's no way out of Doma-1")
}

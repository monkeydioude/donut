package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	core "donut/internal/core/domain"
	"donut/internal/presence/domain"
)

// ADR 0010 test 1
func TestCheckTeleport(t *testing.T) {
	grid := core.Grid{Width: 64, Height: 32, Wraps: true}

	t.Run("another map is unknown", func(t *testing.T) {
		assert.EqualError(t, domain.CheckTeleport("world", grid, "mars", core.Cell{Col: 1, Row: 1}), "unknown map: mars")
	})

	t.Run("a cell outside the map, even on a wrapping one", func(t *testing.T) {
		assert.EqualError(t, domain.CheckTeleport("world", grid, "world", core.Cell{Col: 64, Row: 1}), "(64, 1) is outside the map")
		assert.EqualError(t, domain.CheckTeleport("world", grid, "world", core.Cell{Col: -1, Row: 2}), "(-1, 2) is outside the map")
	})

	t.Run("a cell of the map is fine", func(t *testing.T) {
		assert.NoError(t, domain.CheckTeleport("world", grid, "world", core.Cell{Col: 63, Row: 31}))
	})
}

package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
	"donut/internal/presence/domain"
	"donut/pkg/errd"
)

func TestParseGuestID(t *testing.T) {
	id, err := domain.ParseGuestID("6F9619FF-8B86-D011-B42D-00C04FC964FF")
	require.NoError(t, err)
	assert.Equal(t, "6f9619ff-8b86-d011-b42d-00c04fc964ff", id)

	for _, bad := range []string{"", "nope", "6f9619ff-8b86-d011-b42d"} {
		_, err := domain.ParseGuestID(bad)
		assert.ErrorIs(t, err, errd.ErrBadRequest, bad)
	}
}

func TestPublicID(t *testing.T) {
	guest := "6f9619ff-8b86-d011-b42d-00c04fc964ff"
	assert.Equal(t, domain.PublicID(guest), domain.PublicID(guest), "the same guest always has the same public id")
	assert.NotEqual(t, domain.PublicID(guest), domain.PublicID("7f9619ff-8b86-d011-b42d-00c04fc964ff"))
	assert.NotContains(t, domain.PublicID(guest), guest)
	assert.Len(t, domain.PublicID(guest), 32)
}

func TestCleanName(t *testing.T) {
	name, err := domain.CleanName("  Guest-1234 ")
	require.NoError(t, err)
	assert.Equal(t, "Guest-1234", name)

	name, err = domain.CleanName(strings.Repeat("é", 24))
	require.NoError(t, err)
	assert.Equal(t, strings.Repeat("é", 24), name, "24 characters, even when they take more bytes")

	for _, bad := range []string{"", "   ", strings.Repeat("a", 25), "bad\nname"} {
		_, err := domain.CleanName(bad)
		assert.ErrorIs(t, err, errd.ErrBadRequest, bad)
	}
}

func TestCheckStep(t *testing.T) {
	grid := core.Grid{Width: 160, Height: 160}
	from := core.Cell{Col: 2, Row: 2}
	everywhere := func(core.Cell) bool { return true }
	nowhere := func(core.Cell) bool { return false }

	for _, to := range []core.Cell{{Col: 1, Row: 1}, {Col: 3, Row: 2}, {Col: 2, Row: 3}, {Col: 3, Row: 1}} {
		assert.NoError(t, domain.CheckStep(grid, from, to, everywhere), "a neighbor: %v", to)
	}
	assert.ErrorIs(t, domain.CheckStep(grid, from, from, everywhere), domain.ErrNotNeighbor)
	assert.ErrorIs(t, domain.CheckStep(grid, from, core.Cell{Col: 4, Row: 2}, everywhere), domain.ErrNotNeighbor)
	assert.ErrorIs(t, domain.CheckStep(grid, from, core.Cell{Col: 3, Row: 2}, nowhere), domain.ErrNotWalkable)

	seam := core.Cell{Col: 159, Row: 2}
	assert.ErrorIs(t, domain.CheckStep(grid, seam, core.Cell{Col: 0, Row: 2}, everywhere), domain.ErrNotNeighbor, "a bounded map has no seam")
	grid.Wraps = true
	assert.NoError(t, domain.CheckStep(grid, seam, core.Cell{Col: 0, Row: 2}, everywhere), "a step across the seam")
}

func TestHeldChunks(t *testing.T) {
	grid := core.Grid{Width: 1024, Height: 512, Wraps: true}
	held := domain.HeldChunks{}

	got := held.Receive(grid, core.Cell{Col: 40, Row: 40})
	assert.Len(t, got, 9)
	assert.Equal(t, core.ChunkKey{CX: 1, CY: 1}, got[0], "the player's chunk first")
	assert.Empty(t, held.Receive(grid, core.Cell{Col: 41, Row: 40}), "nothing twice")
	assert.True(t, held.Sees(core.Cell{Col: 0, Row: 0}))

	assert.ElementsMatch(t, []core.ChunkKey{{CX: 3, CY: 0}, {CX: 3, CY: 1}, {CX: 3, CY: 2}}, held.Receive(grid, core.Cell{Col: 64, Row: 40}),
		"one chunk east: a new column of 3")
	assert.Empty(t, held.Forget(grid, core.Cell{Col: 64, Row: 40}), "chunk column 0 is only 2 away")
	assert.Equal(t, []core.ChunkKey{{CX: 0, CY: 0}, {CX: 0, CY: 1}, {CX: 0, CY: 2}}, held.Forget(grid, core.Cell{Col: 96, Row: 40}),
		"3 away: forgotten, row by row")
	assert.False(t, held.Sees(core.Cell{Col: 0, Row: 0}))

	far := domain.HeldChunks{}
	far.Receive(grid, core.Cell{Col: 0, Row: 0})
	assert.True(t, far.Sees(core.Cell{Col: 1023, Row: 511}), "across both seams")
}

func TestRequestLimiter(t *testing.T) {
	start := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	limiter := domain.NewRequestLimiter(start)
	for i := range domain.MaxRequestsPerSecond {
		assert.True(t, limiter.Allow(start), "request %d", i+1)
	}
	assert.False(t, limiter.Allow(start), "one too many in the same second")
	assert.True(t, limiter.Allow(start.Add(100*time.Millisecond)), "a tenth of a second gives one back")
	assert.False(t, limiter.Allow(start.Add(100*time.Millisecond)))
}

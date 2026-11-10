package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	core "donut/internal/core/domain"
)

func cell(col, row int) core.Cell { return core.Cell{Col: col, Row: row} }

var (
	torus   = core.Grid{Width: 64, Height: 64, Wraps: true}
	bounded = core.Grid{Width: 64, Height: 64}
	world   = core.Grid{Width: 1024, Height: 512, Wraps: true}
)

// ADR 0003 test 1
func TestNormalizeKeepsCellsWithinTheMap(t *testing.T) {
	for _, tc := range []struct{ in, want core.Cell }{
		{cell(-1, 0), cell(63, 0)},
		{cell(64, 65), cell(0, 1)},
		{cell(-65, -1), cell(63, 63)},
		{cell(5, 7), cell(5, 7)},
	} {
		got, ok := torus.Normalize(tc.in)
		assert.True(t, ok)
		assert.Equal(t, tc.want, got, "wrapping %v", tc.in)
	}

	_, ok := bounded.Normalize(cell(-1, 0))
	assert.False(t, ok, "a cell outside a bounded grid has no place in it")
	got, ok := bounded.Normalize(cell(64, 3))
	assert.False(t, ok)
	assert.Equal(t, cell(64, 3), got, "and isn't moved")
	assert.False(t, torus.Contains(cell(64, 0)), "Contains doesn't wrap: a cell is told within the map")
}

// ADR 0003 test 2
func TestAcrossTheSeam(t *testing.T) {
	assert.Equal(t, cell(1, 0), torus.Offset(cell(63, 0), cell(0, 0)), "one column east")
	assert.Equal(t, cell(-1, -1), torus.Offset(cell(0, 0), cell(63, 63)), "one cell up-left")
	assert.True(t, torus.AreNeighbors(cell(63, 5), cell(0, 6)))
	assert.False(t, torus.AreNeighbors(cell(63, 5), cell(63, 5)), "a cell isn't its own neighbor")

	assert.Equal(t, 1, world.ChunkDistance(core.ChunkKey{CX: 0, CY: 0}, core.ChunkKey{CX: 31, CY: 0}))
	assert.Equal(t, 1, world.ChunkDistance(core.ChunkKey{CX: 0, CY: 15}, core.ChunkKey{CX: 31, CY: 0}))
	assert.Equal(t, 16, world.ChunkDistance(core.ChunkKey{CX: 0, CY: 0}, core.ChunkKey{CX: 16, CY: 0}), "the far side")

	assert.Equal(t, cell(-63, 0), bounded.Offset(cell(63, 0), cell(0, 0)), "a bounded grid doesn't wrap")
	assert.False(t, bounded.AreNeighbors(cell(63, 5), cell(0, 5)))
	assert.Equal(t, 1, bounded.ChunkDistance(core.ChunkKey{CX: 0, CY: 0}, core.ChunkKey{CX: 1, CY: 1}))
}

// ADR 0003 test 3
func TestChunkOfACell(t *testing.T) {
	assert.Equal(t, core.ChunkKey{CX: 0, CY: 0}, core.ChunkOf(cell(31, 31)))
	assert.Equal(t, core.ChunkKey{CX: 1, CY: 0}, core.ChunkOf(cell(32, 0)))
	assert.Equal(t, core.ChunkKey{CX: -1, CY: 0}, core.ChunkOf(cell(-1, 5)), "rounds down")
	assert.Equal(t, cell(64, 32), core.ChunkKey{CX: 2, CY: 1}.Origin())
	assert.Equal(t, 32, world.ChunkColumns())
	assert.Equal(t, 16, world.ChunkRows())
	assert.Equal(t, 1, core.Grid{Width: 5, Height: 5}.ChunkColumns(), "a small map is one chunk")
}

func TestChunksAround(t *testing.T) {
	around := world.ChunksAround(core.ChunkKey{CX: 0, CY: 0}, 1)
	assert.Len(t, around, 9)
	assert.Equal(t, core.ChunkKey{CX: 0, CY: 0}, around[0], "the center first")
	assert.Contains(t, around, core.ChunkKey{CX: 31, CY: 15}, "across both seams")

	edge := core.Grid{Width: 96, Height: 96}.ChunksAround(core.ChunkKey{CX: 0, CY: 0}, 1)
	assert.ElementsMatch(t, []core.ChunkKey{{CX: 0, CY: 0}, {CX: 1, CY: 0}, {CX: 0, CY: 1}, {CX: 1, CY: 1}}, edge,
		"nothing past a bounded map's edges")

	small := core.Grid{Width: 64, Height: 64, Wraps: true}.ChunksAround(core.ChunkKey{CX: 0, CY: 0}, 1)
	assert.Len(t, small, 4, "a chunk never comes twice")
}

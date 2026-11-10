package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
	"donut/internal/worldmap/domain"
)

func TestWorldChunkContents(t *testing.T) {
	m := domain.Generate(domain.GenerateOptions{Width: 64, Height: 32, RegionNames: []string{"Only"}}, constantRng(0.5))
	m.SetGround(cell(33, 0), domain.TileVoid)
	m.Objects = []domain.Object{{ID: "r1", Kind: domain.ObjectRock, Col: 40, Row: 3}, {ID: "r2", Kind: domain.ObjectRock, Col: 2, Row: 2}}
	world := domain.NewWorld(m)

	t.Run("a chunk's ground as a short list of kinds and an index per cell, with its rocks", func(t *testing.T) {
		contents, ok := world.Chunk(core.ChunkKey{CX: 1})
		require.True(t, ok)
		assert.Equal(t, core.ChunkKey{CX: 1}, contents.Key)
		assert.Equal(t, 1, contents.Version)
		assert.Equal(t, []string{"grass", "void"}, contents.Kinds)
		require.Len(t, contents.Ground, core.CellsPerChunk)
		assert.Equal(t, uint8(0), contents.Ground[0])
		assert.Equal(t, uint8(1), contents.Ground[1], "(33, 0) is void")
		assert.Equal(t, make([]int8, core.CellsPerChunk), contents.Height)
		assert.Equal(t, uint8(1), contents.Region[0])
		assert.Equal(t, []core.PlacedObject{{ID: "r1", Kind: "rock", Cell: cell(40, 3)}}, contents.Objects, "only its own rocks")
	})

	t.Run("a chunk with no cells stored is all void", func(t *testing.T) {
		delete(m.Chunks, core.ChunkKey{CX: 1})
		contents, ok := domain.NewWorld(m).Chunk(core.ChunkKey{CX: 1})
		require.True(t, ok)
		assert.Equal(t, []string{"void"}, contents.Kinds)
		assert.Equal(t, make([]uint8, core.CellsPerChunk), contents.Ground)
	})

	t.Run("no chunk outside the map", func(t *testing.T) {
		_, ok := world.Chunk(core.ChunkKey{CX: 2})
		assert.False(t, ok)
	})
}

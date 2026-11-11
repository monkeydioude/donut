package infra_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
	"donut/internal/worldmap/domain"
	"donut/internal/worldmap/infra"
)

func someChunk() *domain.Chunk {
	chunk := domain.NewChunk(core.ChunkKey{CX: 3, CY: 1}, domain.TileGrass)
	chunk.Version = 4
	chunk.Ground[5] = domain.TileVoid
	chunk.Height[7], chunk.Height[8] = -2, 2
	chunk.Region[9] = 255
	return chunk
}

// ADR 0003 test 5
func TestChunkFormat(t *testing.T) {
	t.Run("a chunk encodes and decodes back to the same cells", func(t *testing.T) {
		data := infra.EncodeChunk(someChunk())
		assert.Len(t, data, 3073)
		assert.Equal(t, byte(1), data[0], "format 1")
		decoded, err := infra.DecodeChunk(core.ChunkKey{CX: 3, CY: 1}, 4, data)
		require.NoError(t, err)
		assert.Equal(t, someChunk(), decoded)
	})

	broken := map[string]func([]byte) []byte{
		"a wrong length":         func(data []byte) []byte { return data[:3072] },
		"no data":                func([]byte) []byte { return nil },
		"an unknown format":      func(data []byte) []byte { data[0] = 2; return data },
		"an unknown ground code": func(data []byte) []byte { data[1+10] = 10; return data },
	}
	for name, breakIt := range broken {
		t.Run("refuses "+name, func(t *testing.T) {
			_, err := infra.DecodeChunk(core.ChunkKey{}, 1, breakIt(infra.EncodeChunk(someChunk())))
			assert.Error(t, err)
		})
	}
}

package infra

import (
	"fmt"

	core "donut/internal/core/domain"
	"donut/internal/worldmap/domain"
)

const (
	// chunkFormat1 is the layout of a stored chunk (ADR 0003): this byte, then the 1,024 ground codes, the 1,024
	// heights (signed) and the 1,024 region ids, each row by row.
	chunkFormat1 = 1
	// chunkDataSize is how long a chunk is in format 1.
	chunkDataSize = 1 + 3*core.CellsPerChunk
)

// EncodeChunk is a chunk's cells in format 1.
func EncodeChunk(c *domain.Chunk) []byte {
	data := make([]byte, 0, chunkDataSize)
	data = append(data, chunkFormat1)
	for _, kind := range c.Ground {
		data = append(data, byte(kind))
	}
	for _, height := range c.Height {
		data = append(data, byte(height))
	}
	return append(data, c.Region[:]...)
}

// DecodeChunk reads a chunk's cells. It refuses another format, a wrong length and an unknown ground code.
func DecodeChunk(key core.ChunkKey, version int, data []byte) (*domain.Chunk, error) {
	if len(data) != chunkDataSize || data[0] != chunkFormat1 {
		return nil, fmt.Errorf("DecodeChunk %v: %d bytes in format %d, expected %d bytes in format %d", key, len(data), firstByte(data), chunkDataSize, chunkFormat1)
	}
	chunk := &domain.Chunk{Key: key, Version: version}
	ground, heights, regions := data[1:1+core.CellsPerChunk], data[1+core.CellsPerChunk:1+2*core.CellsPerChunk], data[1+2*core.CellsPerChunk:]
	for i := range core.CellsPerChunk {
		chunk.Ground[i] = domain.TileKind(ground[i])
		if !chunk.Ground[i].Known() {
			return nil, fmt.Errorf("DecodeChunk %v: unknown ground code %d at cell %d", key, ground[i], i)
		}
		chunk.Height[i] = int8(heights[i])
	}
	copy(chunk.Region[:], regions)
	return chunk, nil
}

func firstByte(data []byte) int {
	if len(data) == 0 {
		return -1
	}
	return int(data[0])
}

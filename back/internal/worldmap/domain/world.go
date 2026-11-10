package domain

import (
	core "donut/internal/core/domain"
)

// World is what a room needs of a map: where one can walk, and what players are sent of each chunk.
// It is built once when the room opens and never changes.
type World struct {
	Terrain
	chunks map[core.ChunkKey]*core.ChunkContents
}

func NewWorld(m Map) World {
	objects := objectsByChunk(m.Objects)
	chunks := make(map[core.ChunkKey]*core.ChunkContents, len(m.Chunks))
	for key, chunk := range m.Chunks {
		chunks[key] = chunk.contents(objects[key])
	}
	return World{Terrain: NewTerrain(m), chunks: chunks}
}

// Chunk is what players are sent of a chunk of the map: a chunk with no cells stored is all void.
// False for a chunk outside the map.
func (w World) Chunk(key core.ChunkKey) (*core.ChunkContents, bool) {
	if !w.grid.HasChunk(key) {
		return nil, false
	}
	if contents, ok := w.chunks[key]; ok {
		return contents, true
	}
	return NewChunk(key, TileVoid).contents(nil), true
}

func objectsByChunk(objects []Object) map[core.ChunkKey][]Object {
	byChunk := map[core.ChunkKey][]Object{}
	for _, object := range objects {
		key := core.ChunkOf(object.Cell())
		byChunk[key] = append(byChunk[key], object)
	}
	return byChunk
}

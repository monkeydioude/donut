package domain

import (
	"cmp"
	"slices"

	core "donut/internal/core/domain"
)

const (
	// HoldRadius: a player receives the chunks at most this many chunks from theirs: 3 x 3 (ADR 0003).
	HoldRadius = 1
	// ForgetRadius: a player forgets a chunk only once it is farther than this: outside the 5 x 5 around theirs.
	// The gap stops chunks from bouncing in and out when they walk back and forth across a border.
	ForgetRadius = 2
)

// HeldChunks are the chunks a connection holds: sent to it, and not forgotten yet. A player sees the players standing
// on them. Fog of war will filter what goes in.
type HeldChunks map[core.ChunkKey]bool

// Receive adds the chunks in range of a player on `at` that they don't hold yet, and gives them: theirs first.
func (h HeldChunks) Receive(grid core.Grid, at core.Cell) []core.ChunkKey {
	var received []core.ChunkKey
	for _, key := range grid.ChunksAround(core.ChunkOf(at), HoldRadius) {
		if !h[key] {
			h[key] = true
			received = append(received, key)
		}
	}
	return received
}

// Forget removes the chunks too far from a player on `at`, and gives them, row by row.
func (h HeldChunks) Forget(grid core.Grid, at core.Cell) []core.ChunkKey {
	center := core.ChunkOf(at)
	var far []core.ChunkKey
	for key := range h {
		if grid.ChunkDistance(center, key) > ForgetRadius {
			far = append(far, key)
		}
	}
	slices.SortFunc(far, func(a, b core.ChunkKey) int { return cmp.Or(cmp.Compare(a.CY, b.CY), cmp.Compare(a.CX, b.CX)) })
	for _, key := range far {
		delete(h, key)
	}
	return far
}

// Sees tells if a cell is on a held chunk.
func (h HeldChunks) Sees(c core.Cell) bool {
	return h[core.ChunkOf(c)]
}

package domain

// CellsPerChunk is how many cells a chunk holds.
const CellsPerChunk = ChunkSize * ChunkSize

// ChunkContents is what a player is sent of a chunk: its cells, and the objects standing on them.
// It is built once when a map is loaded and shared by every player: never change it.
type ChunkContents struct {
	Key     ChunkKey
	Version int
	// Kinds are the names of the ground kinds the chunk uses. Ground gives each cell's, row by row, as an index
	// into Kinds.
	Kinds  []string
	Ground []uint8
	// Height and Region are each cell's, row by row.
	Height  []int8
	Region  []uint8
	Objects []PlacedObject
}

// PlacedObject is an object standing on the map: a dome stands on its back cell (ADR 0012).
type PlacedObject struct {
	ID   string
	Kind string
	Cell Cell
	// LeadsTo is the map a dome leads to; empty for other kinds.
	LeadsTo string
}

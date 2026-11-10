package domain

import "fmt"

// ChunkSize is the side of a chunk, in cells: maps are stored and sent 32 x 32 cells at a time.
const ChunkSize = 32

// ChunkKey names a chunk: chunk (CX, CY) holds the cells from (32·CX, 32·CY) to (32·CX + 31, 32·CY + 31).
type ChunkKey struct {
	CX int
	CY int
}

// String is "[cx,cy]": short in logs, and not mistaken for a cell.
func (k ChunkKey) String() string {
	return fmt.Sprintf("[%d,%d]", k.CX, k.CY)
}

// Origin is the chunk's first cell, its top corner.
func (k ChunkKey) Origin() Cell {
	return Cell{Col: k.CX * ChunkSize, Row: k.CY * ChunkSize}
}

// ChunkOf is the chunk holding a cell.
func ChunkOf(c Cell) ChunkKey {
	return ChunkKey{CX: floorDiv(c.Col, ChunkSize), CY: floorDiv(c.Row, ChunkSize)}
}

// Grid is a map's shape: its size in cells, and whether it wraps around both ways (the surface of a donut)
// or has edges.
type Grid struct {
	Width  int
	Height int
	Wraps  bool
}

// Contains tells if a cell is within the map: 0 ≤ col < width and 0 ≤ row < height.
func (g Grid) Contains(c Cell) bool {
	return c.Col >= 0 && c.Row >= 0 && c.Col < g.Width && c.Row < g.Height
}

// Normalize gives the cell within the map. On a wrapping grid any cell has one, across the seams;
// on a bounded grid, a cell outside has none.
func (g Grid) Normalize(c Cell) (Cell, bool) {
	if !g.Wraps {
		return c, g.Contains(c)
	}
	return Cell{Col: mod(c.Col, g.Width), Row: mod(c.Row, g.Height)}, true
}

// Index is a cell's position in a row-by-row array of the map's cells. The cell must be within the map.
func (g Grid) Index(c Cell) int {
	return c.Row*g.Width + c.Col
}

// Offset is the shortest way from a to b, as a column and row difference. It goes across the seams on a
// wrapping grid: from (63, 0) to (0, 0) on a 64-wide map is one column east.
func (g Grid) Offset(a, b Cell) Cell {
	return Cell{Col: g.shortest(b.Col-a.Col, g.Width), Row: g.shortest(b.Row-a.Row, g.Height)}
}

// AreNeighbors: b is one of the 8 cells around a, across the seams on a wrapping grid.
func (g Grid) AreNeighbors(a, b Cell) bool {
	d := g.Offset(a, b)
	return max(abs(d.Col), abs(d.Row)) == 1
}

// ChunkColumns and ChunkRows are how many chunks the map has across and down.
func (g Grid) ChunkColumns() int { return ceilDiv(g.Width, ChunkSize) }
func (g Grid) ChunkRows() int    { return ceilDiv(g.Height, ChunkSize) }

// HasChunk tells if a chunk is part of the map.
func (g Grid) HasChunk(k ChunkKey) bool {
	return k.CX >= 0 && k.CY >= 0 && k.CX < g.ChunkColumns() && k.CY < g.ChunkRows()
}

// ChunkDistance is how many chunks apart two chunks are, diagonals counting as one, across the seams on a
// wrapping grid.
func (g Grid) ChunkDistance(a, b ChunkKey) int {
	dc := g.shortest(b.CX-a.CX, g.ChunkColumns())
	dr := g.shortest(b.CY-a.CY, g.ChunkRows())
	return max(abs(dc), abs(dr))
}

// ChunksAround is every chunk at most `radius` chunks from `center`, the center first, then row by row.
// It wraps on a wrapping grid, and leaves out what is past the edges of a bounded one. No chunk comes twice.
func (g Grid) ChunksAround(center ChunkKey, radius int) []ChunkKey {
	keys := []ChunkKey{center}
	seen := map[ChunkKey]bool{center: true}
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			key, ok := g.normalizeChunk(ChunkKey{CX: center.CX + dx, CY: center.CY + dy})
			if ok && !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	return keys
}

func (g Grid) normalizeChunk(k ChunkKey) (ChunkKey, bool) {
	if !g.Wraps {
		return k, g.HasChunk(k)
	}
	return ChunkKey{CX: mod(k.CX, g.ChunkColumns()), CY: mod(k.CY, g.ChunkRows())}, true
}

// shortest is the difference d made as short as can be on an axis of `size` that wraps: between −size/2 and
// size/2. It is d itself on a bounded grid.
func (g Grid) shortest(d, size int) int {
	if !g.Wraps {
		return d
	}
	d = mod(d, size)
	if d > size/2 {
		d -= size
	}
	return d
}

// mod is the remainder of a / b, never negative.
func mod(a, b int) int {
	return ((a % b) + b) % b
}

func floorDiv(a, b int) int {
	return (a - mod(a, b)) / b
}

func ceilDiv(a, b int) int {
	return (a + b - 1) / b
}

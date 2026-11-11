package domain

// DomeCells is a dome's footprint, in cells per side (ADR 0012).
const DomeCells = 2

// Dome is a dome city seen from the world map (ADR 0012): it covers DomeCells × DomeCells cells from its back cell,
// the lowest column and row, and leads to the city's own map.
type Dome struct {
	Back    Cell
	LeadsTo string
}

// Footprint is the cells the dome covers, row by row.
func (d Dome) Footprint() []Cell {
	cells := make([]Cell, 0, DomeCells*DomeCells)
	for dr := range DomeCells {
		for dc := range DomeCells {
			cells = append(cells, Cell{Col: d.Back.Col + dc, Row: d.Back.Row + dr})
		}
	}
	return cells
}

// Around is the cells touching the footprint (8 neighbors), 12 of them, row by row from the back: where one stands to
// go in. Cells past a bounded map's edges are left out.
func (d Dome) Around(grid Grid) []Cell {
	cells := []Cell{}
	for dr := -1; dr <= DomeCells; dr++ {
		for dc := -1; dc <= DomeCells; dc++ {
			inside := dr >= 0 && dr < DomeCells && dc >= 0 && dc < DomeCells
			if c, ok := grid.Normalize(Cell{Col: d.Back.Col + dc, Row: d.Back.Row + dr}); ok && !inside {
				cells = append(cells, c)
			}
		}
	}
	return cells
}

// Covers tells if a cell is one of the dome's, across the seams on a wrapping grid.
func (d Dome) Covers(grid Grid, c Cell) bool {
	offset := grid.Offset(d.Back, c)
	return offset.Col >= 0 && offset.Col < DomeCells && offset.Row >= 0 && offset.Row < DomeCells
}

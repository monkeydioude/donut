// Package domain holds what every module shares.
package domain

import "fmt"

// Cell is a position on a map's grid.
type Cell struct {
	Col int
	Row int
}

// String is "(col,row)": short in logs, paths included.
func (c Cell) String() string {
	return fmt.Sprintf("(%d,%d)", c.Col, c.Row)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

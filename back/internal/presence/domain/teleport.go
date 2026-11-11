package domain

import (
	"fmt"

	core "donut/internal/core/domain"
)

// Refusal is why a request wasn't done, in words for the player.
type Refusal struct {
	Reason string
}

func (r Refusal) Error() string {
	return r.Reason
}

// CheckTeleport tells why a player may not be teleported to a cell of a map (ADR 0010): it's outside the map. Nil
// when they may: the cell needs not be walkable. A map that doesn't exist is refused by the hub (ADR 0012).
func CheckTeleport(grid core.Grid, to core.Cell) error {
	if !grid.Contains(to) {
		return Refusal{Reason: fmt.Sprintf("(%d, %d) is outside the map", to.Col, to.Row)}
	}
	return nil
}

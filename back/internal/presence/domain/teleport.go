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

// CheckTeleport tells why a player of the room of `slug` may not be teleported to a cell of `toMap` (ADR 0010): it's
// another map, or the cell is outside this one. Nil when they may: the cell needs not be walkable.
func CheckTeleport(slug string, grid core.Grid, toMap string, to core.Cell) error {
	switch {
	case toMap != slug:
		return Refusal{Reason: fmt.Sprintf("unknown map: %s", toMap)}
	case !grid.Contains(to):
		return Refusal{Reason: fmt.Sprintf("(%d, %d) is outside the map", to.Col, to.Row)}
	}
	return nil
}

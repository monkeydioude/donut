package domain

import (
	"time"

	core "donut/internal/core/domain"
)

// Event is what a player's connection is told.
type Event interface {
	event()
}

// Welcome greets a player who joined: who they are, the players they see them included, and the time of a step.
type Welcome struct {
	You     string
	Players []Player
	Step    time.Duration
}

// Joined: another player came into view: they arrived on the map, or onto a chunk you hold, or a chunk they stand on
// became yours.
type Joined struct {
	Player Player
}

// Moved: a player, maybe you, stepped on a cell.
type Moved struct {
	ID   string
	Cell core.Cell
}

// Left: a player went out of view: they left the map, or the chunks you hold, or you forgot the chunk they stand on.
type Left struct {
	ID string
}

// ChunkShown: a chunk came into your range. Its contents are shared by every player: never change them.
type ChunkShown struct {
	Chunk *core.ChunkContents
}

// ChunkForgotten: a chunk went out of your range; drop it.
type ChunkForgotten struct {
	Key core.ChunkKey
}

// Planned answers a plan: the path from where you stand to To, empty when there is none.
type Planned struct {
	To   core.Cell
	Path []core.Cell
}

// Walking: the path the server walks you along, empty when there is nowhere to go.
type Walking struct {
	Path []core.Cell
}

// Stopped: your next step stopped being walkable; the walk ended on Cell.
type Stopped struct {
	Cell core.Cell
}

func (Welcome) event()        {}
func (Joined) event()         {}
func (Moved) event()          {}
func (Left) event()           {}
func (Planned) event()        {}
func (Walking) event()        {}
func (Stopped) event()        {}
func (ChunkShown) event()     {}
func (ChunkForgotten) event() {}

package domain

import (
	"time"

	core "donut/internal/core/domain"
)

// Event is what a player's connection is told.
type Event interface {
	event()
}

// Welcome greets a player who joined, or arrived from another map: the map, who they are, the players they see them
// included, and the time of a step (ADR 0012).
type Welcome struct {
	You     string
	Players []Player
	Step    time.Duration
	Map     core.MapInfo
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

// Planned answers a plan: the path from where you stand to To, empty when there is none, and what going there does
// (ADR 0012). For Enter or Exit from where one goes through at once, the path is the cell one stands on.
type Planned struct {
	To     core.Cell
	Path   []core.Cell
	Action Action
}

// Action is what a click on a cell does: move there, enter the dome there, or leave the city (ADR 0012).
type Action string

const (
	ActionMove  Action = "move"
	ActionEnter Action = "enter"
	ActionExit  Action = "exit"
)

// Walking: the path the server walks you along, empty when there is nowhere to go.
type Walking struct {
	Path []core.Cell
}

// Stopped: your next step stopped being walkable; the walk ended on Cell.
type Stopped struct {
	Cell core.Cell
}

// Teleported answers a teleport (ADR 0010): you stand on Cell now, walkable or not.
type Teleported struct {
	Cell     core.Cell
	Walkable bool
}

// Refused answers a request that wasn't done (ADR 0010): which one, and why, in words for the player.
type Refused struct {
	Request string
	Reason  string
}

func (Welcome) event()        {}
func (Joined) event()         {}
func (Moved) event()          {}
func (Left) event()           {}
func (Planned) event()        {}
func (Walking) event()        {}
func (Stopped) event()        {}
func (Teleported) event()     {}
func (Refused) event()        {}
func (ChunkShown) event()     {}
func (ChunkForgotten) event() {}

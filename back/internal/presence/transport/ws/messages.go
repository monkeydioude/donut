package wstransport

import (
	"encoding/json"
	"strconv"

	core "donut/internal/core/domain"
	"donut/internal/presence/domain"
)

// The JSON messages of the websocket (ADR 0001, ADR 0002, ADR 0003), each with a "type".

type cellOut struct {
	Col int `json:"col"`
	Row int `json:"row"`
}

type playerOut struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Robot string `json:"robot"`
	Col   int    `json:"col"`
	Row   int    `json:"row"`
}

type welcomeOut struct {
	Type    string      `json:"type"`
	You     string      `json:"you"`
	Players []playerOut `json:"players"`
	StepMs  int64       `json:"stepMs"`
}

type joinedOut struct {
	Type   string    `json:"type"`
	Player playerOut `json:"player"`
}

type movedOut struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Col  int    `json:"col"`
	Row  int    `json:"row"`
}

type leftOut struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type plannedOut struct {
	Type string    `json:"type"`
	Col  int       `json:"col"`
	Row  int       `json:"row"`
	Path []cellOut `json:"path"`
}

type walkingOut struct {
	Type string    `json:"type"`
	Path []cellOut `json:"path"`
}

type stoppedOut struct {
	Type string `json:"type"`
	Col  int    `json:"col"`
	Row  int    `json:"row"`
}

// teleportedOut answers a teleport (ADR 0010): where the player stands now, and whether they can walk from there.
type teleportedOut struct {
	Type     string `json:"type"`
	Col      int    `json:"col"`
	Row      int    `json:"row"`
	Walkable bool   `json:"walkable"`
}

// refusedOut tells a request wasn't done, and why (ADR 0010).
type refusedOut struct {
	Type    string `json:"type"`
	Request string `json:"request"`
	Reason  string `json:"reason"`
}

// chunkOut is a chunk coming into the player's range: its ground as indices into kinds, its heights and regions,
// each cell row by row, and the objects on it.
type chunkOut struct {
	Type    string            `json:"type"`
	CX      int               `json:"cx"`
	CY      int               `json:"cy"`
	Version int               `json:"version"`
	Kinds   []string          `json:"kinds"`
	Ground  numberList[uint8] `json:"ground"`
	Height  numberList[int8]  `json:"height"`
	Region  numberList[uint8] `json:"region"`
	Objects []objectOut       `json:"objects"`
}

type objectOut struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Col  int    `json:"col"`
	Row  int    `json:"row"`
}

type forgetOut struct {
	Type string `json:"type"`
	CX   int    `json:"cx"`
	CY   int    `json:"cy"`
}

// numberList is written as a JSON array of numbers. Go would write a []uint8 as a base64 string.
type numberList[T uint8 | int8] []T

func (l numberList[T]) MarshalJSON() ([]byte, error) {
	out := make([]byte, 0, 2+3*len(l))
	out = append(out, '[')
	for i, n := range l {
		if i > 0 {
			out = append(out, ',')
		}
		out = strconv.AppendInt(out, int64(n), 10)
	}
	return append(out, ']'), nil
}

func toChunkOut(c *core.ChunkContents) chunkOut {
	objects := make([]objectOut, len(c.Objects))
	for i, o := range c.Objects {
		objects[i] = objectOut{ID: o.ID, Kind: o.Kind, Col: o.Cell.Col, Row: o.Cell.Row}
	}
	return chunkOut{Type: "chunk", CX: c.Key.CX, CY: c.Key.CY, Version: c.Version, Kinds: c.Kinds,
		Ground: c.Ground, Height: c.Height, Region: c.Region, Objects: objects}
}

func toPlayerOut(p domain.Player) playerOut {
	return playerOut{ID: p.ID, Name: p.Name, Robot: p.Robot, Col: p.Cell.Col, Row: p.Cell.Row}
}

// toCellsOut never gives null: an empty path is [].
func toCellsOut(cells []core.Cell) []cellOut {
	out := make([]cellOut, len(cells))
	for i, c := range cells {
		out[i] = cellOut{Col: c.Col, Row: c.Row}
	}
	return out
}

// encodeEvent is the JSON message of an event.
func encodeEvent(event domain.Event) any {
	switch e := event.(type) {
	case domain.Welcome:
		players := make([]playerOut, len(e.Players))
		for i, p := range e.Players {
			players[i] = toPlayerOut(p)
		}
		return welcomeOut{Type: "welcome", You: e.You, Players: players, StepMs: e.Step.Milliseconds()}
	case domain.Joined:
		return joinedOut{Type: "joined", Player: toPlayerOut(e.Player)}
	case domain.Moved:
		return movedOut{Type: "moved", ID: e.ID, Col: e.Cell.Col, Row: e.Cell.Row}
	case domain.Left:
		return leftOut{Type: "left", ID: e.ID}
	case domain.Planned:
		return plannedOut{Type: "planned", Col: e.To.Col, Row: e.To.Row, Path: toCellsOut(e.Path)}
	case domain.Walking:
		return walkingOut{Type: "walking", Path: toCellsOut(e.Path)}
	case domain.Stopped:
		return stoppedOut{Type: "stopped", Col: e.Cell.Col, Row: e.Cell.Row}
	case domain.Teleported:
		return teleportedOut{Type: "teleported", Col: e.Cell.Col, Row: e.Cell.Row, Walkable: e.Walkable}
	case domain.Refused:
		return refusedOut{Type: "refused", Request: e.Request, Reason: e.Reason}
	case domain.ChunkShown:
		return toChunkOut(e.Chunk)
	case domain.ChunkForgotten:
		return forgetOut{Type: "forget", CX: e.Key.CX, CY: e.Key.CY}
	}
	return nil
}

// logged is what the logs show of a message: a chunk by its key and size, never in full.
func logged(event domain.Event, message any) any {
	if e, ok := event.(domain.ChunkShown); ok {
		return map[string]any{"type": "chunk", "chunk": e.Chunk.Key.String(), "version": e.Chunk.Version, "objects": len(e.Chunk.Objects)}
	}
	return message
}

// request is what a client asks: "plan" or "walk" toward a cell, or "teleport" to a cell of a map (ADR 0010).
type request struct {
	kind  string
	to    core.Cell
	toMap string
}

type requestIn struct {
	Type string  `json:"type"`
	Map  *string `json:"map"`
	Col  *int    `json:"col"`
	Row  *int    `json:"row"`
}

// decodeRequest reads a plan, walk or teleport message. Anything else, bad JSON, a missing field or a cell that
// isn't a whole number is not a request.
func decodeRequest(data []byte) (request, bool) {
	var msg requestIn
	if err := json.Unmarshal(data, &msg); err != nil || msg.Col == nil || msg.Row == nil {
		return request{}, false
	}
	req := request{kind: msg.Type, to: core.Cell{Col: *msg.Col, Row: *msg.Row}}
	switch msg.Type {
	case "plan", "walk":
		return req, true
	case "teleport":
		if msg.Map == nil {
			return request{}, false
		}
		req.toMap = *msg.Map
		return req, true
	}
	return request{}, false
}

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

// welcomeOut greets a player on a map: the map, as GET /api/maps/{slug} describes it (ADR 0012), who they are, who
// they see, and the time of a step.
type welcomeOut struct {
	Type    string      `json:"type"`
	You     string      `json:"you"`
	Players []playerOut `json:"players"`
	StepMs  int64       `json:"stepMs"`
	Map     mapOut      `json:"map"`
}

// mapOut is a map's description, the same as the HTTP API's.
type mapOut struct {
	ID        string      `json:"id"`
	Slug      string      `json:"slug"`
	Name      string      `json:"name"`
	Width     int         `json:"width"`
	Height    int         `json:"height"`
	Wraps     bool        `json:"wraps"`
	ChunkSize int         `json:"chunkSize"`
	Regions   []regionOut `json:"regions"`
	StandsOn  *placeOut   `json:"standsOn"`
}

type regionOut struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Biome string `json:"biome"`
}

type placeOut struct {
	Map    string `json:"map"`
	Col    int    `json:"col"`
	Row    int    `json:"row"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
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

// plannedOut answers a plan: the path, and what going there does: "move", "enter" or "exit" (ADR 0012).
type plannedOut struct {
	Type   string    `json:"type"`
	Col    int       `json:"col"`
	Row    int       `json:"row"`
	Path   []cellOut `json:"path"`
	Action string    `json:"action"`
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

// objectOut is an object of a chunk: a dome also tells the map it leads to (ADR 0012).
type objectOut struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Col     int    `json:"col"`
	Row     int    `json:"row"`
	LeadsTo string `json:"leadsTo,omitempty"`
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
		objects[i] = objectOut{ID: o.ID, Kind: o.Kind, Col: o.Cell.Col, Row: o.Cell.Row, LeadsTo: o.LeadsTo}
	}
	return chunkOut{Type: "chunk", CX: c.Key.CX, CY: c.Key.CY, Version: c.Version, Kinds: c.Kinds,
		Ground: c.Ground, Height: c.Height, Region: c.Region, Objects: objects}
}

func toPlayerOut(p domain.Player) playerOut {
	return playerOut{ID: p.ID, Name: p.Name, Robot: p.Robot, Col: p.Cell.Col, Row: p.Cell.Row}
}

// toMapOut never gives null regions: none is [].
func toMapOut(m core.MapInfo) mapOut {
	regions := make([]regionOut, len(m.Regions))
	for i, r := range m.Regions {
		regions[i] = regionOut{ID: r.ID, Name: r.Name, Biome: r.Biome}
	}
	out := mapOut{ID: m.ID, Slug: m.Slug, Name: m.Name, Width: m.Width, Height: m.Height, Wraps: m.Wraps, ChunkSize: core.ChunkSize, Regions: regions}
	if p := m.StandsOn; p != nil {
		out.StandsOn = &placeOut{Map: p.Map, Col: p.Col, Row: p.Row, Width: p.Width, Height: p.Height}
	}
	return out
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
		return welcomeOut{Type: "welcome", You: e.You, Players: players, StepMs: e.Step.Milliseconds(), Map: toMapOut(e.Map)}
	case domain.Joined:
		return joinedOut{Type: "joined", Player: toPlayerOut(e.Player)}
	case domain.Moved:
		return movedOut{Type: "moved", ID: e.ID, Col: e.Cell.Col, Row: e.Cell.Row}
	case domain.Left:
		return leftOut{Type: "left", ID: e.ID}
	case domain.Planned:
		return plannedOut{Type: "planned", Col: e.To.Col, Row: e.To.Row, Path: toCellsOut(e.Path), Action: string(e.Action)}
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

// request is what a client asks: "plan" or "walk" toward a cell, "teleport" to a cell of a map (ADR 0010), "enter"
// the dome on a cell or "exit" the city by a click on a cell (ADR 0012).
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

// decodeRequest reads a plan, walk, teleport, enter or exit message. Anything else, bad JSON, a missing field or a
// cell that isn't a whole number is not a request.
func decodeRequest(data []byte) (request, bool) {
	var msg requestIn
	if err := json.Unmarshal(data, &msg); err != nil || msg.Col == nil || msg.Row == nil {
		return request{}, false
	}
	req := request{kind: msg.Type, to: core.Cell{Col: *msg.Col, Row: *msg.Row}}
	switch msg.Type {
	case "plan", "walk", "enter", "exit":
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

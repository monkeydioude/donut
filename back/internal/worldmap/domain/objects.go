package domain

import (
	"regexp"

	core "donut/internal/core/domain"
)

// slugPattern is a map's slug a dome may lead to: lowercase words of letters and digits joined by dashes, as "doma-1".
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// maxSlugLength bounds a slug a dome leads to.
const maxSlugLength = 64

// ObjectKind is what stands on the map.
type ObjectKind string

const (
	ObjectRock ObjectKind = "rock"
	// ObjectDome is a dome city seen from the world map (ADR 0012): it covers core.DomeCells × core.DomeCells cells,
	// and leads to the city's own map.
	ObjectDome ObjectKind = "dome"
)

func (k ObjectKind) Known() bool {
	return k == ObjectRock || k == ObjectDome
}

// Object stands on the map. A rock stands on its cell; a dome on its back cell, the lowest column and row of its
// footprint. Its ID is empty until saved.
type Object struct {
	ID   string     `db:"id"`
	Kind ObjectKind `db:"kind"`
	Col  int        `db:"col"`
	Row  int        `db:"row"`
	// LeadsTo is the slug of the map a dome leads to; empty for a rock.
	LeadsTo string `db:"leads_to"`
}

// Cell is the cell the object stands on: a dome's back cell.
func (o Object) Cell() core.Cell {
	return core.Cell{Col: o.Col, Row: o.Row}
}

// Footprint is every cell the object covers: its cell for a rock, the 2 × 2 from its back cell for a dome.
func (o Object) Footprint() []core.Cell {
	if o.Kind != ObjectDome {
		return []core.Cell{o.Cell()}
	}
	return o.Dome().Footprint()
}

// Dome is a dome object's footprint and where it leads.
func (o Object) Dome() core.Dome {
	return core.Dome{Back: o.Cell(), LeadsTo: o.LeadsTo}
}

// validateObjects: each object is of a known kind, within the map, on grass, and no two cover the same cell. A dome
// also stays within one chunk, on one height, and leads to a map no other dome leads to.
func (m Map) validateObjects() error {
	taken := make(map[core.Cell]bool, len(m.Objects))
	leadsTo := map[string]bool{}
	for _, object := range m.Objects {
		if err := m.validateObject(object, taken); err != nil {
			return err
		}
		if object.Kind != ObjectDome {
			continue
		}
		if leadsTo[object.LeadsTo] {
			return invalid("two domes lead to %q", object.LeadsTo)
		}
		leadsTo[object.LeadsTo] = true
	}
	return nil
}

func (m Map) validateObject(object Object, taken map[core.Cell]bool) error {
	cell, grid := object.Cell(), m.Grid()
	switch {
	case !object.Kind.Known():
		return invalid("unknown object kind %q", object.Kind)
	case object.Kind == ObjectDome && object.LeadsTo == "":
		return invalid("dome at %v leads nowhere", cell)
	case object.Kind == ObjectDome && (len(object.LeadsTo) > maxSlugLength || !slugPattern.MatchString(object.LeadsTo)):
		return invalid("dome at %v leads to %q, which isn't a map's slug", cell, object.LeadsTo)
	case object.Kind != ObjectDome && object.LeadsTo != "":
		return invalid("%s at %v can't lead to a map", object.Kind, cell)
	}
	footprint := object.Footprint()
	for _, c := range footprint {
		switch {
		case !grid.Contains(c):
			return invalid("%s at %v: %v is outside the map", object.Kind, cell, c)
		case core.ChunkOf(c) != core.ChunkOf(cell):
			return invalid("%s at %v crosses the border of chunk %v", object.Kind, cell, core.ChunkOf(cell))
		case m.GroundAt(c) != TileGrass:
			return invalid("%s at %v: %v is %v, not grass", object.Kind, cell, c, m.GroundAt(c))
		case m.HeightAt(c) != m.HeightAt(cell):
			return invalid("%s at %v stands on two heights", object.Kind, cell)
		case taken[c]:
			return invalid("two objects on %v", c)
		}
	}
	for _, c := range footprint {
		taken[c] = true
	}
	return nil
}

// Around is the cells touching a dome's footprint (8 neighbors): where one stands to go in (ADR 0012).
func (o Object) Around(grid core.Grid) []core.Cell {
	return o.Dome().Around(grid)
}

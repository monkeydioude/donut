package domain_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
	"donut/internal/worldmap/domain"
	"donut/pkg/errd"
)

func dome(col, row int, leadsTo string) domain.Object {
	return domain.Object{Kind: domain.ObjectDome, Col: col, Row: row, LeadsTo: leadsTo}
}

// ADR 0012 test 4
func TestADomesRules(t *testing.T) {
	valid := func() domain.Map {
		m := domain.Generate(options(0), constantRng(0.5))
		m.Objects = []domain.Object{dome(4, 4, "city")}
		return m
	}

	t.Run("a dome on 4 grass cells at one height is accepted", func(t *testing.T) {
		assert.NoError(t, valid().Validate())
	})

	t.Run("its 4 cells aren't walkable, the 12 around them are its sides", func(t *testing.T) {
		m := valid()
		terrain := domain.NewTerrain(m)
		for _, c := range []core.Cell{cell(4, 4), cell(5, 4), cell(4, 5), cell(5, 5)} {
			assert.False(t, terrain.IsWalkable(c), "%v is under the dome", c)
		}
		around := m.Objects[0].Around(m.Grid())
		assert.Len(t, around, 12)
		for _, c := range around {
			assert.True(t, terrain.IsWalkable(c), "%v is beside the dome", c)
			offset := m.Grid().Offset(cell(4, 4), c)
			assert.True(t, offset.Col >= -1 && offset.Col <= 2 && offset.Row >= -1 && offset.Row <= 2, "%v touches the footprint", c)
		}
	})

	cases := map[string]func(m *domain.Map){
		"on a rock": func(m *domain.Map) {
			m.Objects = append(m.Objects, domain.Object{Kind: domain.ObjectRock, Col: 5, Row: 5})
		},
		"on water":              func(m *domain.Map) { m.SetGround(cell(5, 4), domain.TileWater) },
		"on two heights":        func(m *domain.Map) { m.SetHeight(cell(4, 5), 1) },
		"across a chunk border": func(m *domain.Map) { m.Objects[0].Col = 31 },
		"past the map's edge":   func(m *domain.Map) { m.Objects[0].Row = 31 },
		"leading nowhere":       func(m *domain.Map) { m.Objects[0].LeadsTo = "" },
		"leading to a path":     func(m *domain.Map) { m.Objects[0].LeadsTo = "../city" },
		"leading to a dot":      func(m *domain.Map) { m.Objects[0].LeadsTo = "." },
		"leading to capitals":   func(m *domain.Map) { m.Objects[0].LeadsTo = "City" },
		"two domes to one map":  func(m *domain.Map) { m.Objects = append(m.Objects, dome(10, 10, "city")) },
		"a rock leading somewhere": func(m *domain.Map) {
			m.Objects = append(m.Objects, domain.Object{Kind: domain.ObjectRock, Col: 9, Row: 9, LeadsTo: "city"})
		},
		"two domes sharing a cell": func(m *domain.Map) { m.Objects = append(m.Objects, dome(5, 5, "town")) },
	}
	for name, breakIt := range cases {
		t.Run("refuses a dome "+name, func(t *testing.T) {
			m := valid()
			breakIt(&m)
			assert.ErrorIs(t, m.Validate(), errd.ErrBadRequest)
		})
	}

	t.Run("two domes to two maps are accepted", func(t *testing.T) {
		m := valid()
		m.Objects = append(m.Objects, dome(10, 10, "town"))
		assert.NoError(t, m.Validate())
	})
}

// ADR 0012 test 5: on flat grass, the dome stands on the cell closest to the center, moved back into its chunk.
func TestTheGeneratorPutsTheDomeOnTheSpawn(t *testing.T) {
	opts := options(0.3)
	opts.Dome = domain.DomeOptions{LeadsTo: "city"}

	m := domain.Generate(opts, seeded(1))

	domes := domesOf(m)
	require.Len(t, domes, 1)
	assert.Equal(t, dome(30, 15, "city"), domes[0], "(31, 15) is closest to the center, on the last column of chunk 0")
	require.NoError(t, m.Validate())
	assertClearedAround(t, m, domes[0])

	t.Run("without dome options, no dome", func(t *testing.T) {
		assert.Empty(t, domesOf(domain.Generate(options(0.3), seeded(1))))
	})
}

// ADR 0012 test 5, on the starter world
func TestTheStarterWorldHasDoma1OnItsSpawn(t *testing.T) {
	m := starter()
	domes := domesOf(m)
	require.Len(t, domes, 1)
	assert.Equal(t, "doma-1", domes[0].LeadsTo)
	middle := core.Cell{Col: domes[0].Col + 1, Row: domes[0].Row + 1}
	fromCenter := math.Hypot(float64(middle.Col)-float64(m.Width)/2, float64(middle.Row)-float64(m.Height)/2)
	assert.Less(t, fromCenter, 4.0, "the dome stands where the spawn was, by the center")
	assertClearedAround(t, m, domes[0])
}

// assertClearedAround: the dome and the ring of cells around it are grass at one height, and nothing stands on the
// ring: one can walk all round it.
func assertClearedAround(t *testing.T, m domain.Map, d domain.Object) {
	t.Helper()
	level := m.HeightAt(d.Cell())
	terrain := domain.NewTerrain(m)
	for _, c := range append(d.Footprint(), d.Around(m.Grid())...) {
		assert.Equal(t, domain.TileGrass, m.GroundAt(c), "%v is grass", c)
		assert.Equal(t, level, m.HeightAt(c), "%v is at the dome's height", c)
	}
	for _, c := range d.Around(m.Grid()) {
		assert.True(t, terrain.IsWalkable(c), "%v beside the dome is walkable", c)
	}
}

func domesOf(m domain.Map) []domain.Object {
	var domes []domain.Object
	for _, o := range m.Objects {
		if o.Kind == domain.ObjectDome {
			domes = append(domes, o)
		}
	}
	return domes
}

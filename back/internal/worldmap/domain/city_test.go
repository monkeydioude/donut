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

// bounded is a flat grass map that doesn't wrap, of any size.
func bounded(width, height int) domain.Map {
	return domain.Generate(domain.GenerateOptions{Slug: "b", Name: "B", Width: width, Height: height}, constantRng(0.5))
}

// ADR 0012 test 1
func TestMapSizes(t *testing.T) {
	for _, size := range [][2]int{{16, 48}, {32, 32}, {4096, 16}} {
		assert.NoError(t, bounded(size[0], size[1]).Validate(), "a bounded %dx%d", size[0], size[1])
	}
	for _, size := range [][2]int{{24, 32}, {32, 24}, {4112, 16}} {
		m := bounded(32, 32)
		m.Width, m.Height = size[0], size[1]
		assert.ErrorIs(t, m.Validate(), errd.ErrBadRequest, "a bounded %dx%d", size[0], size[1])
	}
	wrapping := func(side int) domain.Map {
		return domain.Generate(domain.GenerateOptions{Width: side, Height: side, Wraps: true}, constantRng(0.5))
	}
	assert.NoError(t, wrapping(160).Validate())
	assert.NoError(t, wrapping(192).Validate())
	m := wrapping(160)
	m.Width = 176
	assert.ErrorContains(t, m.Validate(), "multiple of 32", "a wrapping map's side is a multiple of a chunk")
}

// ADR 0012 test 2
func TestPartialChunks(t *testing.T) {
	m := bounded(48, 48)
	require.NoError(t, m.Validate())
	chunk := m.Chunks[core.ChunkKey{CX: 1, CY: 1}]
	require.NotNil(t, chunk)
	assert.Equal(t, domain.TileGrass, m.GroundAt(cell(47, 47)), "the map's last cell")
	world := domain.NewWorld(m)
	contents, ok := world.Chunk(core.ChunkKey{CX: 1, CY: 1})
	require.True(t, ok)
	assert.Equal(t, "void", contents.Kinds[contents.Ground[0*32+20]], "(52, 32) is past the edge")
	assert.Equal(t, "grass", contents.Kinds[contents.Ground[15*32+15]], "(47, 47) is in")
	assert.False(t, world.IsWalkable(cell(48, 40)))

	m.SetGround(cell(50, 40), domain.TileGrass)
	assert.ErrorContains(t, m.Validate(), "past the map's edge, it must be void")
}

// ADR 0012 test 3
func TestFloorsAndWalls(t *testing.T) {
	m := bounded(32, 32)
	m.SetGround(cell(1, 1), domain.TileFloor)
	m.SetGround(cell(2, 1), domain.TileWall)
	m.SetHeight(cell(2, 1), 3)
	m.SetHeight(cell(1, 1), -3)
	require.NoError(t, m.Validate())
	terrain := domain.NewTerrain(m)
	assert.True(t, terrain.IsWalkable(cell(1, 1)), "floor")
	assert.False(t, terrain.IsWalkable(cell(2, 1)), "wall")

	m.Objects = []domain.Object{{Kind: domain.ObjectRock, Col: 1, Row: 1}}
	assert.ErrorContains(t, m.Validate(), "not grass", "a rock on floor")
}

// ADR 0012 test 6
func TestDoma1sLayout(t *testing.T) {
	m := domain.GenerateCity(domain.Doma1())
	require.NoError(t, m.Validate())
	assert.Equal(t, domain.Summary{Slug: "doma-1", Name: "Doma-1", Width: 32, Height: 32}, m.Summary)
	assert.Equal(t, []domain.Region{{ID: 1, Name: "Doma-1", Biome: domain.BiomeDome}}, m.Regions)
	fromMiddle := func(c core.Cell) float64 { return math.Hypot(float64(c.Col)+0.5-16, float64(c.Row)+0.5-16) }

	eachCell(m, func(c core.Cell) {
		kind, d := m.GroundAt(c), fromMiddle(c)
		inPark := c.Col >= 13 && c.Col <= 18 && c.Row >= 13 && c.Row <= 18
		switch {
		case inPark:
			require.Equal(t, domain.TileGrass, kind, "%v is the park", c)
		case d <= 14.5:
			require.Equal(t, domain.TileFloor, kind, "%v is floor", c)
		case d <= 15.5:
			require.Equal(t, domain.TileWall, kind, "%v is wall", c)
		default:
			require.Equal(t, domain.TileVoid, kind, "%v is void", c)
		}
		if kind == domain.TileVoid {
			require.Zero(t, m.RegionAt(c), "%v has no region", c)
		} else {
			require.Equal(t, uint8(1), m.RegionAt(c), "%v is in Doma-1", c)
		}
	})

	t.Run("the wall ring is closed: no floor touches the void by a side", func(t *testing.T) {
		eachCell(m, func(c core.Cell) {
			if !m.GroundAt(c).Walkable() {
				return
			}
			for _, d := range []core.Cell{{Col: 1}, {Col: -1}, {Row: 1}, {Row: -1}} {
				next := core.Cell{Col: c.Col + d.Col, Row: c.Row + d.Row}
				require.NotEqual(t, domain.TileVoid, m.GroundAt(next), "%v touches the void at %v", c, next)
			}
		})
	})

	t.Run("the wall is a cutaway: +3 at the back, +1 at the front, +2 at the sides", func(t *testing.T) {
		for c, height := range map[core.Cell]int8{cell(15, 1): 3, cell(1, 15): 3, cell(16, 30): 1, cell(30, 16): 1, cell(26, 5): 2, cell(5, 26): 2} {
			require.Equal(t, domain.TileWall, m.GroundAt(c), "%v is on the ring", c)
			assert.Equal(t, height, m.HeightAt(c), "%v, col + row = %d", c, c.Col+c.Row)
		}
		assert.Zero(t, m.HeightAt(cell(16, 16)), "the floor and park")
	})

	t.Run("the spawn is in the park, and every walkable cell is reached from it", func(t *testing.T) {
		terrain := domain.NewTerrain(m)
		spawn, err := terrain.SpawnPoint()
		require.NoError(t, err)
		assert.Equal(t, cell(15, 15), spawn, "the park's middle, ties to the lowest row then column")
		eachCell(m, func(c core.Cell) {
			if terrain.IsWalkable(c) && c != spawn {
				require.NotEmpty(t, terrain.Path(spawn, c), "%v is reached", c)
			}
		})
	})
}

package domain_test

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
	"donut/internal/worldmap/domain"
	"donut/pkg/errd"
)

func seeded(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed))
}

// constantRng always gives the same float.
type constantRng float64

func (r constantRng) Float64() float64 { return float64(r) }

// sequenceRng gives its floats in order, then its last one forever.
type sequenceRng struct{ floats []float64 }

func (r *sequenceRng) Float64() float64 {
	f := r.floats[0]
	if len(r.floats) > 1 {
		r.floats = r.floats[1:]
	}
	return f
}

func options(chance float64) domain.GenerateOptions {
	return domain.GenerateOptions{Slug: "test", Name: "Test", Width: 64, Height: 32, RockChance: chance, RegionNames: []string{"North", "South"}}
}

// eachCell calls fn with every cell of the map.
func eachCell(m domain.Map, fn func(core.Cell)) {
	for row := range m.Height {
		for col := range m.Width {
			fn(cell(col, row))
		}
	}
}

// ADR 0003 test 6, and ADR 0001 test 1
func TestGenerate(t *testing.T) {
	starter := domain.Generate(domain.StarterWorld(), seeded(42))

	t.Run("the same seed gives the same world", func(t *testing.T) {
		assert.Equal(t, starter, domain.Generate(domain.StarterWorld(), seeded(42)))
	})

	t.Run("the starter world is the donut's surface: 1024 x 512, wrapping, all grass at height 0", func(t *testing.T) {
		assert.Equal(t, domain.Summary{Slug: "world", Name: "World", Width: 1024, Height: 512, Wraps: true}, starter.Summary)
		require.Len(t, starter.Chunks, 32*16)
		for key, chunk := range starter.Chunks {
			assert.Equal(t, 1, chunk.Version, "chunk %v", key)
			for i := range chunk.Ground {
				require.Equal(t, domain.TileGrass, chunk.Ground[i], "chunk %v, cell %d", key, i)
				require.Zero(t, chunk.Height[i], "chunk %v, cell %d", key, i)
			}
		}
		require.NoError(t, starter.Validate())
	})

	t.Run("every cell is in one of the 12 regions, and every region has cells", func(t *testing.T) {
		require.Len(t, starter.Regions, 12)
		assert.Equal(t, domain.Region{ID: 1, Name: "Sprinkle Meadow", Biome: domain.BiomeMeadow}, starter.Regions[0])
		cells := map[uint8]int{}
		eachCell(starter, func(c core.Cell) { cells[starter.RegionAt(c)]++ })
		for _, region := range starter.Regions {
			assert.Positive(t, cells[uint8(region.ID)], "region %d %q has cells", region.ID, region.Name)
		}
		assert.Zero(t, cells[0], "no cell without a region")
	})

	t.Run("a cell is in the region of the closest point, distances wrapping", func(t *testing.T) {
		// two points on a 160-wide wrapping map: (20, 80) and (100, 80)
		opts := domain.GenerateOptions{Width: 160, Height: 160, Wraps: true, RegionNames: []string{"West", "East"}}
		m := domain.Generate(opts, &sequenceRng{floats: []float64{20.0 / 160, 0.5, 100.0 / 160, 0.5, 0.99}})
		assert.Equal(t, uint8(1), m.RegionAt(cell(30, 80)))
		assert.Equal(t, uint8(2), m.RegionAt(cell(90, 80)))
		assert.Equal(t, uint8(1), m.RegionAt(cell(150, 80)), "30 cells from the west point across the seam, 50 from the east one")

		opts.Wraps = false
		bounded := domain.Generate(opts, &sequenceRng{floats: []float64{20.0 / 160, 0.5, 100.0 / 160, 0.5, 0.99}})
		assert.Equal(t, uint8(2), bounded.RegionAt(cell(150, 80)), "no seam on a bounded map")
	})

	t.Run("no rock chance gives no rocks, a full chance a rock on every cell", func(t *testing.T) {
		assert.Empty(t, domain.Generate(options(0), constantRng(0.5)).Objects)
		assert.Len(t, domain.Generate(options(1), constantRng(0.99)).Objects, 64*32)
	})

	t.Run("rocks are inside the map, one per cell at most", func(t *testing.T) {
		m := domain.Generate(options(0.3), seeded(7))
		assert.NotEmpty(t, m.Objects)
		require.NoError(t, m.Validate())
	})
}

// ADR 0003 test 4, and ADR 0001 test 2
func TestValidate(t *testing.T) {
	valid := func() domain.Map { return domain.Generate(options(0), constantRng(0.5)) }
	rock := func(col, row int) domain.Object { return domain.Object{Kind: domain.ObjectRock, Col: col, Row: row} }
	firstChunk := func(m *domain.Map) *domain.Chunk { return m.Chunks[core.ChunkKey{}] }

	cases := map[string]func(m *domain.Map){
		"a width of 0":                   func(m *domain.Map) { m.Width = 0 },
		"a width not a multiple of 32":   func(m *domain.Map) { m.Width = 48 },
		"a height over 4096":             func(m *domain.Map) { m.Height = 4128 },
		"a wrapping map under 160 cells": func(m *domain.Map) { m.Wraps = true },
		"a chunk outside the map": func(m *domain.Map) {
			m.Chunks[core.ChunkKey{CX: 2}] = domain.NewChunk(core.ChunkKey{CX: 2}, domain.TileGrass)
		},
		"a chunk stored under another key": func(m *domain.Map) {
			m.Chunks[core.ChunkKey{CX: 1}] = domain.NewChunk(core.ChunkKey{}, domain.TileGrass)
		},
		"a chunk at version 0":         func(m *domain.Map) { firstChunk(m).Version = 0 },
		"an unknown ground code":       func(m *domain.Map) { firstChunk(m).Ground[3] = domain.TileKind(9) },
		"a cell in an unknown region":  func(m *domain.Map) { firstChunk(m).Region[0] = 3 },
		"a region with id 0":           func(m *domain.Map) { m.Regions[0].ID = 0 },
		"two regions with one id":      func(m *domain.Map) { m.Regions[1].ID = 1 },
		"a region without a name":      func(m *domain.Map) { m.Regions[0].Name = "" },
		"a region of an unknown biome": func(m *domain.Map) { m.Regions[0].Biome = "lava" },
		"an unknown object kind":       func(m *domain.Map) { m.Objects = []domain.Object{{Kind: "tree", Col: 1, Row: 1}} },
		"an object outside":            func(m *domain.Map) { m.Objects = []domain.Object{rock(64, 0)} },
		"an object at a negative":      func(m *domain.Map) { m.Objects = []domain.Object{rock(0, -1)} },
		"an object on void":            func(m *domain.Map) { m.SetGround(cell(2, 2), domain.TileVoid); m.Objects = []domain.Object{rock(2, 2)} },
		"an object on a missing chunk": func(m *domain.Map) { delete(m.Chunks, core.ChunkKey{CX: 1}); m.Objects = []domain.Object{rock(40, 2)} },
		"two objects on one cell":      func(m *domain.Map) { m.Objects = []domain.Object{rock(2, 2), rock(2, 2)} },
	}
	for name, breakIt := range cases {
		t.Run("refuses "+name, func(t *testing.T) {
			m := valid()
			breakIt(&m)
			assert.ErrorIs(t, m.Validate(), errd.ErrBadRequest)
		})
	}

	t.Run("accepts a valid map, missing chunks included", func(t *testing.T) {
		m := valid()
		m.Objects = []domain.Object{rock(0, 0), rock(63, 31)}
		assert.NoError(t, m.Validate())
		delete(m.Chunks, core.ChunkKey{CX: 1})
		m.Objects = m.Objects[:1]
		assert.NoError(t, m.Validate(), "a missing chunk is all void")
	})

	t.Run("accepts a wrapping map of 160 x 160", func(t *testing.T) {
		m := domain.Generate(domain.GenerateOptions{Width: 160, Height: 160, Wraps: true}, constantRng(0.5))
		assert.NoError(t, m.Validate())
	})
}

func mapWithRocks(width, height int, rocks ...core.Cell) domain.Map {
	m := domain.Generate(domain.GenerateOptions{Width: width, Height: height}, constantRng(0.5))
	for _, cell := range rocks {
		m.Objects = append(m.Objects, domain.Object{Kind: domain.ObjectRock, Col: cell.Col, Row: cell.Row})
	}
	return m
}

// ADR 0001 test 3, and ADR 0003 test 7 for void
func TestTerrainIsWalkable(t *testing.T) {
	m := mapWithRocks(4, 3, cell(1, 1))
	m.SetGround(cell(2, 2), domain.TileVoid)
	terrain := domain.NewTerrain(m)

	assert.True(t, terrain.IsWalkable(cell(0, 0)), "an empty cell")
	assert.False(t, terrain.IsWalkable(cell(1, 1)), "a rock")
	assert.False(t, terrain.IsWalkable(cell(2, 2)), "void")
	assert.False(t, terrain.IsWalkable(cell(4, 0)), "outside, right")
	assert.False(t, terrain.IsWalkable(cell(0, -1)), "outside, above")
}

// ADR 0003 test 9, and ADR 0001 test 4
func TestTerrainSpawnPoint(t *testing.T) {
	t.Run("the center when free", func(t *testing.T) {
		spawn, err := domain.NewTerrain(mapWithRocks(5, 5)).SpawnPoint()
		require.NoError(t, err)
		assert.Equal(t, cell(2, 2), spawn)
	})

	t.Run("the closest free cell when the center is taken", func(t *testing.T) {
		spawn, err := domain.NewTerrain(mapWithRocks(5, 5, cell(2, 2), cell(2, 1))).SpawnPoint()
		require.NoError(t, err)
		assert.Equal(t, cell(1, 2), spawn)
	})

	t.Run("the center of the starter world", func(t *testing.T) {
		spawn, err := domain.NewTerrain(domain.Generate(domain.StarterWorld(), constantRng(0.5))).SpawnPoint()
		require.NoError(t, err)
		assert.Equal(t, cell(511, 255), spawn, "ties go to the lowest row, then column")
	})

	t.Run("an error on a full map", func(t *testing.T) {
		_, err := domain.NewTerrain(mapWithRocks(1, 1, cell(0, 0))).SpawnPoint()
		assert.ErrorIs(t, err, errd.ErrConflict)
	})
}

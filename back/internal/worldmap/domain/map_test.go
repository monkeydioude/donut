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
		"a height of 4":                func(m *domain.Map) { firstChunk(m).Height[5] = 4 },
		"a height of -4":               func(m *domain.Map) { firstChunk(m).Height[5] = -4 },
		"a rock on water": func(m *domain.Map) {
			m.SetGround(cell(2, 2), domain.TileWater)
			m.Objects = []domain.Object{rock(2, 2)}
		},
		"a rock in a hole": func(m *domain.Map) { m.SetGround(cell(2, 2), domain.TileHole); m.Objects = []domain.Object{rock(2, 2)} },
		"a rock on a bridge": func(m *domain.Map) {
			m.SetGround(cell(2, 2), domain.TileBridge)
			m.Objects = []domain.Object{rock(2, 2)}
		},
		"a rock on the ocean": func(m *domain.Map) {
			m.SetGround(cell(2, 2), domain.TileOcean)
			m.Objects = []domain.Object{rock(2, 2)}
		},
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

	t.Run("accepts water, holes, bridges, the ocean and heights from -3 to 3 (ADR 0007, ADR 0008)", func(t *testing.T) {
		m := valid()
		for i, kind := range []domain.TileKind{domain.TileWater, domain.TileHole, domain.TileBridge, domain.TileOcean} {
			m.SetGround(cell(i, 0), kind)
		}
		m.SetHeight(cell(0, 1), domain.MinHeight)
		m.SetHeight(cell(1, 1), domain.MaxHeight)
		assert.NoError(t, m.Validate())
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

// ADR 0001 test 3, ADR 0003 test 7 for void, and ADR 0007 test 2
func TestTerrainIsWalkable(t *testing.T) {
	m := mapWithRocks(4, 3, cell(1, 1))
	m.SetGround(cell(2, 2), domain.TileVoid)
	m.SetGround(cell(0, 1), domain.TileWater)
	m.SetGround(cell(0, 2), domain.TileHole)
	m.SetGround(cell(3, 0), domain.TileBridge)
	m.SetGround(cell(3, 2), domain.TileOcean)
	terrain := domain.NewTerrain(m)

	assert.True(t, terrain.IsWalkable(cell(0, 0)), "an empty cell")
	assert.True(t, terrain.IsWalkable(cell(3, 0)), "a bridge")
	assert.False(t, terrain.IsWalkable(cell(1, 1)), "a rock")
	assert.False(t, terrain.IsWalkable(cell(2, 2)), "void")
	assert.False(t, terrain.IsWalkable(cell(0, 1)), "water")
	assert.False(t, terrain.IsWalkable(cell(0, 2)), "a hole")
	assert.False(t, terrain.IsWalkable(cell(3, 2)), "the ocean")
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

	t.Run("the center of the flat starter world", func(t *testing.T) {
		spawn, err := domain.NewTerrain(domain.Generate(flatStarter(), constantRng(0.5))).SpawnPoint()
		require.NoError(t, err)
		assert.Equal(t, cell(1023, 511), spawn, "ties go to the lowest row, then column")
	})

	t.Run("an error on a full map", func(t *testing.T) {
		_, err := domain.NewTerrain(mapWithRocks(1, 1, cell(0, 0))).SpawnPoint()
		assert.ErrorIs(t, err, errd.ErrConflict)
	})
}

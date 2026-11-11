package domain_test

import (
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
	"donut/internal/worldmap/domain"
)

// flatStarter is the starter world without its terrain nor its dome: flat grass and rocks.
func flatStarter() domain.GenerateOptions {
	opts := domain.StarterWorld()
	opts.Relief, opts.Landmasses, opts.Water, opts.Holes = domain.ReliefOptions{}, domain.LandmassOptions{}, domain.WaterOptions{}, domain.HoleOptions{}
	opts.Mountains, opts.Glaciers, opts.Dome = domain.MountainOptions{}, domain.GlacierOptions{}, domain.DomeOptions{}
	return opts
}

// starter is the starter world of seed 42, generated once for every test: it takes a while.
var starter = sync.OnceValue(func() domain.Map { return domain.Generate(domain.StarterWorld(), seeded(42)) })

// ADR 0003 test 6, ADR 0001 test 1, ADR 0007 test 7 and ADR 0008 test 3
func TestGenerate(t *testing.T) {
	t.Run("the same seed gives the same world, another seed another one", func(t *testing.T) {
		assert.Equal(t, starter(), domain.Generate(domain.StarterWorld(), seeded(42)))
		assert.NotEqual(t, starter().Chunks, domain.Generate(domain.StarterWorld(), seeded(43)).Chunks)
	})

	t.Run("the starter world is the donut's surface: 2048 x 1024, wrapping, and valid", func(t *testing.T) {
		assert.Equal(t, domain.Summary{Slug: "world", Name: "World", Width: 2048, Height: 1024, Wraps: true}, starter().Summary)
		require.Len(t, starter().Chunks, 64*32)
		for key, chunk := range starter().Chunks {
			assert.Equal(t, 1, chunk.Version, "chunk %v", key)
		}
		require.NoError(t, starter().Validate())
	})

	t.Run("without terrain, all grass at height 0", func(t *testing.T) {
		flat := domain.Generate(flatStarter(), seeded(42))
		for key, chunk := range flat.Chunks {
			for i := range chunk.Ground {
				require.Equal(t, domain.TileGrass, chunk.Ground[i], "chunk %v, cell %d", key, i)
				require.Zero(t, chunk.Height[i], "chunk %v, cell %d", key, i)
			}
		}
	})

	t.Run("every cell is in one of the 18 regions, and every region has cells", func(t *testing.T) {
		m := starter()
		require.Len(t, m.Regions, 18)
		assert.Equal(t, domain.Region{ID: 1, Name: "Sprinkle Meadow", Biome: domain.BiomeMeadow}, m.Regions[0])
		assert.Equal(t, domain.Region{ID: 18, Name: "Custard Sea", Biome: domain.BiomeMeadow}, m.Regions[17])
		cells := map[uint8]int{}
		eachCell(m, func(c core.Cell) { cells[m.RegionAt(c)]++ })
		for _, region := range m.Regions {
			assert.Positive(t, cells[uint8(region.ID)], "region %d %q has cells", region.ID, region.Name)
		}
		assert.Zero(t, cells[0], "no cell without a region")
	})

	t.Run("without landmasses, a cell is in the region of the closest point, distances wrapping", func(t *testing.T) {
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

// ADR 0008 test 4
func TestGeneratedLandmasses(t *testing.T) {
	m := bare()
	land := bodies(m, isLand)
	cells := m.Width * m.Height

	t.Run("land is 30 to 50% of the cells, the rest ocean at -1", func(t *testing.T) {
		landCells, ocean := 0, 0
		eachCell(m, func(c core.Cell) {
			if m.GroundAt(c) == domain.TileOcean {
				ocean++
				require.Equal(t, int8(-1), m.HeightAt(c), "the ocean at %v", c)
			} else {
				landCells++
			}
		})
		assert.Equal(t, cells, landCells+ocean, "every cell is land or ocean")
		assert.InDelta(t, 0.4, float64(landCells)/float64(cells), 0.1)
	})

	t.Run("3 continents of more than 100,000 cells, and 4 to 6 islands of 8,000 to 60,000", func(t *testing.T) {
		continents, islands := 0, 0
		for _, body := range land {
			switch {
			case len(body) > 100_000:
				continents++
			case len(body) >= 8_000 && len(body) <= 60_000:
				islands++
			}
		}
		assert.Equal(t, 3, continents)
		assert.True(t, islands >= 4 && islands <= 6, "%d islands", islands)
	})

	t.Run("the spawn is on the continent that holds the center", func(t *testing.T) {
		spawn := spawnOf(t, m)
		center := cell(m.Width/2-1, m.Height/2-1)
		body := bodyHolding(m, land, spawn)
		require.NotNil(t, body)
		assert.Greater(t, len(body), 100_000)
		assert.True(t, bodyHas(m, body, center), "the spawn's continent holds the center %v", center)
	})
}

// ADR 0007 test 8 and ADR 0008 test 5
func TestGeneratedRelief(t *testing.T) {
	m := bare()
	grid := m.Grid()
	toOcean := stepsFrom(m, func(c core.Cell) bool { return m.GroundAt(c) == domain.TileOcean })
	perLevel := map[int8]int{}
	landCells := 0
	coast, inland := levelSum{}, levelSum{}
	eachCell(m, func(c core.Cell) {
		h := m.HeightAt(c)
		require.True(t, h >= domain.MinHeight && h <= domain.MaxGroundHeight, "%v at %d", c, h)
		for _, n := range neighborsOf(grid, c) {
			if absInt(int(h)-int(m.HeightAt(n))) > 1 {
				require.Fail(t, "a cliff", "%v at %d next to %v at %d", c, h, n, m.HeightAt(n))
			}
		}
		if !isLand(m.GroundAt(c)) {
			return
		}
		landCells++
		perLevel[h]++
		switch steps := toOcean[grid.Index(c)]; {
		case steps <= 10:
			coast.add(h)
		case steps > 100:
			inland.add(h)
		}
	})
	assert.InDelta(t, 0.5, float64(perLevel[0])/float64(landCells), 0.1, "about half the land at 0: %v", perLevel)
	for _, end := range []int8{domain.MinHeight, domain.MaxGroundHeight} {
		assert.Positive(t, perLevel[end], "some cells at %d", end)
		assert.Less(t, float64(perLevel[end])/float64(landCells), 0.05, "few cells at %d", end)
	}
	assert.Greater(t, inland.mean(), coast.mean(), "the land rises inland: coast %v, inland %v", coast, inland)
}

// ADR 0008 test 6
func TestGeneratedBanks(t *testing.T) {
	m := bare()
	grid := m.Grid()
	eachCell(m, func(c core.Cell) {
		if !isLand(m.GroundAt(c)) {
			return
		}
		lowest, highest, water := int8(domain.MaxGroundHeight), int8(domain.MinHeight), false
		for _, n := range neighborsOf(grid, c) {
			if kind := m.GroundAt(n); kind == domain.TileWater || kind == domain.TileOcean {
				water = true
				lowest, highest = min(lowest, m.HeightAt(n)), max(highest, m.HeightAt(n))
			}
		}
		if water {
			h := m.HeightAt(c)
			require.True(t, h >= lowest && h <= highest+1, "%v at %d is a bank of water from %d to %d", c, h, lowest, highest)
		}
	})
}

// ADR 0008 tests 7 and 8
func TestGeneratedLakesAndRivers(t *testing.T) {
	m := bare()
	grid := m.Grid()
	waters := bodies(m, isFreshWater)
	var lakes, rivers [][]core.Cell
	for _, body := range waters {
		if len(body) <= 20 {
			lakes = append(lakes, body)
		} else {
			rivers = append(rivers, body)
		}
	}

	t.Run("at least 3 standalone lakes: flat bodies of 6 to 20 cells", func(t *testing.T) {
		standalone := 0
		for _, lake := range lakes {
			levels := levelsOf(m, lake)
			require.Len(t, levels, 1, "the lake at %v is flat: %v", lake[0], levels)
			if len(lake) >= 6 {
				standalone++
			}
		}
		assert.GreaterOrEqual(t, standalone, 3)
	})

	t.Run("river water is at -1 or above, unless it sinks into a lake below the sea", func(t *testing.T) {
		for _, river := range rivers {
			bottom := levelsOf(m, river)[0]
			if bottom >= -1 {
				continue
			}
			sink := largest(bodiesWithin(m, river, func(c core.Cell) bool { return m.GroundAt(c) == domain.TileWater && m.HeightAt(c) == bottom }))
			assert.GreaterOrEqual(t, len(sink), 6, "the river at %v ends in a lake at %d", river[0], bottom)
		}
	})

	t.Run("at least 10 rivers, each from a flat lake of 6 cells or more at its highest level, flowing down", func(t *testing.T) {
		require.GreaterOrEqual(t, len(rivers), 10)
		for _, river := range rivers {
			levels := levelsOf(m, river)
			top, bottom := levels[len(levels)-1], levels[0]
			source := bodiesWithin(m, river, func(c core.Cell) bool { return m.GroundAt(c) == domain.TileWater && m.HeightAt(c) == top })
			assert.GreaterOrEqual(t, len(largest(source)), 6, "the river at %v has a source lake at %d", river[0], top)
			for _, c := range river {
				if m.GroundAt(c) != domain.TileWater || m.HeightAt(c) == bottom {
					continue
				}
				flows := false
				for _, n := range neighborsOf(grid, c) {
					flows = flows || (m.GroundAt(n) == domain.TileWater && m.HeightAt(n) <= m.HeightAt(c))
				}
				require.True(t, flows, "the water at %v flows on", c)
			}
		}
	})

	t.Run("a river reaches the ocean", func(t *testing.T) {
		reaches := false
		for _, river := range rivers {
			for _, c := range river {
				for _, n := range neighborsOf(grid, c) {
					reaches = reaches || m.GroundAt(n) == domain.TileOcean
				}
			}
		}
		assert.True(t, reaches)
	})

	t.Run("at the spawn's column, water is 15 to 30 rows above the spawn", func(t *testing.T) {
		spawn := spawnOf(t, m)
		rows := []int{}
		for above := 15; above <= 30; above++ {
			if c, _ := grid.Normalize(cell(spawn.Col, spawn.Row-above)); isFreshWater(m.GroundAt(c)) {
				rows = append(rows, above)
			}
		}
		assert.NotEmpty(t, rows, "water above the spawn %v", spawn)
	})
}

// ADR 0008 test 9
func TestGeneratedBridges(t *testing.T) {
	m := starter()
	grid := m.Grid()
	terrain := domain.NewTerrain(m)
	spawn := spawnOf(t, m)
	bridges := bridgesOf(t, m)

	t.Run("at least 10 bridges, each a straight run of 1 to 3 cells, 1 wide, a level above the water, with walkable ends", func(t *testing.T) {
		require.GreaterOrEqual(t, len(bridges), 10)
		for _, b := range bridges {
			require.True(t, len(b.deck) >= 1 && len(b.deck) <= 3, "the bridge at %v spans %d cells", b.deck[0], len(b.deck))
			for _, c := range b.deck {
				require.Equal(t, b.level, m.HeightAt(c), "%v is at the deck's level", c)
				for _, side := range []core.Cell{{Col: c.Col + b.along.Row, Row: c.Row + b.along.Col}, {Col: c.Col - b.along.Row, Row: c.Row - b.along.Col}} {
					side, _ = grid.Normalize(side)
					require.NotEqual(t, domain.TileBridge, m.GroundAt(side), "%v: the bridge is 1 wide", side)
					if m.GroundAt(side) == domain.TileWater {
						require.Equal(t, b.level-1, m.HeightAt(side), "the water beside %v is a level under the deck", c)
					}
				}
			}
			for _, end := range b.ends {
				assert.Equal(t, domain.TileGrass, m.GroundAt(end), "%v is a bank", end)
				assert.Equal(t, b.level, m.HeightAt(end), "%v at the deck's level", end)
				assert.True(t, terrain.IsWalkable(end), "%v is walkable", end)
			}
		}
	})

	t.Run("one is within 30 columns of the spawn, and a path crosses it from the spawn", func(t *testing.T) {
		var nearest *bridge
		offset := m.Width
		for i, b := range bridges {
			if d := absInt(grid.Offset(spawn, b.deck[0]).Col); d < offset {
				nearest, offset = &bridges[i], d
			}
		}
		require.NotNil(t, nearest)
		require.LessOrEqual(t, offset, 30)
		// the far end is the one farther up: the spawn's river runs above it
		farEnd := nearest.ends[0]
		if grid.Offset(spawn, nearest.ends[1]).Row < grid.Offset(spawn, farEnd).Row {
			farEnd = nearest.ends[1]
		}
		assert.NotEmpty(t, terrain.Path(spawn, farEnd), "from %v to %v, across the bridge at %v", spawn, farEnd, nearest.deck[0])
	})
}

// ADR 0007 test 11 and ADR 0008 test 10
func TestGeneratedHoles(t *testing.T) {
	m := starter()
	grid := m.Grid()
	holes := map[core.Cell]bool{}
	eachCell(m, func(c core.Cell) {
		if m.GroundAt(c) == domain.TileHole {
			holes[c] = true
		}
	})
	require.NotEmpty(t, holes)

	t.Run("in blobs of 1 to 6 cells", func(t *testing.T) {
		for _, blob := range bodies(m, func(kind domain.TileKind) bool { return kind == domain.TileHole }) {
			require.True(t, len(blob) >= 1 && len(blob) <= 6, "the hole at %v has %d cells", blob[0], len(blob))
		}
	})

	t.Run("at least 3 steps from any water, the ocean, or a bridge", func(t *testing.T) {
		for hole := range holes {
			for dc := -2; dc <= 2; dc++ {
				for dr := -2; dr <= 2; dr++ {
					c, _ := grid.Normalize(cell(hole.Col+dc, hole.Row+dr))
					kind := m.GroundAt(c)
					// mountains and glaciers come after the holes (ADR 0009): they may stand next to one
					dry := kind == domain.TileGrass || kind == domain.TileHole || kind == domain.TileMountain || kind == domain.TileGlacier
					require.True(t, dry, "the hole at %v is close to %v at %v", hole, kind, c)
				}
			}
		}
	})
}

// ADR 0007 test 12
func TestGeneratedRocks(t *testing.T) {
	m := starter()
	ends := map[core.Cell]bool{}
	for _, b := range bridgesOf(t, m) {
		for _, end := range b.ends {
			ends[end] = true
		}
	}
	require.NotEmpty(t, m.Objects)
	for _, rock := range m.Objects {
		assert.Equal(t, domain.TileGrass, m.GroundAt(rock.Cell()), "the rock at %v", rock.Cell())
		assert.False(t, ends[rock.Cell()], "the rock at %v is on a bridge's end", rock.Cell())
	}
}

// ADR 0008 test 11
func TestGeneratedRegions(t *testing.T) {
	m := bare()
	grid := m.Grid()
	// each landmass is numbered; islets and the ocean aren't
	massOf := make([]int, m.Width*m.Height)
	for k, body := range bodies(m, isLand) {
		for _, c := range body {
			if len(body) >= 8_000 {
				massOf[grid.Index(c)] = k + 1
			}
		}
	}
	masses, sea := map[uint8]map[int]bool{}, map[uint8]bool{}
	eachCell(m, func(c core.Cell) {
		region := m.RegionAt(c)
		if m.GroundAt(c) == domain.TileOcean {
			sea[region] = true
			return
		}
		if masses[region] == nil {
			masses[region] = map[int]bool{}
		}
		if mass := massOf[grid.Index(c)]; mass > 0 {
			masses[region][mass] = true
		}
	})
	for region, held := range masses {
		assert.LessOrEqual(t, len(held), 1, "region %d spans one landmass: %v", region, held)
		assert.False(t, sea[region], "region %d is on land only", region)
	}
	assert.Len(t, sea, 3, "3 ocean regions")
}

// bridge is a bridge of the map: its deck cells in a line along `along`, the two bank cells past its ends, and its level.
type bridge struct {
	deck  []core.Cell
	along core.Cell
	ends  [2]core.Cell
	level int8
}

// bridgesOf finds each bridge: a run of bridge cells along a column or a row.
func bridgesOf(t *testing.T, m domain.Map) []bridge {
	t.Helper()
	grid := m.Grid()
	var bridges []bridge
	for _, deck := range bodies(m, func(kind domain.TileKind) bool { return kind == domain.TileBridge }) {
		along := core.Cell{Row: 1}
		if len(deck) > 1 && deck[0].Row == deck[1].Row && deck[0].Col != deck[1].Col {
			along = core.Cell{Col: 1}
		} else if len(deck) == 1 {
			up, _ := grid.Normalize(cell(deck[0].Col, deck[0].Row-1))
			if m.GroundAt(up) != domain.TileGrass {
				along = core.Cell{Col: 1}
			}
		}
		first, last := deck[0], deck[0]
		for _, c := range deck {
			if o := grid.Offset(first, c); o.Col*along.Col+o.Row*along.Row < 0 {
				first = c
			}
			if o := grid.Offset(last, c); o.Col*along.Col+o.Row*along.Row > 0 {
				last = c
			}
		}
		before, _ := grid.Normalize(cell(first.Col-along.Col, first.Row-along.Row))
		after, _ := grid.Normalize(cell(last.Col+along.Col, last.Row+along.Row))
		bridges = append(bridges, bridge{deck: deck, along: along, ends: [2]core.Cell{before, after}, level: m.HeightAt(deck[0])})
	}
	return bridges
}

func isLand(kind domain.TileKind) bool {
	return kind != domain.TileOcean && kind != domain.TileVoid
}

// isFreshWater: the kinds a lake or a river is made of, its bridges included, so a bridge doesn't cut it in two.
func isFreshWater(kind domain.TileKind) bool {
	return kind == domain.TileWater || kind == domain.TileBridge
}

// bodies are the connected groups of cells of the kinds `in` allows, diagonals included, across the seams.
func bodies(m domain.Map, in func(domain.TileKind) bool) [][]core.Cell {
	return bodiesWithin(m, nil, func(c core.Cell) bool { return in(m.GroundAt(c)) })
}

// bodiesWithin are the connected groups among `cells` (every cell when nil) that `in` allows.
func bodiesWithin(m domain.Map, cells []core.Cell, in func(core.Cell) bool) [][]core.Cell {
	grid := m.Grid()
	seen := make([]bool, m.Width*m.Height)
	var out [][]core.Cell
	visit := func(start core.Cell) {
		if i := grid.Index(start); seen[i] || !in(start) {
			return
		} else {
			seen[i] = true
		}
		body := []core.Cell{start}
		for q := 0; q < len(body); q++ {
			for _, n := range neighborsOf(grid, body[q]) {
				if j := grid.Index(n); !seen[j] && in(n) {
					seen[j] = true
					body = append(body, n)
				}
			}
		}
		out = append(out, body)
	}
	if cells != nil {
		for _, c := range cells {
			visit(c)
		}
	} else {
		eachCell(m, visit)
	}
	return out
}

func largest(bodies [][]core.Cell) []core.Cell {
	var best []core.Cell
	for _, body := range bodies {
		if len(body) > len(best) {
			best = body
		}
	}
	return best
}

func bodyHolding(m domain.Map, bodies [][]core.Cell, c core.Cell) []core.Cell {
	for _, body := range bodies {
		if bodyHas(m, body, c) {
			return body
		}
	}
	return nil
}

func bodyHas(m domain.Map, body []core.Cell, c core.Cell) bool {
	for _, other := range body {
		if other == c {
			return true
		}
	}
	return false
}

// levelsOf are the heights of a body's water cells, sorted, each once.
func levelsOf(m domain.Map, body []core.Cell) []int8 {
	seen := map[int8]bool{}
	for _, c := range body {
		if m.GroundAt(c) == domain.TileWater {
			seen[m.HeightAt(c)] = true
		}
	}
	var levels []int8
	for level := int8(domain.MinHeight); level <= domain.MaxHeight; level++ {
		if seen[level] {
			levels = append(levels, level)
		}
	}
	return levels
}

// stepsFrom counts each cell's steps from the closest cell `from` allows, diagonals counting as 1; −1 when none.
func stepsFrom(m domain.Map, from func(core.Cell) bool) []int32 {
	grid := m.Grid()
	steps := make([]int32, m.Width*m.Height)
	var ring []core.Cell
	eachCell(m, func(c core.Cell) {
		steps[grid.Index(c)] = -1
		if from(c) {
			steps[grid.Index(c)] = 0
			ring = append(ring, c)
		}
	})
	for distance := int32(1); len(ring) > 0; distance++ {
		var next []core.Cell
		for _, c := range ring {
			for _, n := range neighborsOf(grid, c) {
				if steps[grid.Index(n)] < 0 {
					steps[grid.Index(n)] = distance
					next = append(next, n)
				}
			}
		}
		ring = next
	}
	return steps
}

// levelSum averages heights.
type levelSum struct{ total, count int }

func (s *levelSum) add(h int8) { s.total, s.count = s.total+int(h), s.count+1 }
func (s levelSum) mean() float64 {
	if s.count == 0 {
		return 0
	}
	return float64(s.total) / float64(s.count)
}

func spawnOf(t *testing.T, m domain.Map) core.Cell {
	t.Helper()
	spawn, err := domain.NewTerrain(m).SpawnPoint()
	require.NoError(t, err)
	return spawn
}

// neighborsOf are the 8 cells around c, across the seams on a wrapping map.
func neighborsOf(grid core.Grid, c core.Cell) []core.Cell {
	var cells []core.Cell
	for dc := -1; dc <= 1; dc++ {
		for dr := -1; dr <= 1; dr++ {
			if n, ok := grid.Normalize(cell(c.Col+dc, c.Row+dr)); ok && (dc != 0 || dr != 0) {
				cells = append(cells, n)
			}
		}
	}
	return cells
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// bare is the starter world of seed 42 without its massifs and glacier ring: the land as ADR 0008 makes it, whose
// rules hold before the freeze. Its ground and heights are the starter's, but for the cells the massifs and the ring
// took. Rocks and regions differ: they are drawn after them.
var bare = sync.OnceValue(func() domain.Map {
	opts := domain.StarterWorld()
	opts.Mountains, opts.Glaciers = domain.MountainOptions{}, domain.GlacierOptions{}
	return domain.Generate(opts, seeded(42))
})

// cellsOf are the cells of a ground kind, row by row.
func cellsOf(m domain.Map, kind domain.TileKind) []core.Cell {
	var cells []core.Cell
	eachCell(m, func(c core.Cell) {
		if m.GroundAt(c) == kind {
			cells = append(cells, c)
		}
	})
	return cells
}

// ADR 0011 test 5
func TestGeneratedMassifsAndRing(t *testing.T) {
	m := starter()
	grid := m.Grid()
	spawn := spawnOf(t, m)
	// the generator's spawn, where the pass and the massifs' clearance are measured from: Doma-1's dome stands on it,
	// so the terrain's spawn is a cell beside it (ADR 0012)
	center := domain.StarterWorld().Mountains.Spawn
	require.Equal(t, center, domain.StarterWorld().Glaciers.Spawn)
	ridge := m.Height / 2
	inPass := func(c core.Cell) bool { return absInt(grid.Offset(center, c).Col) <= 48 }

	t.Run("only mountains and glaciers rise above +3", func(t *testing.T) {
		eachCell(m, func(c core.Cell) {
			if h := m.HeightAt(c); h > domain.MaxGroundHeight {
				require.True(t, m.GroundAt(c).RisesAboveGround(), "%v at %d is %v", c, h, m.GroundAt(c))
			}
		})
		assert.NotEmpty(t, cellsOf(m, domain.TileMountain))
	})

	t.Run("the core is glacier, holes and mountains outside the pass; the edge is ragged; nothing freezes past 64 rows", func(t *testing.T) {
		edgeIce, edgeOther := 0, 0
		eachCell(m, func(c core.Cell) {
			kind, r := m.GroundAt(c), absInt(c.Row-ridge)
			switch {
			case kind == domain.TileGlacier:
				require.LessOrEqual(t, r, 64, "the glacier at %v", c)
				require.False(t, inPass(c), "the glacier at %v is in the pass", c)
			case r <= 40 && !inPass(c):
				require.True(t, kind == domain.TileHole || kind == domain.TileMountain, "%v at %v, in the core", kind, c)
			}
			if r > 40 && r <= 64 && !inPass(c) {
				if kind == domain.TileGlacier {
					edgeIce++
				} else {
					edgeOther++
				}
			}
		})
		assert.Positive(t, edgeIce, "some of the edge is glacier")
		assert.Positive(t, edgeOther, "some of the edge isn't")
	})

	t.Run("no mountain within 24 cells of the spawn", func(t *testing.T) {
		for _, c := range cellsOf(m, domain.TileMountain) {
			offset := grid.Offset(center, c)
			require.GreaterOrEqual(t, math.Hypot(float64(offset.Col), float64(offset.Row)), 24.0, "the mountain at %v", c)
		}
	})

	t.Run("from the spawn, a path leaves the ring on both sides, through the pass", func(t *testing.T) {
		terrain := domain.NewTerrain(m).WithSearchLimit(m.Width * m.Height)
		for _, side := range []int{-1, 1} {
			target, found := core.Cell{}, false
			for r := 70; r < 200 && !found; r++ {
				target = core.Cell{Col: spawn.Col, Row: ridge + side*r}
				found = terrain.IsWalkable(target)
			}
			require.True(t, found, "a walkable cell past the ring, side %d", side)
			assert.NotEmpty(t, terrain.Path(spawn, target), "from %v to %v", spawn, target)
		}
	})
}

// ADR 0011 test 6
func TestMassifsAndRingOnlyReplaceCells(t *testing.T) {
	m, before := starter(), bare()
	assert.Empty(t, cellsOf(before, domain.TileMountain))
	assert.Empty(t, cellsOf(before, domain.TileGlacier))
	eachCell(m, func(c core.Cell) {
		was := before.GroundAt(c)
		switch kind := m.GroundAt(c); kind {
		case domain.TileMountain:
			require.Equal(t, domain.TileGrass, was, "the mountain at %v was grass", c)
			require.GreaterOrEqual(t, before.HeightAt(c), int8(1), "the mountain at %v was at level 1 or above", c)
			require.Greater(t, m.HeightAt(c), before.HeightAt(c), "the mountain at %v rose", c)
		case domain.TileGlacier:
			frozen := was == domain.TileGrass || was == domain.TileWater || was == domain.TileOcean || was == domain.TileBridge
			require.True(t, frozen, "the glacier at %v was %v", c, was)
			require.Greater(t, m.HeightAt(c), before.HeightAt(c), "the glacier at %v rose", c)
		default:
			require.Equal(t, was, kind, "the ground at %v", c)
			require.Equal(t, before.HeightAt(c), m.HeightAt(c), "the height at %v", c)
		}
	})
}

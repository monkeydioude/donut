package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
	"donut/internal/worldmap/app"
	"donut/internal/worldmap/domain"
	"donut/internal/worldmap/infra"
	"donut/mapfiles"
	"donut/pkg/errd"
	"donut/pkg/logs"
	"donut/pkg/pgtest"
)

type fixture struct {
	tx   pgx.Tx
	repo *infra.Repo
	maps *app.Service
}

func newFixture(t *testing.T, ctx context.Context) fixture {
	tx := pgtest.MigratedTx(t, ctx)
	repo := infra.NewRepo(tx)
	return fixture{tx: tx, repo: repo, maps: app.NewService(repo, logs.Discard())}
}

// smallWorld is a wrapping 160 x 192 map with 3 regions and rocks, quicker to save than the starter world.
func smallWorld() domain.Map {
	opts := domain.GenerateOptions{Slug: "small", Name: "Small", Width: 160, Height: 192, Wraps: true, RockChance: 0.1,
		RegionNames: []string{"One", "Two", "Three"}}
	return domain.Generate(opts, seeded(5))
}

// ADR 0003 test 11, ADR 0001 test 7, ADR 0007 test 20, and ADR 0012 test 11
// DONUT_TEST_DATABASE_URL=postgres://donut:donut@localhost:5434/donut go test ./internal/worldmap/app
func TestEnsureMapsSavesTheWorldOnce(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, ctx)

	created, err := f.maps.EnsureMaps(ctx, []string{"world"}, mapfiles.Read)
	require.NoError(t, err)
	assert.Equal(t, []string{"world"}, created)

	created, err = f.maps.EnsureMaps(ctx, []string{"world"}, mapfiles.Read)
	require.NoError(t, err)
	assert.Empty(t, created, "a second boot creates nothing")

	list, err := f.maps.ListMaps(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, domain.Summary{ID: list[0].ID, Slug: "world", Name: "World", Width: 2048, Height: 1024, Wraps: true}, list[0],
		"the list has no contents")

	saved, err := f.repo.MapBySlug(ctx, "world")
	require.NoError(t, err)
	file, err := mapfiles.Read("world")
	require.NoError(t, err)
	assert.Len(t, saved.Chunks, 2048)
	assert.Equal(t, file.Chunks, saved.Chunks, "the file's cells")
	assert.Len(t, saved.Regions, 18)
	assert.Len(t, saved.Objects, len(file.Objects))
}

// ADR 0012 test 11: only the maps the database doesn't have are read and saved.
func TestEnsureMapsSavesOnlyTheMissingOnes(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, ctx)
	_, err := f.repo.InsertMap(ctx, smallWorld())
	require.NoError(t, err)
	var read []string
	files := func(slug string) (domain.Map, error) {
		read = append(read, slug)
		m := smallWorld()
		m.Slug = slug
		return m, nil
	}

	created, err := f.maps.EnsureMaps(ctx, []string{"small", "other"}, files)
	require.NoError(t, err)
	assert.Equal(t, []string{"other"}, created)
	assert.Equal(t, []string{"other"}, read, "a saved map's file isn't read")
}

// ADR 0012 test 11: reset-maps replaces each map, and stops at the first refused file.
func TestResetMapsReplacesEachMap(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, ctx)
	ids := map[string]string{}
	for _, slug := range []string{"one", "two", "three"} {
		m := smallWorld()
		m.Slug = slug
		id, err := f.repo.InsertMap(ctx, m)
		require.NoError(t, err)
		ids[slug] = id
	}
	refused := errors.New("ground.png (1, 2): #123456 isn't a ground")
	files := func(slug string) (domain.Map, error) {
		if slug == "two" {
			return domain.Map{}, refused
		}
		m := smallWorld()
		m.Slug = slug
		return m, nil
	}

	err := f.maps.ResetMaps(ctx, []string{"one", "two", "three"}, files)
	assert.ErrorIs(t, err, refused)
	for slug, replaced := range map[string]bool{"one": true, "two": false, "three": false} {
		m, err := f.repo.MapBySlug(ctx, slug)
		require.NoError(t, err)
		assert.Equal(t, replaced, m.ID != ids[slug], "%s replaced: %v", slug, replaced)
	}
}

// ADR 0012 test 12: a dome reads back with the map it leads to.
func TestADomeReadsBackWithWhereItLeads(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, ctx)
	m := smallWorld()
	m.Objects = []domain.Object{{Kind: domain.ObjectDome, Col: 32, Row: 64, LeadsTo: "doma-1"}}
	_, err := f.repo.InsertMap(ctx, m)
	require.NoError(t, err)

	read, err := f.repo.MapBySlug(ctx, "small")
	require.NoError(t, err)
	require.Len(t, read.Objects, 1)
	assert.Equal(t, domain.ObjectDome, read.Objects[0].Kind)
	assert.Equal(t, core.Cell{Col: 32, Row: 64}, read.Objects[0].Cell())
	assert.Equal(t, "doma-1", read.Objects[0].LeadsTo)
}

// ADR 0007 test 21
func TestResetMapReplacesItWithItsFile(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, ctx)
	oldID, err := f.repo.InsertMap(ctx, smallWorld())
	require.NoError(t, err)
	file := smallWorld()
	file.Name = "Small, from its file"
	file.Regions = file.Regions[:1]
	eachChunk(file, func(chunk *domain.Chunk) {
		for i := range chunk.Region {
			chunk.Region[i] = 1
			chunk.Height[i] = 2
		}
	})
	file.Objects = file.Objects[:3]
	read := func(slug string) (domain.Map, error) {
		assert.Equal(t, "small", slug)
		return file, nil
	}

	require.NoError(t, f.maps.ResetMap(ctx, "small", read))

	saved, err := f.repo.MapBySlug(ctx, "small")
	require.NoError(t, err)
	assert.NotEqual(t, oldID, saved.ID, "a new map")
	assert.Equal(t, "Small, from its file", saved.Name)
	assert.Equal(t, file.Chunks, saved.Chunks, "the file's cells")
	assert.Equal(t, []domain.Region{{ID: 1, Name: "One", Biome: domain.BiomeMeadow}}, saved.Regions)
	assert.Len(t, saved.Objects, 3, "the file's rocks only")
	var left int
	require.NoError(t, f.tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM maps) + (SELECT count(*) FROM map_objects WHERE map_id = $1)
		+ (SELECT count(*) FROM map_chunks WHERE map_id = $1) + (SELECT count(*) FROM map_regions WHERE map_id = $1)`, oldID).Scan(&left))
	assert.Equal(t, 1, left, "only the new map: nothing left of the old one")

	t.Run("a map not in the database yet is saved", func(t *testing.T) {
		file.Slug = "fresh"
		require.NoError(t, f.maps.ResetMap(ctx, "fresh", func(string) (domain.Map, error) { return file, nil }))
		_, err := f.repo.MapBySlug(ctx, "fresh")
		assert.NoError(t, err)
	})

	t.Run("a refused file changes nothing", func(t *testing.T) {
		refused := errors.New("height.png (3, 4): #8c8c8c isn't a height")
		err := f.maps.ResetMap(ctx, "small", func(string) (domain.Map, error) { return domain.Map{}, refused })
		assert.ErrorIs(t, err, refused)
		still, err := f.repo.MapBySlug(ctx, "small")
		require.NoError(t, err)
		assert.Equal(t, saved.ID, still.ID)
	})
}

func eachChunk(m domain.Map, fn func(*domain.Chunk)) {
	for _, chunk := range m.Chunks {
		fn(chunk)
	}
}

// ADR 0003 test 12, and ADR 0001 test 8
func TestAMapReadsBackAsSaved(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, ctx)
	saved := smallWorld()
	_, err := f.repo.InsertMap(ctx, saved)
	require.NoError(t, err)

	read, err := f.repo.MapBySlug(ctx, "small")
	require.NoError(t, err)
	assert.NotEmpty(t, read.ID)
	saved.ID = read.ID
	assert.Equal(t, saved.Description, read.Description, "size, wrapping and regions")
	assert.Equal(t, saved.Chunks, read.Chunks, "every cell of every chunk")
	require.Len(t, read.Objects, len(saved.Objects))
	for i, object := range read.Objects {
		assert.NotEmpty(t, object.ID)
		assert.Equal(t, saved.Objects[i].Cell(), object.Cell())
		assert.Equal(t, domain.ObjectRock, object.Kind)
		assert.Empty(t, object.LeadsTo, "a rock leads nowhere")
	}

	_, err = f.maps.World(ctx, "nowhere")
	assert.ErrorIs(t, err, errd.ErrNotFound)
	_, err = f.maps.Description(ctx, "nowhere")
	assert.ErrorIs(t, err, errd.ErrNotFound)
}

// ADR 0003 test 12
func TestAChunkWithNoRowIsVoid(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, ctx)
	id, err := f.repo.InsertMap(ctx, smallWorld())
	require.NoError(t, err)
	_, err = f.tx.Exec(ctx, `DELETE FROM map_chunks WHERE map_id = $1 AND cx = 1 AND cy = 2`, id)
	require.NoError(t, err)
	_, err = f.tx.Exec(ctx, `DELETE FROM map_objects WHERE map_id = $1 AND col BETWEEN 32 AND 63 AND row BETWEEN 64 AND 95`, id)
	require.NoError(t, err)

	world, err := f.maps.World(ctx, "small")
	require.NoError(t, err)
	contents, ok := world.Chunk(core.ChunkKey{CX: 1, CY: 2})
	require.True(t, ok)
	assert.Equal(t, []string{"void"}, contents.Kinds)
	assert.False(t, world.IsWalkable(core.Cell{Col: 40, Row: 70}))
}

func TestDescriptionHasRegionsButNoCells(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, ctx)
	_, err := f.repo.InsertMap(ctx, smallWorld())
	require.NoError(t, err)

	d, err := f.maps.Description(ctx, "small")
	require.NoError(t, err)
	assert.Equal(t, 160, d.Width)
	assert.True(t, d.Wraps)
	assert.Equal(t, []domain.Region{
		{ID: 1, Name: "One", Biome: domain.BiomeMeadow},
		{ID: 2, Name: "Two", Biome: domain.BiomeMeadow},
		{ID: 3, Name: "Three", Biome: domain.BiomeMeadow},
	}, d.Regions)
}

// ADR 0012 test 12: a city's description tells the dome it's entered from, with that map's size; the world's none.
func TestACityStandsOnTheDomeLeadingToIt(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, ctx)
	world := smallWorld()
	world.Objects = []domain.Object{{Kind: domain.ObjectDome, Col: 32, Row: 64, LeadsTo: "doma-1"}}
	_, err := f.repo.InsertMap(ctx, world)
	require.NoError(t, err)
	_, err = f.repo.InsertMap(ctx, domain.GenerateCity(domain.Doma1()))
	require.NoError(t, err)

	city, err := f.maps.Description(ctx, "doma-1")
	require.NoError(t, err)
	assert.Equal(t, &core.Place{Map: "small", Col: 32, Row: 64, Width: 160, Height: 192}, city.StandsOn)
	small, err := f.maps.Description(ctx, "small")
	require.NoError(t, err)
	assert.Nil(t, small.StandsOn)
}

// ADR 0012 test 11: with only the world saved, the boot saves Doma-1
func TestEnsureMapsSavesDoma1BesideTheWorld(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, ctx)
	_, err := f.repo.InsertMap(ctx, domain.Map{Description: domain.Description{Summary: domain.Summary{Slug: "world", Name: "World", Width: 160, Height: 160, Wraps: true}}})
	require.NoError(t, err)

	created, err := f.maps.EnsureMaps(ctx, mapfiles.Slugs(), mapfiles.Read)
	require.NoError(t, err)
	assert.Equal(t, []string{"doma-1"}, created)
	city, err := f.repo.MapBySlug(ctx, "doma-1")
	require.NoError(t, err)
	assert.Equal(t, "Doma-1", city.Name)
}

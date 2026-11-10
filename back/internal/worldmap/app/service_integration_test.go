package app_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
	"donut/internal/worldmap/app"
	"donut/internal/worldmap/domain"
	"donut/internal/worldmap/infra"
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

// ADR 0003 test 11, and ADR 0001 test 7
// DONUT_TEST_DATABASE_URL=postgres://donut:donut@localhost:5434/donut go test ./internal/worldmap/app
func TestEnsureStarterMapCreatesTheWorldOnce(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, ctx)

	created, err := f.maps.EnsureStarterMap(ctx, 42)
	require.NoError(t, err)
	assert.True(t, created)

	created, err = f.maps.EnsureStarterMap(ctx, 42)
	require.NoError(t, err)
	assert.False(t, created, "a second boot creates nothing")

	list, err := f.maps.ListMaps(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, domain.Summary{ID: list[0].ID, Slug: "world", Name: "World", Width: 1024, Height: 512, Wraps: true}, list[0],
		"the list has no contents")

	saved, err := f.repo.MapBySlug(ctx, "world")
	require.NoError(t, err)
	generated := domain.Generate(domain.StarterWorld(), seeded(42))
	assert.Len(t, saved.Chunks, 512)
	assert.Len(t, saved.Regions, 12)
	assert.Len(t, saved.Objects, len(generated.Objects))
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

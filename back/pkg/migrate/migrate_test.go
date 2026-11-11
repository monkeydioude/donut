package migrate_test

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"donut/migrations"
	"donut/pkg/logs"
	"donut/pkg/migrate"
	"donut/pkg/pgtest"
)

func appliedFiles(t *testing.T, ctx context.Context, tx pgx.Tx, schema string) []string {
	t.Helper()
	rows, err := tx.Query(ctx, `SELECT filename FROM `+pgx.Identifier{schema, "schema_migrations"}.Sanitize()+` ORDER BY filename`)
	require.NoError(t, err)
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)
	return names
}

// ADR 0001 test 5
// DONUT_TEST_DATABASE_URL=postgres://donut:donut@localhost:5434/donut go test ./pkg/migrate
func TestApplyRunsEveryMigrationOnce(t *testing.T) {
	ctx := context.Background()
	tx := pgtest.Tx(t, ctx)
	schema := pgtest.NewSchemaName()
	files := pgtest.RenameSchema(t, migrations.Files, migrations.Schema, schema)

	require.NoError(t, migrate.Apply(ctx, tx, files, schema, logs.Discard()))
	assert.Equal(t, []string{"0001.up.sql", "0002.up.sql", "0003.up.sql", "0004.up.sql", "0005.up.sql", "0006.up.sql", "0007.up.sql", "0008.up.sql"}, appliedFiles(t, ctx, tx, schema))
	var maps int
	require.NoError(t, tx.QueryRow(ctx, `SELECT count(*) FROM `+pgx.Identifier{schema, "maps"}.Sanitize()).Scan(&maps))

	require.NoError(t, migrate.Apply(ctx, tx, files, schema, logs.Discard()), "a second run applies nothing, so it doesn't fail on existing tables")
	assert.Equal(t, []string{"0001.up.sql", "0002.up.sql", "0003.up.sql", "0004.up.sql", "0005.up.sql", "0006.up.sql", "0007.up.sql", "0008.up.sql"}, appliedFiles(t, ctx, tx, schema))
}

// ADR 0001 test 6
func TestApplyStopsOnABrokenMigration(t *testing.T) {
	ctx := context.Background()
	tx := pgtest.Tx(t, ctx)
	schema := pgtest.NewSchemaName()
	files := pgtest.RenameSchema(t, migrations.Files, migrations.Schema, schema).(fstest.MapFS)
	files["up/9999.up.sql"] = &fstest.MapFile{Data: []byte(`CREATE TABLE broken (`)}

	err := migrate.Apply(ctx, tx, files, schema, logs.Discard())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "9999.up.sql")
	assert.Equal(t, []string{"0001.up.sql", "0002.up.sql", "0003.up.sql", "0004.up.sql", "0005.up.sql", "0006.up.sql", "0007.up.sql", "0008.up.sql"}, appliedFiles(t, ctx, tx, schema))
}

// ADR 0003 test 10
func TestMigration0002DropsTheOldMaps(t *testing.T) {
	ctx := context.Background()
	tx := pgtest.Tx(t, ctx)
	schema := pgtest.NewSchemaName()
	files := pgtest.RenameSchema(t, migrations.Files, migrations.Schema, schema).(fstest.MapFS)
	before := fstest.MapFS{"init.sql": files["init.sql"], "up/0001.up.sql": files["up/0001.up.sql"]}
	require.NoError(t, migrate.Apply(ctx, tx, before, schema, logs.Discard()))
	table := func(name string) string { return pgx.Identifier{schema, name}.Sanitize() }
	var mapID string
	require.NoError(t, tx.QueryRow(ctx, `INSERT INTO `+table("maps")+` (slug, name, width, height, ground)
		VALUES ('meadow', 'Meadow', 1, 1, '["grass"]') RETURNING id`).Scan(&mapID))
	_, err := tx.Exec(ctx, `INSERT INTO `+table("map_objects")+` (map_id, kind, col, row) VALUES ($1, 'rock', 0, 0)`, mapID)
	require.NoError(t, err)

	require.NoError(t, migrate.Apply(ctx, tx, files, schema, logs.Discard()))

	var maps, objects, groundColumns int
	require.NoError(t, tx.QueryRow(ctx, `SELECT count(*) FROM `+table("maps")).Scan(&maps))
	require.NoError(t, tx.QueryRow(ctx, `SELECT count(*) FROM `+table("map_objects")).Scan(&objects))
	require.NoError(t, tx.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = 'maps' AND column_name = 'ground'`, schema).Scan(&groundColumns))
	assert.Zero(t, maps, "meadow is gone")
	assert.Zero(t, objects, "its rocks with it")
	assert.Zero(t, groundColumns, "and so is the ground column")
}

// ADR 0007 test 22
// ADR 0007 test 22, ADR 0008 test 16, ADR 0009 test 10 and ADR 0011: migrations 0003, 0004, 0005 and 0006 each delete
// every map.
func TestMigrationsDeleteEveryMap(t *testing.T) {
	for _, migration := range []string{"0003", "0004", "0005", "0006"} {
		t.Run(migration, func(t *testing.T) { testMigrationDeletesEveryMap(t, migration) })
	}
}

// testMigrationDeletesEveryMap applies the migrations before the one given, makes a map, then applies them all.
func testMigrationDeletesEveryMap(t *testing.T, migration string) {
	ctx := context.Background()
	tx, schema, files := migratedBefore(t, ctx, migration)
	table := func(name string) string { return pgx.Identifier{schema, name}.Sanitize() }
	var mapID string
	require.NoError(t, tx.QueryRow(ctx, `INSERT INTO `+table("maps")+` (slug, name, width, height, wraps)
		VALUES ('world', 'World', 160, 160, true) RETURNING id`).Scan(&mapID))
	_, err := tx.Exec(ctx, `INSERT INTO `+table("map_regions")+` (map_id, id, name, biome) VALUES ($1, 1, 'One', 'meadow')`, mapID)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, `INSERT INTO `+table("map_objects")+` (map_id, kind, col, row) VALUES ($1, 'rock', 0, 0)`, mapID)
	require.NoError(t, err)

	require.NoError(t, migrate.Apply(ctx, tx, files, schema, logs.Discard()))

	for _, name := range []string{"maps", "map_regions", "map_objects", "map_chunks"} {
		var rows int
		require.NoError(t, tx.QueryRow(ctx, `SELECT count(*) FROM `+table(name)).Scan(&rows))
		assert.Zero(t, rows, "%s is empty", name)
	}
}

// migratedBefore is a transaction with the migrations before the one given applied to a schema of its own, and every
// migration file, to apply them all next.
func migratedBefore(t *testing.T, ctx context.Context, migration string) (pgx.Tx, string, fstest.MapFS) {
	t.Helper()
	tx := pgtest.Tx(t, ctx)
	schema := pgtest.NewSchemaName()
	files := pgtest.RenameSchema(t, migrations.Files, migrations.Schema, schema).(fstest.MapFS)
	before := fstest.MapFS{"init.sql": files["init.sql"]}
	for name, file := range files {
		if strings.HasPrefix(name, "up/") && name < "up/"+migration {
			before[name] = file
		}
	}
	require.NoError(t, migrate.Apply(ctx, tx, before, schema, logs.Discard()))
	return tx, schema, files
}

// ADR 0012 test 10: migration 0007 keeps every map, and only domes lead to a map.
func TestMigration0007AddsWhereDomesLead(t *testing.T) {
	ctx := context.Background()
	tx, schema, files := migratedBefore(t, ctx, "0007")
	table := func(name string) string { return pgx.Identifier{schema, name}.Sanitize() }
	var mapID string
	require.NoError(t, tx.QueryRow(ctx, `INSERT INTO `+table("maps")+` (slug, name, width, height, wraps)
		VALUES ('world', 'World', 160, 160, true) RETURNING id`).Scan(&mapID))
	_, err := tx.Exec(ctx, `INSERT INTO `+table("map_objects")+` (map_id, kind, col, row) VALUES ($1, 'rock', 0, 0)`, mapID)
	require.NoError(t, err)

	require.NoError(t, migrate.Apply(ctx, tx, files, schema, logs.Discard()))

	var maps, rocks int
	require.NoError(t, tx.QueryRow(ctx, `SELECT count(*) FROM `+table("maps")).Scan(&maps))
	require.NoError(t, tx.QueryRow(ctx, `SELECT count(*) FROM `+table("map_objects")+` WHERE leads_to IS NULL`).Scan(&rocks))
	assert.Equal(t, 1, maps, "the map stays")
	assert.Equal(t, 1, rocks, "its rock too, leading nowhere")
	insert := func(kind string, leadsTo any) error {
		_, err := tx.Exec(ctx, `SAVEPOINT object`)
		require.NoError(t, err)
		_, err = tx.Exec(ctx, `INSERT INTO `+table("map_objects")+` (map_id, kind, col, row, leads_to) VALUES ($1, $2, 4, 4, $3)`, mapID, kind, leadsTo)
		_, rollback := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT object`)
		require.NoError(t, rollback)
		return err
	}
	assert.NoError(t, insert("dome", "doma-1"))
	assert.Error(t, insert("dome", nil), "a dome leads somewhere")
	assert.Error(t, insert("rock", "doma-1"), "a rock leads nowhere")
}

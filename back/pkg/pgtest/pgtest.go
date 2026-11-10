// Package pgtest gives integration tests a PostgreSQL schema of their own, with every migration applied,
// inside a transaction rolled back at the end: no data stays. Tests skip without DONUT_TEST_DATABASE_URL.
package pgtest

import (
	"context"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"donut/migrations"
	"donut/pkg/logs"
	"donut/pkg/migrate"
)

// Tx is a transaction on DONUT_TEST_DATABASE_URL, rolled back when the test ends. No migration applied.
func Tx(t *testing.T, ctx context.Context) pgx.Tx {
	t.Helper()
	url := os.Getenv("DONUT_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set DONUT_TEST_DATABASE_URL for PostgreSQL integration tests")
	}
	conn, err := pgx.Connect(ctx, url)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close(ctx) })
	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { tx.Rollback(ctx) })
	return tx
}

// MigratedTx is Tx with the real migrations applied to a schema of its own, set as the search path.
func MigratedTx(t *testing.T, ctx context.Context) pgx.Tx {
	t.Helper()
	tx := Tx(t, ctx)
	schema := NewSchemaName()
	require.NoError(t, migrate.Apply(ctx, tx, RenameSchema(t, migrations.Files, migrations.Schema, schema), schema, logs.Discard()))
	_, err := tx.Exec(ctx, "SET search_path TO "+pgx.Identifier{schema}.Sanitize())
	require.NoError(t, err)
	return tx
}

// NewSchemaName is a schema name no other test uses.
func NewSchemaName() string {
	return "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
}

// RenameSchema copies the migration files with every mention of schema `from` replaced by `to`.
func RenameSchema(t *testing.T, files fs.FS, from, to string) fs.FS {
	t.Helper()
	renamed := fstest.MapFS{}
	err := fs.WalkDir(files, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		sql, err := fs.ReadFile(files, path)
		if err != nil {
			return err
		}
		renamed[path] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(string(sql), from, to))}
		return nil
	})
	require.NoError(t, err)
	return renamed
}

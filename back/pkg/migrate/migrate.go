// Package migrate applies the SQL migrations at boot, like Kabane's scripts/migrate.sh:
// init.sql once on an empty schema, then every up/*.sql not applied yet, in name order.
package migrate

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// lockKey is the Postgres advisory lock taken while migrating: two servers booting together migrate one after the other.
const lockKey = 0x646f6e7574 // "donut"

// Conn is one database session: a connection or a transaction. The advisory lock belongs to it.
type Conn interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Apply runs the migrations of files (init.sql and up/*.sql) on schema. Each file runs in its own
// transaction and is recorded in schema_migrations; the first failure stops and returns its error.
func Apply(ctx context.Context, conn Conn, files fs.FS, schema string, log *slog.Logger) error {
	m := migrator{conn: conn, files: files, schema: schema, log: log.With("component", "migrate", "schema", schema)}
	m.log.Debug("migrate: waiting for the lock")
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockKey); err != nil {
		return fmt.Errorf("migrate: lock: %w", err)
	}
	m.log.Debug("migrate: lock taken")
	defer func() {
		_, err := conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, lockKey)
		m.log.Debug("migrate: lock released", "error", err)
	}()

	if err := m.applyInit(ctx); err != nil {
		return err
	}
	ups, err := fs.Glob(files, "up/*.sql")
	if err != nil {
		return fmt.Errorf("migrate: list: %w", err)
	}
	m.log.Debug("migrate: files found", "count", len(ups), "files", ups)
	for _, name := range ups {
		if err := m.applyOnce(ctx, name); err != nil {
			return err
		}
	}
	m.log.Info("migrate: up to date", "files", len(ups))
	return nil
}

type migrator struct {
	conn   Conn
	files  fs.FS
	schema string
	log    *slog.Logger
}

// applyInit runs init.sql when the schema has no schema_migrations table yet.
func (m migrator) applyInit(ctx context.Context) error {
	var table *string
	if err := m.conn.QueryRow(ctx, `SELECT to_regclass($1)::text`, m.schema+".schema_migrations").Scan(&table); err != nil {
		return fmt.Errorf("migrate: look for schema_migrations: %w", err)
	}
	if table != nil {
		m.log.Debug("migrate: schema exists, init.sql skipped")
		return nil
	}
	m.log.Info("migrate: empty database, applying init.sql")
	return runFile(ctx, m.conn, m.files, "init.sql", nil)
}

// applyOnce runs an up file unless schema_migrations already has it.
func (m migrator) applyOnce(ctx context.Context, path string) error {
	name := path[len("up/"):]
	table := pgx.Identifier{m.schema, "schema_migrations"}.Sanitize()
	var done bool
	if err := m.conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM `+table+` WHERE filename = $1)`, name).Scan(&done); err != nil {
		return fmt.Errorf("migrate: check %s: %w", name, err)
	}
	if done {
		m.log.Debug("migrate: already applied", "file", name)
		return nil
	}
	m.log.Info("migrate: applying", "file", name)
	return runFile(ctx, m.conn, m.files, path, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO `+table+` (filename) VALUES ($1)`, name)
		return err
	})
}

// runFile executes a file's SQL in a transaction, then record (when not nil) in the same transaction.
func runFile(ctx context.Context, conn Conn, files fs.FS, path string, record func(pgx.Tx) error) error {
	sql, err := fs.ReadFile(files, path)
	if err != nil {
		return fmt.Errorf("migrate: read %s: %w", path, err)
	}
	err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			return err
		}
		if record == nil {
			return nil
		}
		return record(tx)
	})
	if err != nil {
		return fmt.Errorf("migrate: %s: %w", path, err)
	}
	return nil
}

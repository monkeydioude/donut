// Package migrations holds the SQL migrations, built into the binary so the server applies them at boot.
package migrations

import "embed"

// Schema is the Postgres schema the migrations create and fill.
const Schema = "donut"

// Files are init.sql and up/*.sql.
//
//go:embed init.sql up/*.sql
var Files embed.FS

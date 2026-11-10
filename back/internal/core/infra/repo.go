// Package infra holds the base repository shared by modules.
package infra

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Requester is anything that can run SQL: a pool or a transaction.
type Requester interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Repository is embedded by every module repo. It holds the Requester.
type Repository struct {
	DB Requester
}

// RunInTransaction runs fn inside a transaction. fn gets a Requester bound to it.
// Nesting works: pgx turns an inner Begin into a savepoint.
func (r *Repository) RunInTransaction(ctx context.Context, fn func(context.Context, Requester) error) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("RunInTransaction: begin: %w", err)
	}
	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			err = errors.Join(err, fmt.Errorf("RunInTransaction: rollback: %w", rbErr))
		}
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("RunInTransaction: commit: %w", err)
	}
	return nil
}

// Package pg holds small helpers around pgx.
package pg

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"donut/pkg/errd"
)

// Wrap prefixes err. A pgx "no rows" becomes errd.ErrNotFound, a unique violation errd.ErrConflict.
func Wrap(err error, prefix string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", prefix, errd.ErrNotFound)
	}
	if IsUniqueViolation(err) {
		return fmt.Errorf("%s: %w", prefix, errd.ErrConflict)
	}
	return fmt.Errorf("%s: %w", prefix, err)
}

// IsUniqueViolation tells if err is a Postgres unique constraint error.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// CollectOne scans the only row into a T. No row is errd.ErrNotFound.
func CollectOne[T any](rows pgx.Rows, err error, prefix string) (T, error) {
	var zero T
	if err != nil {
		return zero, Wrap(err, prefix)
	}
	res, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[T])
	if err != nil {
		return zero, Wrap(err, prefix)
	}
	return res, nil
}

// CollectMany scans every row into a slice of T. The slice is never nil.
func CollectMany[T any](rows pgx.Rows, err error, prefix string) ([]T, error) {
	if err != nil {
		return nil, Wrap(err, prefix)
	}
	res, err := pgx.CollectRows(rows, pgx.RowToStructByName[T])
	if err != nil {
		return nil, Wrap(err, prefix)
	}
	if res == nil {
		res = []T{}
	}
	return res, nil
}

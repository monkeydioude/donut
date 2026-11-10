// Package errd holds the sentinel errors shared by every layer.
// Transport maps them to HTTP status codes.
package errd

import "errors"

var (
	ErrNotFound   = errors.New("not found")
	ErrBadRequest = errors.New("bad request")
	ErrConflict   = errors.New("conflict")
)

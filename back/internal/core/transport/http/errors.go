// Package corehttp holds what every module's HTTP layer shares.
package corehttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/calqs/gopkg/router/response"

	"donut/pkg/errd"
)

// MapServiceErrors turns a service error into its HTTP status. Unknown errors are logged and hidden.
func MapServiceErrors(ctx context.Context, err error) response.Response {
	switch {
	case errors.Is(err, errd.ErrNotFound):
		return response.NotFound("not found", err)
	case errors.Is(err, errd.ErrBadRequest):
		return response.BadRequest("bad request", err)
	case errors.Is(err, errd.ErrConflict):
		return response.Conflict("conflict", err)
	default:
		slog.ErrorContext(ctx, "service error", "error", err.Error())
		return response.InternalServerError("internal server error", err)
	}
}

// PanicRecovery answers 500 instead of crashing on a handler's panic.
func PanicRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.ErrorContext(r.Context(), "panic", "error", err, "stack", string(debug.Stack()))
				response.InternalServerError("internal server error").Send(w)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

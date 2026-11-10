// Package presence is the players module: who is on a map and where they walk, live, over a websocket.
package presence

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"donut/internal/presence/app"
	wstransport "donut/internal/presence/transport/ws"
)

type Module struct {
	Hub     *app.Hub
	Handler *wstransport.Handler
}

// NewModule takes the context the rooms tick in, how to load a map for its room, the clock,
// the origins allowed to open the websocket, and the logger.
func NewModule(ctx context.Context, load app.LoadWorld, now func() time.Time, originPatterns []string, log *slog.Logger) *Module {
	hub := app.NewHub(ctx, load, now, log)
	return &Module{Hub: hub, Handler: wstransport.NewHandler(hub, originPatterns, log)}
}

// Routes registers the websocket on a plain mux: the API router wraps the ResponseWriter,
// which then cannot be taken over by the websocket upgrade.
func (m *Module) Routes(mux *http.ServeMux) {
	mux.Handle("GET /api/maps/{slug}/ws", m.Handler)
}

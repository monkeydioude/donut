package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/calqs/gopkg/router/router"

	core "donut/internal/core/infra"
	corehttp "donut/internal/core/transport/http"
	"donut/internal/presence"
	presenceapp "donut/internal/presence/app"
	"donut/internal/worldmap"
)

// homeMap is the map new guests wake up in, and where a guest whose place is gone goes (ADR 0012).
const homeMap = "doma-1"

// routing is the composition root: the only place that wires every module.
// It also gives the maps module, which the boot uses to create the starter map.
func routing(ctx context.Context, db core.Requester, config Config, log *slog.Logger) (http.Handler, *worldmap.Module, error) {
	origins, err := config.OriginPatterns()
	if err != nil {
		return nil, nil, err
	}
	maps := worldmap.NewModule(db, log)
	players := presence.NewModule(ctx, func(ctx context.Context, slug string) (presenceapp.World, error) {
		return maps.Service.World(ctx, slug)
	}, time.Now, homeMap, origins, log)

	api := router.NewRouter(ctx, router.WithBaseURL("/api"))
	api.Use(corehttp.PanicRecovery)
	maps.Routes(api)

	mux := http.NewServeMux()
	players.Routes(mux)
	mux.Handle("/", api)
	return corehttp.LogRequests(log)(mux), maps, nil
}

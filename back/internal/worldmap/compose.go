// Package worldmap is the maps module: the maps, their ground and objects layers, the starter map.
package worldmap

import (
	"log/slog"

	"github.com/calqs/gopkg/router/router"

	core "donut/internal/core/infra"
	"donut/internal/worldmap/app"
	"donut/internal/worldmap/infra"
	httptransport "donut/internal/worldmap/transport/http"
)

type Module struct {
	Service *app.Service
	Handler *httptransport.Handler
}

func NewModule(db core.Requester, log *slog.Logger) *Module {
	svc := app.NewService(infra.NewRepo(db), log)
	return &Module{Service: svc, Handler: httptransport.NewHandler(svc)}
}

func (m *Module) Routes(api *router.Router) {
	httptransport.Routes(m.Handler)(api)
}

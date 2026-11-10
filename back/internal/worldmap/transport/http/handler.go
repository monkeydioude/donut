// Package httptransport is the worldmap module HTTP layer.
package httptransport

import (
	"context"

	"github.com/calqs/gopkg/dt"
	"github.com/calqs/gopkg/router/handler"
	"github.com/calqs/gopkg/router/jsonresponse"
	method "github.com/calqs/gopkg/router/public"
	"github.com/calqs/gopkg/router/response"
	"github.com/calqs/gopkg/router/router"

	core "donut/internal/core/domain"
	corehttp "donut/internal/core/transport/http"
	"donut/internal/worldmap/domain"
)

// MapReader reads maps for the routes.
type MapReader interface {
	ListMaps(ctx context.Context) ([]domain.Summary, error)
	Description(ctx context.Context, slug string) (domain.Description, error)
}

type Handler struct {
	maps MapReader
}

func NewHandler(maps MapReader) *Handler {
	return &Handler{maps: maps}
}

func Routes(h *Handler) func(api *router.Router) {
	return func(api *router.Router) {
		api.Handle("/maps", method.Get(h.ListMaps))
		api.Handle("/maps/{slug}", method.Get(h.GetMap))
	}
}

type SummaryOut struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Wraps  bool   `json:"wraps"`
}

type RegionOut struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Biome string `json:"biome"`
}

// DescriptionOut is a map without its cells: these come over the websocket, chunk by chunk (ADR 0003).
type DescriptionOut struct {
	SummaryOut
	ChunkSize int         `json:"chunkSize"`
	Regions   []RegionOut `json:"regions"`
}

func (h *Handler) ListMaps(req *handler.Request[handler.None]) response.Response {
	maps, err := h.maps.ListMaps(req.Request.Context())
	if err != nil {
		return corehttp.MapServiceErrors(req.Request.Context(), err)
	}
	return jsonresponse.StatusOk(dt.SliceTransform(maps, toSummaryOut))
}

func (h *Handler) GetMap(req *handler.Request[handler.None]) response.Response {
	d, err := h.maps.Description(req.Request.Context(), req.Request.PathValue("slug"))
	if err != nil {
		return corehttp.MapServiceErrors(req.Request.Context(), err)
	}
	return jsonresponse.StatusOk(toDescriptionOut(d))
}

func toSummaryOut(s domain.Summary) SummaryOut {
	return SummaryOut{ID: s.ID, Slug: s.Slug, Name: s.Name, Width: s.Width, Height: s.Height, Wraps: s.Wraps}
}

func toDescriptionOut(d domain.Description) DescriptionOut {
	regions := dt.SliceTransform(d.Regions, func(r domain.Region) RegionOut {
		return RegionOut{ID: r.ID, Name: r.Name, Biome: string(r.Biome)}
	})
	return DescriptionOut{SummaryOut: toSummaryOut(d.Summary), ChunkSize: core.ChunkSize, Regions: regions}
}

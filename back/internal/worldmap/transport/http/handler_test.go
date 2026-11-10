package httptransport_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/calqs/gopkg/router/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"donut/internal/worldmap/domain"
	httptransport "donut/internal/worldmap/transport/http"
	"donut/pkg/errd"
)

// fakeMaps holds one map, "world".
type fakeMaps struct{}

func (fakeMaps) ListMaps(context.Context) ([]domain.Summary, error) {
	return []domain.Summary{world().Summary}, nil
}

func (fakeMaps) Description(_ context.Context, slug string) (domain.Description, error) {
	if slug != "world" {
		return domain.Description{}, fmt.Errorf("Description: %w", errd.ErrNotFound)
	}
	return world(), nil
}

func world() domain.Description {
	return domain.Description{
		Summary: domain.Summary{ID: "m1", Slug: "world", Name: "World", Width: 1024, Height: 512, Wraps: true},
		Regions: []domain.Region{{ID: 1, Name: "Sprinkle Meadow", Biome: domain.BiomeMeadow}},
	}
}

func serve(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	api := router.NewRouter(context.Background(), router.WithBaseURL("/api"))
	httptransport.Routes(httptransport.NewHandler(fakeMaps{}))(api)
	recorder := httptest.NewRecorder()
	api.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder
}

// ADR 0003 test 13, and ADR 0001 test 10
func TestGetMap(t *testing.T) {
	t.Run("answers the map's description, without its cells", func(t *testing.T) {
		res := serve(t, "/api/maps/world")
		require.Equal(t, http.StatusOK, res.Code)
		assert.JSONEq(t, `{
			"id": "m1", "slug": "world", "name": "World", "width": 1024, "height": 512, "wraps": true,
			"chunkSize": 32,
			"regions": [{"id": 1, "name": "Sprinkle Meadow", "biome": "meadow"}]
		}`, res.Body.String())
	})

	t.Run("answers 404 for an unknown map", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, serve(t, "/api/maps/nowhere").Code)
	})
}

func TestListMaps(t *testing.T) {
	res := serve(t, "/api/maps")
	require.Equal(t, http.StatusOK, res.Code)
	assert.JSONEq(t, `[{"id": "m1", "slug": "world", "name": "World", "width": 1024, "height": 512, "wraps": true}]`, res.Body.String())
}

package httptransport_test

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/calqs/gopkg/router/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"donut/internal/worldmap/app"
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

// worldETag is the ETag of the fake world's image.
const worldETag = `"0123456789abcdef"`

func (fakeMaps) Image(_ context.Context, slug string) (app.MapImage, error) {
	if slug != "world" {
		return app.MapImage{}, fmt.Errorf("Image: %w", errd.ErrNotFound)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 1024, 512))); err != nil {
		return app.MapImage{}, err
	}
	return app.MapImage{PNG: buf.Bytes(), ETag: worldETag}, nil
}

func world() domain.Description {
	return domain.Description{
		Summary: domain.Summary{ID: "m1", Slug: "world", Name: "World", Width: 1024, Height: 512, Wraps: true},
		Regions: []domain.Region{{ID: 1, Name: "Sprinkle Meadow", Biome: domain.BiomeMeadow}},
	}
}

func serve(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	return serveWith(t, httptest.NewRequest(http.MethodGet, path, nil))
}

func serveWith(t *testing.T, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	api := router.NewRouter(context.Background(), router.WithBaseURL("/api"))
	httptransport.Routes(httptransport.NewHandler(fakeMaps{}))(api)
	recorder := httptest.NewRecorder()
	api.ServeHTTP(recorder, req)
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

// imageRequest asks for the world's image, with an If-None-Match header when given one.
func imageRequest(ifNoneMatch string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/maps/world/image.png", nil)
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}
	return req
}

// ADR 0005 test 6
func TestGetImage(t *testing.T) {
	res := serveWith(t, imageRequest(""))
	require.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "image/png", res.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", res.Header().Get("Cache-Control"))
	assert.Equal(t, worldETag, res.Header().Get("ETag"))
	assert.Equal(t, "nosniff", res.Header().Get("X-Content-Type-Options"))

	decoded, err := png.Decode(res.Body)
	require.NoError(t, err)
	assert.Equal(t, image.Rect(0, 0, 1024, 512), decoded.Bounds())
}

// ADR 0005 test 7
func TestGetImageNotModified(t *testing.T) {
	for _, header := range []string{worldETag, "W/" + worldETag, `"other", ` + worldETag, "*"} {
		res := serveWith(t, imageRequest(header))
		assert.Equal(t, http.StatusNotModified, res.Code, header)
		assert.Empty(t, res.Body.Bytes(), header)
		assert.Equal(t, worldETag, res.Header().Get("ETag"), header)
	}

	t.Run("another ETag gets the image", func(t *testing.T) {
		res := serveWith(t, imageRequest(`"fedcba9876543210"`))
		assert.Equal(t, http.StatusOK, res.Code)
		assert.NotEmpty(t, res.Body.Bytes())
	})
}

// ADR 0005 test 8
func TestGetImageOfAnUnknownMap(t *testing.T) {
	assert.Equal(t, http.StatusNotFound, serve(t, "/api/maps/nowhere/image.png").Code)
}

package app_test

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"donut/internal/worldmap/app"
	"donut/internal/worldmap/domain"
	"donut/pkg/errd"
	"donut/pkg/logs"
)

// oneMapStore holds one map, "small", and counts how often it's read whole.
type oneMapStore struct {
	m     domain.Map
	reads int
}

func (s *oneMapStore) MapBySlug(_ context.Context, slug string) (domain.Map, error) {
	if slug != s.m.Slug {
		return domain.Map{}, fmt.Errorf("MapBySlug: %w", errd.ErrNotFound)
	}
	s.reads++
	return s.m, nil
}

func (s *oneMapStore) ListMaps(context.Context) ([]domain.Summary, error) { return nil, nil }
func (s *oneMapStore) CountMaps(context.Context) (int, error)             { return 1, nil }
func (s *oneMapStore) DescriptionBySlug(context.Context, string) (domain.Description, error) {
	return s.m.Description, nil
}
func (s *oneMapStore) InsertMap(context.Context, domain.Map) (string, error) { return "", nil }
func (s *oneMapStore) ReplaceMap(context.Context, domain.Map) (string, bool, error) {
	return "", false, nil
}

// ADR 0005 test 5
func TestImageIsRenderedOnceThenKept(t *testing.T) {
	ctx := context.Background()
	store := &oneMapStore{m: smallWorld()}
	maps := app.NewService(store, logs.Discard())

	first, err := maps.Image(ctx, "small")
	require.NoError(t, err)
	second, err := maps.Image(ctx, "small")
	require.NoError(t, err)

	assert.Equal(t, 1, store.reads, "the second request is served from memory")
	assert.Equal(t, first.ETag, second.ETag)
	assert.Regexp(t, `^"[0-9a-f]{16}"$`, first.ETag)

	decoded, err := png.Decode(bytes.NewReader(first.PNG))
	require.NoError(t, err)
	assert.Equal(t, 160, decoded.Bounds().Dx())
	assert.Equal(t, 192, decoded.Bounds().Dy())
}

// ADR 0005 test 5, a burst of first requests
func TestImageRendersOnceForABurstOfFirstRequests(t *testing.T) {
	store := &oneMapStore{m: smallWorld()}
	maps := app.NewService(store, logs.Discard())
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			_, err := maps.Image(context.Background(), "small")
			assert.NoError(t, err)
		})
	}
	wg.Wait()
	assert.Equal(t, 1, store.reads)
}

// ADR 0005 test 8, the service's side
func TestImageOfAnUnknownMap(t *testing.T) {
	maps := app.NewService(&oneMapStore{m: smallWorld()}, logs.Discard())
	_, err := maps.Image(context.Background(), "nowhere")
	assert.ErrorIs(t, err, errd.ErrNotFound)
}

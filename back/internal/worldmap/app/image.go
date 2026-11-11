package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/png"
	"sync"
	"time"

	"donut/internal/worldmap/domain"
)

// MapImage is a map's picture as a PNG (ADR 0005), and its ETag: a hash of the PNG, quoted.
type MapImage struct {
	PNG  []byte
	ETag string
}

// imageCache keeps each map's image once rendered: chunks don't change live yet. Renders go one at a time, so a
// burst of first requests renders once and the others wait for it.
type imageCache struct {
	mu        sync.Mutex
	rendering sync.Mutex
	bySlug    map[string]MapImage
}

func (c *imageCache) get(slug string) (MapImage, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	img, ok := c.bySlug[slug]
	return img, ok
}

func (c *imageCache) put(slug string, img MapImage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bySlug[slug] = img
}

// Image is the map's picture, rendered the first time it's asked for, then kept in memory. Only maps that exist are
// kept. An unknown slug is errd.ErrNotFound.
func (s *Service) Image(ctx context.Context, slug string) (MapImage, error) {
	if img, ok := s.images.get(slug); ok {
		s.log.DebugContext(ctx, "map image served from memory", "slug", slug, "bytes", len(img.PNG))
		return img, nil
	}
	s.images.rendering.Lock()
	defer s.images.rendering.Unlock()
	if img, ok := s.images.get(slug); ok {
		s.log.DebugContext(ctx, "map image rendered meanwhile, served from memory", "slug", slug)
		return img, nil
	}
	return s.renderImage(ctx, slug)
}

func (s *Service) renderImage(ctx context.Context, slug string) (MapImage, error) {
	start := time.Now()
	m, err := s.store.MapBySlug(ctx, slug)
	if err != nil {
		s.log.DebugContext(ctx, "map image not rendered", "slug", slug, "error", err)
		return MapImage{}, fmt.Errorf("Image: %w", err)
	}
	img, err := encodePNG(domain.Image(m))
	if err != nil {
		return MapImage{}, fmt.Errorf("Image: %w", err)
	}
	s.images.put(slug, img)
	s.log.DebugContext(ctx, "map image rendered", "slug", slug, "size", fmt.Sprintf("%dx%d", m.Width, m.Height),
		"bytes", len(img.PNG), "took", time.Since(start))
	return img, nil
}

func encodePNG(img image.Image) (MapImage, error) {
	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&buf, img); err != nil {
		return MapImage{}, fmt.Errorf("encodePNG: %w", err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return MapImage{PNG: buf.Bytes(), ETag: `"` + hex.EncodeToString(sum[:8]) + `"`}, nil
}

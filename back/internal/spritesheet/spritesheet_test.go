package spritesheet_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"donut/internal/spritesheet"
)

var (
	outline = color.NRGBA{R: 0x3f, G: 0x35, B: 0x50, A: 0xff}
	quiet   = slog.New(slog.NewTextHandler(io.Discard, nil))
)

// filled is a w × h sprite of one color.
func filled(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

// ADR 0006 test 9
func TestNearBlackBecomesTheOutline(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 4, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 3, G: 2, B: 13, A: 255})
	src.SetNRGBA(1, 0, color.NRGBA{R: 40, G: 40, B: 40, A: 255})
	src.SetNRGBA(2, 0, color.NRGBA{R: 10, G: 41, B: 10, A: 255})
	src.SetNRGBA(3, 0, color.NRGBA{})

	out := spritesheet.Recolor(src, spritesheet.DonutStyle())

	assert.Equal(t, outline, out.NRGBAAt(0, 0))
	assert.Equal(t, outline, out.NRGBAAt(1, 0), "40 is still near-black")
	assert.Equal(t, color.NRGBA{R: 10, G: 41, B: 10, A: 255}, out.NRGBAAt(2, 0), "one channel at 41: it stays")
	assert.Equal(t, color.NRGBA{}, out.NRGBAAt(3, 0), "transparent stays transparent")
}

// ADR 0006 test 10
func TestAlphaIsSnapped(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 200, G: 100, B: 50, A: 127})
	src.SetNRGBA(1, 0, color.NRGBA{R: 200, G: 100, B: 50, A: 128})

	out := spritesheet.Recolor(src, spritesheet.DonutStyle())

	assert.Equal(t, color.NRGBA{}, out.NRGBAAt(0, 0))
	assert.Equal(t, color.NRGBA{R: 200, G: 100, B: 50, A: 255}, out.NRGBAAt(1, 0))
}

// fakeAtlas gives, for each asked row_id, the row_id it finds and its sprite.
type fakeAtlas map[int]struct {
	foundID int
	img     image.Image
}

func (f fakeAtlas) Row(_ context.Context, rowID int) (int, image.Image, error) {
	row, ok := f[rowID]
	if !ok {
		return 0, nil, errors.New("no such row")
	}
	return row.foundID, row.img, nil
}

var red, blue = color.NRGBA{R: 220, G: 40, B: 40, A: 255}, color.NRGBA{R: 40, G: 40, B: 220, A: 255}

func twoPicks() spritesheet.Picks {
	return spritesheet.Picks{Dataset: "d", Config: "default", Split: "train", Sprites: []spritesheet.Pick{
		{Slug: "dome-head", RowID: 70, SHA256: spritesheet.PixelHash(filled(16, 16, red))},
		{Slug: "rusty-toy", RowID: 30, SHA256: spritesheet.PixelHash(filled(16, 16, blue))},
	}}
}

func TestPixelHash(t *testing.T) {
	one := filled(1, 1, color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	want := sha256.Sum256([]byte{1, 2, 3, 255})
	assert.Equal(t, hex.EncodeToString(want[:]), spritesheet.PixelHash(one), "R, G, B, A of each pixel, row by row")

	clear := filled(16, 16, color.NRGBA{})
	tinted := filled(16, 16, color.NRGBA{R: 9, G: 9, B: 9, A: 10})
	assert.Equal(t, spritesheet.PixelHash(clear), spritesheet.PixelHash(tinted), "transparent is transparent, whatever its color")
	assert.NotEqual(t, spritesheet.PixelHash(filled(16, 16, red)), spritesheet.PixelHash(filled(16, 16, blue)))
}

// ADR 0006 test 16
func TestOtherPixelsThanPickedStopIt(t *testing.T) {
	atlas := fakeAtlas{70: {70, filled(16, 16, red)}, 30: {30, filled(16, 16, red)}}

	_, _, err := spritesheet.Make(context.Background(), twoPicks(), atlas, spritesheet.DonutStyle(), quiet)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "rusty-toy")
	assert.Contains(t, err.Error(), "other pixels")
}

// ADR 0006 test 11
func TestTheSheetIsOneRowInThePicksOrder(t *testing.T) {
	atlas := fakeAtlas{70: {70, filled(16, 16, red)}, 30: {30, filled(16, 16, blue)}}

	sheet, manifest, err := spritesheet.Make(context.Background(), twoPicks(), atlas, spritesheet.DonutStyle(), quiet)

	require.NoError(t, err)
	assert.Equal(t, image.Rect(0, 0, 32, 16), sheet.Bounds())
	assert.Equal(t, red, sheet.NRGBAAt(0, 0))
	assert.Equal(t, red, sheet.NRGBAAt(15, 15))
	assert.Equal(t, blue, sheet.NRGBAAt(16, 0))
	assert.Equal(t, spritesheet.Manifest{Cell: 16, Sprites: []spritesheet.Placed{
		{Slug: "dome-head", X: 0, Y: 0, RowID: 70},
		{Slug: "rusty-toy", X: 16, Y: 0, RowID: 30},
	}}, manifest)
}

// ADR 0006 test 12
func TestASpriteNotSixteenBySixteenStopsIt(t *testing.T) {
	atlas := fakeAtlas{70: {70, filled(16, 16, red)}, 30: {30, filled(16, 17, blue)}}

	_, _, err := spritesheet.Make(context.Background(), twoPicks(), atlas, spritesheet.DonutStyle(), quiet)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "rusty-toy")
	assert.Contains(t, err.Error(), "16x17")
}

// ADR 0006 test 13
func TestARowThatChangedStopsIt(t *testing.T) {
	atlas := fakeAtlas{70: {71, filled(16, 16, red)}, 30: {30, filled(16, 16, blue)}}

	_, _, err := spritesheet.Make(context.Background(), twoPicks(), atlas, spritesheet.DonutStyle(), quiet)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "dome-head")
	assert.Contains(t, err.Error(), "the dataset changed")
}

// fakeAPI answers a row whose sprite is on the same server, failing the first `failures` requests.
func fakeAPI(t *testing.T, failures int) (*httptest.Server, *int) {
	t.Helper()
	var pngBytes bytes.Buffer
	require.NoError(t, png.Encode(&pngBytes, filled(16, 16, red)))
	requests := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests <= failures {
			http.Error(w, "busy", http.StatusTooManyRequests)
			return
		}
		switch r.URL.Path {
		case "/filter":
			assert.Equal(t, `"row_id"=70`, r.URL.Query().Get("where"))
			fmt.Fprintf(w, `{"rows": [{"row_idx": 7, "row": {"row_id": 70, "image": {"src": %q}}}]}`, server.URL+"/sprite.png")
		case "/sprite.png":
			w.Write(pngBytes.Bytes())
		}
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

func newAtlas(t *testing.T, server *httptest.Server) *spritesheet.HTTPAtlas {
	t.Helper()
	noWait := func(context.Context, time.Duration) error { return nil }
	options := spritesheet.AtlasOptions{Tries: 3, Sleep: noWait}
	atlas, err := spritesheet.NewHTTPAtlas(server.Client(), server.URL, twoPicks(), options, quiet)
	require.NoError(t, err)
	return atlas
}

// ADR 0006 test 14
func TestAFailedRequestIsTriedAgain(t *testing.T) {
	server, requests := fakeAPI(t, 2)

	foundID, img, err := newAtlas(t, server).Row(context.Background(), 70)

	require.NoError(t, err)
	assert.Equal(t, 70, foundID)
	assert.Equal(t, image.Rect(0, 0, 16, 16), img.Bounds())
	assert.Equal(t, 4, *requests, "two failures, the row, the sprite")
}

// ADR 0006 test 14
func TestThreeFailuresStopIt(t *testing.T) {
	server, requests := fakeAPI(t, 3)

	_, _, err := newAtlas(t, server).Row(context.Background(), 70)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "after 3 tries")
	assert.Equal(t, 3, *requests)
}

func TestASpriteOnAnotherHostIsRefused(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"rows": [{"row": {"row_id": 70, "image": {"src": "https://evil.example/sprite.png"}}}]}`)
	}))
	t.Cleanup(server.Close)

	_, _, err := newAtlas(t, server).Row(context.Background(), 70)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not on")
}

// ADR 0006 test 15
func TestBadPicksAreRefused(t *testing.T) {
	picks := func(sprites string) []byte {
		return []byte(`{"dataset": "d", "config": "default", "split": "train", "sprites": [` + sprites + `]}`)
	}
	pin := strings.Repeat("ab", 32)
	good, err := spritesheet.ParsePicks(picks(`{"slug": "dome-head", "rowId": 70, "sha256": "` + pin + `"}`))
	require.NoError(t, err)
	assert.Equal(t, []spritesheet.Pick{{Slug: "dome-head", RowID: 70, SHA256: pin}}, good.Sprites)

	for name, bad := range map[string][]byte{
		"no sprites":      picks(``),
		"a twice":         picks(`{"slug": "a", "rowId": 1, "sha256": "` + pin + `"}, {"slug": "a", "rowId": 2, "sha256": "` + pin + `"}`),
		"uppercase":       picks(`{"slug": "Dome", "rowId": 1, "sha256": "` + pin + `"}`),
		"empty slug":      picks(`{"slug": "", "rowId": 1, "sha256": "` + pin + `"}`),
		"too long":        picks(`{"slug": "` + strings.Repeat("a", 33) + `", "rowId": 1, "sha256": "` + pin + `"}`),
		"negative row id": picks(`{"slug": "a", "rowId": -1, "sha256": "` + pin + `"}`),
		"no dataset":      []byte(`{"config": "default", "split": "train", "sprites": [{"slug": "a", "rowId": 1}]}`),
		"not JSON":        []byte(`nope`),
	} {
		_, err := spritesheet.ParsePicks(bad)
		assert.Error(t, err, name)
	}
}

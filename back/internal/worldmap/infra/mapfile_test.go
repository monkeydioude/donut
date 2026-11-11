package infra_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand/v2"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
	"donut/internal/worldmap/domain"
	"donut/internal/worldmap/infra"
)

// smallTerrain is a wrapping 160 x 160 map with every kind of ground, heights, 3 regions and rocks.
func smallTerrain() domain.Map {
	opts := domain.GenerateOptions{
		Slug: "small", Name: "Small", Width: 160, Height: 160, Wraps: true, RockChance: 0.1,
		RegionNames: []string{"One", "Two", "Three"},
		Relief:      domain.ReliefOptions{Wavelengths: []int{32, 16}, Shares: [7]float64{0.05, 0.1, 0.15, 0.4, 0.15, 0.1, 0.05}},
		Landmasses:  domain.LandmassOptions{Continents: 1, ContinentRadius: 60, Coast: 0.3, CoastWavelengths: []int{32, 16}, InlandRise: 0.6, InlandReach: 20, RegionsPerContinent: 2, OceanRegions: 1},
		Water: domain.WaterOptions{RiversPerContinent: domain.Span{Min: 2, Max: 2}, LakesPerContinent: domain.Span{Min: 1, Max: 1}, LakeSize: domain.Span{Min: 4, Max: 8},
			LakeFromOcean: 10, LakeFromWater: 6, Wobble: 0.125, RiverTries: 5, Spawn: domain.SpawnRiver{Near: core.Cell{Col: 79, Row: 79}, West: domain.Span{Min: 20, Max: 25}, Above: domain.Span{Min: 8, Max: 10}, Wobble: 3},
			Bridges: domain.BridgeOptions{Every: 20, MaxSpan: 3}},
		Holes:     domain.HoleOptions{PerChunk: 1, Size: domain.Span{Min: 1, Max: 4}, FromWater: 3},
		Mountains: domain.MountainOptions{Level: 2, MinSummit: 3, Cover: 0.6, Size: domain.Span{Min: 2, Max: 6}},
		Glaciers:  domain.GlacierOptions{HalfWidth: 20, PerChunk: 3, Size: domain.Span{Min: 4, Max: 12}, Clearance: 10, Spawn: core.Cell{Col: 79, Row: 79}},
	}
	return domain.Generate(opts, rand.New(rand.NewPCG(9, 9)))
}

func encoded(t *testing.T, m domain.Map) fstest.MapFS {
	t.Helper()
	file, err := infra.EncodeMapFile(m)
	require.NoError(t, err)
	folder := fstest.MapFS{}
	for name, data := range file {
		folder[name] = &fstest.MapFile{Data: data}
	}
	return folder
}

// paint sets one pixel of a PNG of the folder.
func paint(t *testing.T, folder fstest.MapFS, name string, x, y int, c color.NRGBA) {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(folder[name].Data))
	require.NoError(t, err)
	rgba := image.NewNRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{}, draw.Src)
	rgba.SetNRGBA(x, y, c)
	var out bytes.Buffer
	require.NoError(t, png.Encode(&out, rgba))
	folder[name] = &fstest.MapFile{Data: out.Bytes()}
}

// editManifest changes map.json as plain JSON.
func editManifest(t *testing.T, folder fstest.MapFS, edit func(map[string]any)) {
	t.Helper()
	var man map[string]any
	require.NoError(t, json.Unmarshal(folder["map.json"].Data, &man))
	edit(man)
	data, err := json.Marshal(man)
	require.NoError(t, err)
	folder["map.json"] = &fstest.MapFile{Data: data}
}

func gray(v uint8) color.NRGBA { return color.NRGBA{R: v, G: v, B: v, A: 0xff} }

// ADR 0007 test 14
func TestMapFileReadsBackTheSame(t *testing.T) {
	m := smallTerrain()
	kinds := map[domain.TileKind]bool{}
	for _, chunk := range m.Chunks {
		for _, kind := range chunk.Ground {
			kinds[kind] = true
		}
	}
	require.Len(t, kinds, 7, "grass, water, holes, bridges, the ocean, mountains and glaciers to read back")

	folder := encoded(t, m)
	read, err := infra.DecodeMapFile(folder)
	require.NoError(t, err)
	assert.Equal(t, m.Description, read.Description, "name, size, wrapping and regions")
	assert.Equal(t, m.Chunks, read.Chunks, "ground, heights and regions of every cell")
	assert.Equal(t, m.Objects, read.Objects, "rocks, row by row")

	for _, name := range []string{"map.json", "ground.png", "height.png", "regions.png", "objects.png", "README.md", "palette.gpl"} {
		assert.Contains(t, folder, name)
	}
	assert.Contains(t, string(folder["README.md"].Data), "`#6fb7e0` | water")
	assert.Contains(t, string(folder["README.md"].Data), "`#8a8694` | mountain", "ADR 0009")
	assert.Contains(t, string(folder["README.md"].Data), "`#dff4fb` | glacier", "ADR 0009")
}

// ADR 0007 test 15
func TestMapFileRefuses(t *testing.T) {
	cases := map[string]struct {
		breakIt func(t *testing.T, folder fstest.MapFS)
		says    string
	}{
		"an unknown format": {func(t *testing.T, f fstest.MapFS) { editManifest(t, f, func(m map[string]any) { m["format"] = 2 }) },
			"map.json: format 2"},
		"a PNG of the wrong size": {func(t *testing.T, f fstest.MapFS) {
			var out bytes.Buffer
			require.NoError(t, png.Encode(&out, image.NewNRGBA(image.Rect(0, 0, 32, 32))))
			f["height.png"] = &fstest.MapFile{Data: out.Bytes()}
		}, "height.png is 32 x 32, the map is 160 x 160"},
		"an unknown ground color":   {func(t *testing.T, f fstest.MapFS) { paint(t, f, "ground.png", 3, 4, gray(1)) }, "ground.png (3, 4): #010101 isn't a ground kind"},
		"a gray that isn't a level": {func(t *testing.T, f fstest.MapFS) { paint(t, f, "height.png", 3, 4, gray(0x8c)) }, "height.png (3, 4): #8c8c8c isn't a height"},
		"a transparent height":      {func(t *testing.T, f fstest.MapFS) { paint(t, f, "height.png", 3, 4, color.NRGBA{}) }, "height.png (3, 4): transparent isn't a height"},
		"a color of no region":      {func(t *testing.T, f fstest.MapFS) { paint(t, f, "regions.png", 3, 4, gray(1)) }, "regions.png (3, 4): #010101 isn't a region's color"},
		"an unknown object color":   {func(t *testing.T, f fstest.MapFS) { paint(t, f, "objects.png", 3, 4, gray(1)) }, "objects.png (3, 4): #010101 isn't an object"},
		"a half-transparent pixel": {func(t *testing.T, f fstest.MapFS) {
			paint(t, f, "ground.png", 3, 4, color.NRGBA{R: 0x74, G: 0xd3, B: 0xa5, A: 0x80})
		}, "ground.png (3, 4): #74d3a5 is half transparent"},
		"two regions of one color": {func(t *testing.T, f fstest.MapFS) {
			editManifest(t, f, func(m map[string]any) {
				regions := m["regions"].([]any)
				regions[1].(map[string]any)["color"] = regions[0].(map[string]any)["color"]
			})
		}, "regions 1 and 2 have the same color"},
		"a rock on water": {func(t *testing.T, f fstest.MapFS) {
			paint(t, f, "ground.png", 3, 4, domain.WaterBlue)
			paint(t, f, "objects.png", 3, 4, domain.RockGray)
		}, "objects.png (3, 4): a rock on water"},
		"a map too big": {func(t *testing.T, f fstest.MapFS) {
			editManifest(t, f, func(m map[string]any) { m["width"] = 1 << 30 })
		},
			"map.json: invalid map: size"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			folder := encoded(t, smallTerrain())
			c.breakIt(t, folder)
			_, err := infra.DecodeMapFile(folder)
			require.Error(t, err)
			assert.Contains(t, err.Error(), c.says)
		})
	}
}

// ADR 0007 test 16
func TestMapFileAcceptsAHandMadeCliff(t *testing.T) {
	m := domain.Generate(domain.GenerateOptions{Slug: "flat", Name: "Flat", Width: 64, Height: 64}, rand.New(rand.NewPCG(1, 1)))
	folder := encoded(t, m)
	paint(t, folder, "height.png", 5, 5, gray(0xc0))
	read, err := infra.DecodeMapFile(folder)
	require.NoError(t, err)
	assert.Equal(t, int8(2), read.HeightAt(core.Cell{Col: 5, Row: 5}), "2 next to 0")
}

// ADR 0007 test 13
func TestTheGeneratedWorldPassesTheMapFileChecks(t *testing.T) {
	world := domain.Generate(domain.StarterWorld(), rand.New(rand.NewPCG(42, 42)))
	read, err := infra.DecodeMapFile(encoded(t, world))
	require.NoError(t, err)
	assert.Equal(t, world.Chunks, read.Chunks)
	assert.Len(t, read.Objects, len(world.Objects))
}

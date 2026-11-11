package domain_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"

	core "donut/internal/core/domain"
	"donut/internal/worldmap/domain"
)

// stripes is a 64 x 64 map of grass, region 1 on columns 0 to 31 and region 2 on columns 32 to 63.
func stripes(wraps bool) domain.Map {
	m := domain.Map{
		Description: domain.Description{
			Summary: domain.Summary{Slug: "stripes", Width: 64, Height: 64, Wraps: wraps},
			Regions: []domain.Region{{ID: 1, Name: "West"}, {ID: 2, Name: "East"}},
		},
		Chunks: map[core.ChunkKey]*domain.Chunk{},
	}
	for cx := range 2 {
		for cy := range 2 {
			key := core.ChunkKey{CX: cx, CY: cy}
			chunk := domain.NewChunk(key, domain.TileGrass)
			for i := range chunk.Region {
				chunk.Region[i] = uint8(cx + 1)
			}
			m.Chunks[key] = chunk
		}
	}
	return m
}

// at is a pixel's color.
func at(img *image.Paletted, x, y int) color.NRGBA {
	return img.At(x, y).(color.NRGBA)
}

// ADR 0005 test 1
func TestImageIsTheMapsSize(t *testing.T) {
	img := domain.Image(stripes(true))
	assert.Equal(t, 64, img.Bounds().Dx())
	assert.Equal(t, 64, img.Bounds().Dy())
}

// ADR 0005 test 2
func TestImageShadesEachRegion(t *testing.T) {
	img := domain.Image(stripes(true))
	assert.Equal(t, domain.RegionShades[0], at(img, 10, 10))
	assert.Equal(t, domain.RegionShades[1], at(img, 50, 10))

	t.Run("12 shades, all different, none the grass green", func(t *testing.T) {
		seen := map[color.NRGBA]bool{domain.GrassGreen: true}
		for _, shade := range domain.RegionShades {
			assert.False(t, seen[shade], "shade %v twice", shade)
			seen[shade] = true
		}
		assert.Len(t, domain.RegionShades, 12)
	})

	t.Run("region 0 is the grass green, and region 13 takes the first shade again", func(t *testing.T) {
		assert.Equal(t, domain.GrassGreen, domain.RegionColor(0, false))
		assert.Equal(t, domain.RegionShades[0], domain.RegionColor(13, false))
	})
}

// ADR 0005 test 3
func TestImageDarkensBorders(t *testing.T) {
	west, east := domain.RegionColor(1, true), domain.RegionColor(2, true)
	assert.NotEqual(t, domain.RegionShades[0], west, "a border is darker")

	t.Run("between regions", func(t *testing.T) {
		img := domain.Image(stripes(false))
		assert.Equal(t, west, at(img, 31, 20))
		assert.Equal(t, east, at(img, 32, 20))
		assert.Equal(t, domain.RegionShades[0], at(img, 30, 20))
	})

	t.Run("across the seam of a wrapping map", func(t *testing.T) {
		img := domain.Image(stripes(true))
		assert.Equal(t, west, at(img, 0, 20))
		assert.Equal(t, east, at(img, 63, 20))
	})

	t.Run("not across the edge of a bounded map", func(t *testing.T) {
		img := domain.Image(stripes(false))
		assert.Equal(t, domain.RegionShades[0], at(img, 0, 20))
		assert.Equal(t, domain.RegionShades[1], at(img, 63, 20))
	})
}

// ADR 0005 test 4
func TestImageRocksAndVoid(t *testing.T) {
	m := stripes(false)
	m.Objects = []domain.Object{{Kind: domain.ObjectRock, Col: 5, Row: 5}, {Kind: domain.ObjectRock, Col: 31, Row: 5}}
	delete(m.Chunks, core.ChunkKey{CX: 0, CY: 1})
	img := domain.Image(m)

	assert.Equal(t, domain.RockGray, at(img, 5, 5))
	assert.Equal(t, domain.RockGray, at(img, 31, 5), "a rock on a border")
	assert.Equal(t, uint8(0), at(img, 10, 40).A, "a chunk with no row is void, transparent")
	assert.Equal(t, domain.RegionShades[0], at(img, 10, 31), "void next to a region is no border")
}

// ADR 0007 tests 23 to 25
func TestImageTerrain(t *testing.T) {
	m := stripes(false)
	m.SetGround(cell(5, 5), domain.TileWater)
	m.SetGround(cell(6, 5), domain.TileHole)
	m.SetGround(cell(7, 5), domain.TileBridge)
	m.SetGround(cell(8, 5), domain.TileOcean)
	m.SetGround(cell(9, 5), domain.TileMountain)
	m.SetGround(cell(10, 5), domain.TileGlacier)
	for col, height := range map[int]int8{10: 2, 11: 0, 12: -2, 13: 3, 14: -3} {
		m.SetHeight(cell(col, 10), height)
	}
	img := domain.Image(m)

	t.Run("water, the ocean, holes, bridges, mountains and glaciers have their colors", func(t *testing.T) {
		assert.Equal(t, domain.WaterBlue, at(img, 5, 5))
		assert.Equal(t, domain.HoleDark, at(img, 6, 5))
		assert.Equal(t, domain.BridgeBrown, at(img, 7, 5))
		assert.Equal(t, domain.OceanBlue, at(img, 8, 5), "the ocean (ADR 0008)")
		assert.Equal(t, domain.MountainGray, at(img, 9, 5), "a mountain (ADR 0009)")
		assert.Equal(t, domain.GlacierIce, at(img, 10, 5), "a glacier (ADR 0009)")
	})

	t.Run("grass is lighter going up and darker going down, in the same region", func(t *testing.T) {
		brightness := func(x int) int { c := at(img, x, 10); return int(c.R) + int(c.G) + int(c.B) }
		assert.Equal(t, domain.RegionShades[0], at(img, 11, 10), "height 0 is the region's shade")
		assert.Greater(t, brightness(13), brightness(10))
		assert.Greater(t, brightness(10), brightness(11))
		assert.Greater(t, brightness(11), brightness(12))
		assert.Greater(t, brightness(12), brightness(14))
		assert.Equal(t, domain.GrassColor(1, false, 2), at(img, 10, 10))
	})

	t.Run("it stays a paletted PNG, of at most 256 colors", func(t *testing.T) {
		assert.LessOrEqual(t, len(img.Palette), 256)
		assert.Len(t, img.Palette, 190)
	})
}

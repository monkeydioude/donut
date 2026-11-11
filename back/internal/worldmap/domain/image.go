package domain

import (
	"image"
	"image/color"
	"math"

	core "donut/internal/core/domain"
)

// The map image's colors (ADR 0005, ADR 0007), from the client's art: its grass green, a shade of it per region, its
// rocks' light gray, and the terrain's colors. The map file paints its ground with these too.
var (
	GrassGreen  = color.NRGBA{R: 0x74, G: 0xd3, B: 0xa5, A: 0xff}
	RockGray    = color.NRGBA{R: 0xd2, G: 0xce, B: 0xc5, A: 0xff}
	WaterBlue   = color.NRGBA{R: 0x6f, G: 0xb7, B: 0xe0, A: 0xff}
	OceanBlue   = color.NRGBA{R: 0x4f, G: 0x9f, B: 0xd8, A: 0xff}
	HoleDark    = color.NRGBA{R: 0x2b, G: 0x25, B: 0x40, A: 0xff}
	BridgeBrown = color.NRGBA{R: 0xb9, G: 0x8a, B: 0x5a, A: 0xff}
	// RegionShades are close to the grass green, but easy to tell apart. Region n has shade (n - 1) mod 12.
	RegionShades = [...]color.NRGBA{
		{R: 0x7f, G: 0xd9, B: 0x9a, A: 0xff},
		{R: 0x6c, G: 0xc9, B: 0xa0, A: 0xff},
		{R: 0x8a, G: 0xdf, B: 0xae, A: 0xff},
		{R: 0x68, G: 0xcf, B: 0xb0, A: 0xff},
		{R: 0x93, G: 0xd9, B: 0x9c, A: 0xff},
		{R: 0x74, G: 0xc7, B: 0x93, A: 0xff},
		{R: 0x82, G: 0xe0, B: 0xbd, A: 0xff},
		{R: 0x60, G: 0xc2, B: 0xa1, A: 0xff},
		{R: 0x9c, G: 0xe3, B: 0xb0, A: 0xff},
		{R: 0x6f, G: 0xd6, B: 0xbf, A: 0xff},
		{R: 0x86, G: 0xcf, B: 0x8f, A: 0xff},
		{R: 0x79, G: 0xbd, B: 0x9f, A: 0xff},
	}
)

const (
	// borderDarkening is how much of its shade a border cell keeps.
	borderDarkening = 0.78
	// levelShading is how much lighter grass gets per level up (mixed with white), and darker per level down.
	levelShading = 0.07
)

// imagePalette holds every color of a map's image, so it's a small paletted PNG: transparent, then each shade group's
// grass, plain then on a border, at each level, then water, the ocean, holes, bridges and rocks. That's 188 colors.
var imagePalette = func() color.Palette {
	palette := color.Palette{color.NRGBA{}}
	for _, shade := range append([]color.NRGBA{GrassGreen}, RegionShades[:]...) {
		for _, plain := range []color.NRGBA{shade, darker(shade)} {
			for level := MinHeight; level <= MaxHeight; level++ {
				palette = append(palette, atLevel(plain, int8(level)))
			}
		}
	}
	return append(palette, WaterBlue, OceanBlue, HoleDark, BridgeBrown, RockGray)
}()

const transparentIndex = 0

var (
	waterIndex  = uint8(len(imagePalette) - 5)
	oceanIndex  = uint8(len(imagePalette) - 4)
	holeIndex   = uint8(len(imagePalette) - 3)
	bridgeIndex = uint8(len(imagePalette) - 2)
	rockIndex   = uint8(len(imagePalette) - 1)
)

// Image is the whole map, 1 pixel per cell: grass in its region's shade, darker on region borders, lighter or darker
// with its height; water, the ocean, holes and bridges in their colors; rocks over them, and `void` transparent. Cells of a chunk
// with no row are `void`.
func Image(m Map) *image.Paletted {
	grid := m.Grid()
	img := image.NewPaletted(image.Rect(0, 0, m.Width, m.Height), imagePalette)
	for row := range m.Height {
		for col := range m.Width {
			img.SetColorIndex(col, row, cellIndex(m, grid, core.Cell{Col: col, Row: row}))
		}
	}
	for _, o := range m.Objects {
		if o.Kind == ObjectRock {
			img.SetColorIndex(o.Col, o.Row, rockIndex)
		}
	}
	return img
}

// cellIndex is a cell's color in the palette.
func cellIndex(m Map, grid core.Grid, cell core.Cell) uint8 {
	ground, region := cellAt(m, cell)
	switch ground {
	case TileGrass:
		return grassIndex(region, isBorder(m, grid, cell, region), m.HeightAt(cell))
	case TileWater:
		return waterIndex
	case TileOcean:
		return oceanIndex
	case TileHole:
		return holeIndex
	case TileBridge:
		return bridgeIndex
	}
	return transparentIndex
}

// RegionColor is a region's shade at height 0, darker on a border. Region 0 is the grass green.
func RegionColor(region uint8, border bool) color.NRGBA {
	return GrassColor(region, border, 0)
}

// GrassColor is a region's shade, darker on a border, lighter going up and darker going down.
func GrassColor(region uint8, border bool, height int8) color.NRGBA {
	return imagePalette[grassIndex(region, border, height)].(color.NRGBA)
}

// grassIndex is a region's grass in the palette: its shade group, plain or border, at its level.
func grassIndex(region uint8, border bool, height int8) uint8 {
	group := 0
	if region > 0 {
		group = 1 + (int(region)-1)%len(RegionShades)
	}
	variant := 2 * group
	if border {
		variant++
	}
	return uint8(1 + variant*levels + int(height) - MinHeight)
}

// isBorder tells if one of the cell's 4 neighbors, across the seams on a wrapping map, is in another region.
// A `void` neighbor or the edge of a bounded map is no border.
func isBorder(m Map, grid core.Grid, cell core.Cell, region uint8) bool {
	for _, d := range [...]core.Cell{{Col: 1}, {Col: -1}, {Row: 1}, {Row: -1}} {
		next, ok := grid.Normalize(core.Cell{Col: cell.Col + d.Col, Row: cell.Row + d.Row})
		if !ok {
			continue
		}
		if ground, other := cellAt(m, next); ground != TileVoid && other != region {
			return true
		}
	}
	return false
}

// cellAt is a cell's ground and region; `void` when its chunk has no row.
func cellAt(m Map, cell core.Cell) (TileKind, uint8) {
	chunk, ok := m.Chunks[core.ChunkOf(cell)]
	if !ok {
		return TileVoid, 0
	}
	i := chunk.local(cell)
	return chunk.Ground[i], chunk.Region[i]
}

func darker(c color.NRGBA) color.NRGBA {
	return scaled(c, func(v float64) float64 { return v * borderDarkening })
}

// atLevel is a color lighter by levelShading per level up, mixed with white, and darker by as much per level down.
func atLevel(c color.NRGBA, level int8) color.NRGBA {
	amount := levelShading * float64(level)
	if level >= 0 {
		return scaled(c, func(v float64) float64 { return v + (255-v)*amount })
	}
	return scaled(c, func(v float64) float64 { return v * (1 + amount) })
}

func scaled(c color.NRGBA, scale func(float64) float64) color.NRGBA {
	channel := func(v uint8) uint8 { return uint8(math.Round(scale(float64(v)))) }
	return color.NRGBA{R: channel(c.R), G: channel(c.G), B: channel(c.B), A: c.A}
}

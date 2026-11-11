package domain

import (
	"image"
	"image/color"
	"math"

	core "donut/internal/core/domain"
)

// The map image's colors (ADR 0005, ADR 0007, ADR 0009, ADR 0011, ADR 0012), from the client's art: its grass green, a
// shade of it per region, its rocks' light gray, the terrain's colors, and gold for domes, so a city is easy to find.
// The map file paints its ground with these too.
var (
	GrassGreen = color.NRGBA{R: 0x74, G: 0xd3, B: 0xa5, A: 0xff}
	RockGray   = color.NRGBA{R: 0xd2, G: 0xce, B: 0xc5, A: 0xff}
	DomeGold   = color.NRGBA{R: 0xf2, G: 0xc1, B: 0x4e, A: 0xff}
	// FloorPaving and WallMetal are a city's paving and its dome's rim (ADR 0012).
	FloorPaving  = color.NRGBA{R: 0xd9, G: 0xd2, B: 0xc3, A: 0xff}
	WallMetal    = color.NRGBA{R: 0x6f, G: 0x6a, B: 0x7a, A: 0xff}
	WaterBlue    = color.NRGBA{R: 0x6f, G: 0xb7, B: 0xe0, A: 0xff}
	OceanBlue    = color.NRGBA{R: 0x4f, G: 0x9f, B: 0xd8, A: 0xff}
	HoleDark     = color.NRGBA{R: 0x2b, G: 0x25, B: 0x40, A: 0xff}
	BridgeBrown  = color.NRGBA{R: 0xb9, G: 0x8a, B: 0x5a, A: 0xff}
	MountainGray = color.NRGBA{R: 0x8a, G: 0x86, B: 0x94, A: 0xff}
	GlacierIce   = color.NRGBA{R: 0xdf, G: 0xf4, B: 0xfb, A: 0xff}
	// GlacierShade is the glacier's flat ice in the client, which the map image shades by level: GlacierIce, the
	// ground layer's color, is too pale to show the ring's steps.
	GlacierShade = color.NRGBA{R: 0x9f, G: 0xe4, B: 0xf1, A: 0xff}
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

// allLevels is how many heights a mountain or a glacier may have, from MinHeight to MaxHeight.
const allLevels = MaxHeight - MinHeight + 1

// imagePalette holds every color of a map's image, so it's a small paletted PNG: transparent, then each shade group's
// grass, plain then on a border, at each level of the terrain, then water, the ocean, holes, bridges, rocks and domes,
// floors and walls, then mountains and glaciers at each level. That's 211 colors.
var imagePalette = func() color.Palette {
	palette := color.Palette{color.NRGBA{}}
	for _, shade := range append([]color.NRGBA{GrassGreen}, RegionShades[:]...) {
		for _, plain := range []color.NRGBA{shade, darker(shade)} {
			palette = append(palette, shadesOf(plain, MaxGroundHeight)...)
		}
	}
	palette = append(palette, WaterBlue, OceanBlue, HoleDark, BridgeBrown, RockGray, DomeGold, FloorPaving, WallMetal)
	palette = append(palette, shadesOf(MountainGray, MaxHeight)...)
	return append(palette, shadesOf(GlacierShade, MaxHeight)...)
}()

// shadesOf is a color at each level from MinHeight to `top`.
func shadesOf(c color.NRGBA, top int8) []color.Color {
	shades := make([]color.Color, 0, int(top)-MinHeight+1)
	for level := int8(MinHeight); level <= top; level++ {
		shades = append(shades, atLevel(c, level))
	}
	return shades
}

const transparentIndex = 0

var (
	waterIndex    = uint8(len(imagePalette) - 2*allLevels - 8)
	oceanIndex    = waterIndex + 1
	holeIndex     = waterIndex + 2
	bridgeIndex   = waterIndex + 3
	rockIndex     = waterIndex + 4
	domeIndex     = waterIndex + 5
	floorIndex    = waterIndex + 6
	wallIndex     = waterIndex + 7
	mountainIndex = waterIndex + 8
	glacierIndex  = mountainIndex + allLevels
)

// Image is the whole map, 1 pixel per cell: grass in its region's shade, darker on region borders, lighter or darker
// with its height, and mountains and glaciers too; water, the ocean, holes and bridges in their colors; rocks and
// domes over them, and `void` transparent. Cells of a chunk with no row are `void`.
func Image(m Map) *image.Paletted {
	grid := m.Grid()
	img := image.NewPaletted(image.Rect(0, 0, m.Width, m.Height), imagePalette)
	for row := range m.Height {
		for col := range m.Width {
			img.SetColorIndex(col, row, cellIndex(m, grid, core.Cell{Col: col, Row: row}))
		}
	}
	for _, o := range m.Objects {
		for _, cell := range o.Footprint() {
			img.SetColorIndex(cell.Col, cell.Row, objectIndex(o.Kind))
		}
	}
	return img
}

func objectIndex(kind ObjectKind) uint8 {
	if kind == ObjectDome {
		return domeIndex
	}
	return rockIndex
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
	case TileFloor:
		return floorIndex
	case TileWall:
		return wallIndex
	case TileMountain:
		return mountainIndex + levelIndex(m.HeightAt(cell))
	case TileGlacier:
		return glacierIndex + levelIndex(m.HeightAt(cell))
	}
	return transparentIndex
}

// levelIndex is a height's place among a kind's shades, from MinHeight.
func levelIndex(height int8) uint8 {
	return uint8(int(height) - MinHeight)
}

// MountainColor and GlacierColor are their shades at a height, lighter going up.
func MountainColor(height int8) color.NRGBA {
	return imagePalette[mountainIndex+levelIndex(height)].(color.NRGBA)
}

func GlacierColor(height int8) color.NRGBA {
	return imagePalette[glacierIndex+levelIndex(height)].(color.NRGBA)
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

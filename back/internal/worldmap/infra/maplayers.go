package infra

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"

	core "donut/internal/core/domain"
	"donut/internal/worldmap/domain"
)

// namedColor is one exact color of a layer, with what it means there.
type namedColor struct {
	name string
	rgb  color.NRGBA
}

// layer is one PNG of a map folder: its colors, each meaning one value. Transparent is value 0 when the layer allows
// it; the colors are values 1, 2… in order.
type layer struct {
	file string
	// what one value is, as in "#010101 isn't a height"
	what        string
	transparent string // what transparent means, "" when it isn't allowed
	colors      []namedColor
}

// groundLayer's values are the ground codes: transparent is void (0), then grass, water, hole, bridge, ocean,
// mountain and glacier.
func groundLayer() layer {
	return layer{file: groundFile, what: "a ground kind", transparent: "void", colors: []namedColor{
		{"grass", domain.GrassGreen}, {"water", domain.WaterBlue}, {"hole", domain.HoleDark}, {"bridge", domain.BridgeBrown}, {"ocean", domain.OceanBlue},
		{"mountain", domain.MountainGray}, {"glacier", domain.GlacierIce},
	}}
}

// heightColors are the height layer's colors from MinHeight to MaxHeight (ADR 0011): blue to red from −3 to +3, like
// a relief map, then purples above, where only mountains and glaciers go.
var heightColors = [...]color.NRGBA{
	{R: 0x1e, G: 0x3a, B: 0x8a, A: 0xff}, {R: 0x25, G: 0x63, B: 0xeb, A: 0xff}, {R: 0x38, G: 0xbd, B: 0xf8, A: 0xff},
	{R: 0x4a, G: 0xde, B: 0x80, A: 0xff},
	{R: 0xfa, G: 0xcc, B: 0x15, A: 0xff}, {R: 0xf9, G: 0x73, B: 0x16, A: 0xff}, {R: 0xdc, G: 0x26, B: 0x26, A: 0xff},
	{R: 0xd8, G: 0xb4, B: 0xfe, A: 0xff}, {R: 0xa8, G: 0x55, B: 0xf7, A: 0xff}, {R: 0x6b, G: 0x21, B: 0xa8, A: 0xff},
}

// heightLayer is a color per level, from MinHeight to MaxHeight.
func heightLayer() layer {
	l := layer{file: heightFile, what: "a height"}
	for i, rgb := range heightColors {
		l.colors = append(l.colors, namedColor{fmt.Sprintf("height %d", domain.MinHeight+i), rgb})
	}
	return l
}

// heightOf is a height layer's value as a level.
func heightOf(value uint8) int8 {
	return int8(int(value) - 1 + domain.MinHeight)
}

func regionsLayer(regions []manifestRegion) layer {
	l := layer{file: regionsFile, what: "a region's color", transparent: "no region"}
	for _, region := range regions {
		l.colors = append(l.colors, namedColor{region.Name, region.rgb})
	}
	return l
}

func objectsLayer() layer {
	return layer{file: objectsFile, what: "an object", transparent: "nothing", colors: []namedColor{{"rock", domain.RockGray}}}
}

// encode paints each cell with the color of its value, as a paletted PNG.
func (l layer) encode(width, height int, valueAt func(core.Cell) uint8) ([]byte, error) {
	palette := color.Palette{color.NRGBA{}}
	for _, c := range l.colors {
		palette = append(palette, c.rgb)
	}
	img := image.NewPaletted(image.Rect(0, 0, width, height), palette)
	for row := range height {
		for col := range width {
			img.SetColorIndex(col, row, valueAt(core.Cell{Col: col, Row: row}))
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, fmt.Errorf("%s: %w", l.file, err)
	}
	return out.Bytes(), nil
}

// decode reads each cell's value, row by row. It refuses a PNG of another size, read before decoding it, a
// half-transparent pixel, and a color that isn't one of the layer's, naming the pixel.
func (l layer) decode(folder fs.FS, width, height int) ([]uint8, error) {
	img, err := l.read(folder, width, height)
	if err != nil {
		return nil, err
	}
	values := map[color.NRGBA]uint8{}
	for i, c := range l.colors {
		values[c.rgb] = uint8(i + 1)
	}
	cells := make([]uint8, width*height)
	for row := range height {
		for col := range width {
			value, err := l.valueOf(color.NRGBAModel.Convert(img.At(col, row)).(color.NRGBA), values)
			if err != nil {
				return nil, fmt.Errorf("map file: %s (%d, %d): %w", l.file, col, row, err)
			}
			cells[row*width+col] = value
		}
	}
	return cells, nil
}

func (l layer) read(folder fs.FS, width, height int) (image.Image, error) {
	data, err := fs.ReadFile(folder, l.file)
	if err != nil {
		return nil, fmt.Errorf("map file: %w", err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("map file: %s: %w", l.file, err)
	}
	if config.Width != width || config.Height != height {
		return nil, fmt.Errorf("map file: %s is %d x %d, the map is %d x %d", l.file, config.Width, config.Height, width, height)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("map file: %s: %w", l.file, err)
	}
	return img, nil
}

func (l layer) valueOf(c color.NRGBA, values map[color.NRGBA]uint8) (uint8, error) {
	switch {
	case c.A == 0 && l.transparent != "":
		return 0, nil
	case c.A == 0:
		return 0, fmt.Errorf("transparent isn't %s", l.what)
	case c.A != 0xff:
		return 0, fmt.Errorf("%s is half transparent", hexColor(c))
	}
	value, ok := values[c]
	if !ok {
		return 0, fmt.Errorf("%s isn't %s", hexColor(c), l.what)
	}
	return value, nil
}

func hexColor(c color.NRGBA) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

func parseHexColor(text string) (color.NRGBA, error) {
	var c color.NRGBA
	if _, err := fmt.Sscanf(text, "#%02x%02x%02x", &c.R, &c.G, &c.B); err != nil || len(text) != 7 {
		return color.NRGBA{}, fmt.Errorf("%q isn't a color like #7fd99a", text)
	}
	c.A = 0xff
	return c, nil
}

package spritesheet

import (
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
)

// Style is how a sprite is brought to Donut's look.
type Style struct {
	// Outline replaces the near-black pixels.
	Outline color.NRGBA
	// DarkMax: a pixel whose every channel is at most this is near-black.
	DarkMax uint8
}

// DonutStyle recolors near-black to the players' outline color, 0x3f3550 (ADR 0006).
func DonutStyle() Style {
	return Style{Outline: color.NRGBA{R: 0x3f, G: 0x35, B: 0x50, A: 0xff}, DarkMax: 40}
}

// Recolor gives a copy of the sprite with its alpha snapped to 0 or 255, and its near-black pixels in the outline
// color. Its top-left corner is (0, 0).
func Recolor(src image.Image, style Style) *image.NRGBA {
	bounds := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := range bounds.Dy() {
		for x := range bounds.Dx() {
			c := color.NRGBAModel.Convert(src.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.NRGBA)
			out.SetNRGBA(x, y, recolorPixel(c, style))
		}
	}
	return out
}

// PixelHash is the SHA-256, in hex, of a sprite's pixels with alpha snapped: row by row, R, G, B and A each. It's
// what a pick pins, before any recolor.
func PixelHash(src image.Image) string {
	bounds := src.Bounds()
	sum := sha256.New()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := snapAlpha(color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA))
			sum.Write([]byte{c.R, c.G, c.B, c.A})
		}
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func recolorPixel(c color.NRGBA, style Style) color.NRGBA {
	c = snapAlpha(c)
	if c.A != 0 && c.R <= style.DarkMax && c.G <= style.DarkMax && c.B <= style.DarkMax {
		return style.Outline
	}
	return c
}

// snapAlpha makes a pixel transparent (all zero) or opaque.
func snapAlpha(c color.NRGBA) color.NRGBA {
	if c.A < 128 {
		return color.NRGBA{}
	}
	return color.NRGBA{R: c.R, G: c.G, B: c.B, A: 0xff}
}

package state

import "github.com/monkeydioude/donut/pkg/coords"

type Screen struct {
	Width  int
	Height int
}

func (s Screen) Normalize(x, y, w, h int) (int, int) {
	return s.NormalizeX(x, w), s.NormalizeY(y, h)
}

func (s Screen) NormalizeWithCamera(x, y, w, h int, c *coords.Camera) (int, int) {
	return s.NormalizeXWithCamera(x, w, c), s.NormalizeYWithCamera(y, h, c)
}

func (s Screen) NormalizeX(x, w int) int {
	return x * s.Width / w
}

func (s Screen) NormalizeXWithCamera(x, w int, c *coords.Camera) int {
	return c.NormalizeSize(x * s.Width / w)
}

func (s Screen) NormalizeY(y, h int) int {
	return y * s.Height / h
}

func (s Screen) NormalizeYWithCamera(x, w int, c *coords.Camera) int {
	return c.NormalizeSize(x * s.Width / w)
}

func NewScreen(w, h int) Screen {
	return Screen{
		Width:  w,
		Height: h,
	}
}

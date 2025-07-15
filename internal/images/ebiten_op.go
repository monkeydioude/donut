package images

import "github.com/hajimehoshi/ebiten/v2"

func TranslatedImage(x, y int) *ebiten.DrawImageOptions {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	return op
}

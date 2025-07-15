package fsm

import "github.com/hajimehoshi/ebiten/v2"

type State interface {
	Name() string
	Update() error
	Draw(screen *ebiten.Image)
	AllowedNext() []string
	CanGoToPrevious() bool
}

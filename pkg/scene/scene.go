package scene

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Scene interface {
	Name() string
	Load(*GameState) error
	Unload() error
	Update(*GameState) error
	Draw(screen *ebiten.Image, gs *GameState)
}

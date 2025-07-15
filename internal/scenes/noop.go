package scenes

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/monkeydioude/donut/pkg/scene"
)

type Noop struct{}

func (Noop) Name() string {
	return "noop"
}

func (Noop) Update(gs *scene.GameState) error {
	if gs == nil {
		return fmt.Errorf("nil gs pointer")
	}
	_ = gs.Manager.ChangeScene(&Intro{}, gs)
	return nil
}

func (n Noop) Draw(screen *ebiten.Image, gs *scene.GameState) {
	ebitenutil.DebugPrint(screen, fmt.Sprintf("scene '%s'", n.Name()))
}

func (n *Noop) Load(gs *scene.GameState) error {
	return nil
}

func (n *Noop) Unload() error {
	return nil
}

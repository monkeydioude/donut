package scenes

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/monkeydioude/donut/internal/uis"
	"github.com/monkeydioude/donut/pkg/scene"
)

type Intro struct {
	UI *ebitenui.UI
}

func (Intro) Name() string {
	return "Intro"
}

func (i *Intro) Update(gs *scene.GameState) error {
	if gs == nil {
		return fmt.Errorf("nil gs pointer")
	}
	i.UI.Update()
	return nil
}

func (i Intro) Draw(screen *ebiten.Image, gs *scene.GameState) {
	ebitenutil.DebugPrint(screen, fmt.Sprintf("scene '%s'", i.Name()))
	i.UI.Draw(screen)
}

func (i *Intro) Load(gs *scene.GameState) error {
	i.UI = uis.Intro(func() error {
		return gs.Manager.ChangeScene(&Game{}, gs)
	})
	return nil
}

func (i *Intro) Unload() error {
	return nil
}

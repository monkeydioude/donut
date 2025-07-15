package main

import (
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/monkeydioude/donut/config"
	"github.com/monkeydioude/donut/internal/scenes"
	"github.com/monkeydioude/donut/pkg/controls"
	"github.com/monkeydioude/donut/pkg/scene"
	"github.com/monkeydioude/donut/pkg/state"
)

type Game struct {
	sceneManager scene.Manager
	keyboard     *controls.Keyboard
	screen       state.Screen
}

func (g *Game) Update() error {
	g.keyboard.UpdateAll()
	gs := &scene.GameState{
		Manager:  &g.sceneManager,
		Screen:   g.screen,
		Keyboard: g.keyboard,
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		_ = g.sceneManager.ChangeScene(&scenes.Intro{}, gs)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		_ = g.sceneManager.ChangeScene(&scenes.Noop{}, gs)
	}
	return g.sceneManager.Update(gs)
}

func (g *Game) Draw(screen *ebiten.Image) {
	if screen == nil {
		slog.Error("nil screen pointer", "where", "main.Game.Draw")
		return
	}
	// g.rootUI.Draw(screen)
	g.sceneManager.GetCurrentScene().Draw(screen, &scene.GameState{
		Manager: &g.sceneManager,
		Screen:  g.screen,
	})
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	g.screen = state.NewScreen(config.Window.DefaultWidth, config.Window.DefaultHeight)
	return g.screen.Width, g.screen.Height
}

func NewGameMainObject() *Game {
	// root := widget.NewContainer(
	// 	widget.ContainerOpts.BackgroundImage(
	// 		image.NewNineSliceColor(colornames.Gainsboro),
	// 	),
	// )
	return &Game{
		// rootUI:       &ebitenui.UI{Container: root},
		sceneManager: scene.NewManager(&scenes.Noop{}),
		keyboard:     controls.NewKeyboard(config.Controls.KeyMapping),
	}
}

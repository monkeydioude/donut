package scenes

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/monkeydioude/donut/config"
	"github.com/monkeydioude/donut/internal/consts/controls"
	"github.com/monkeydioude/donut/internal/images"
	"github.com/monkeydioude/donut/pkg/coords"
	"github.com/monkeydioude/donut/pkg/scene"
	"github.com/monkeydioude/donut/pkg/state"
)

type Game struct {
	cellImage1 *ebiten.Image
	cellImage2 *ebiten.Image
	conf       *config.GameSceneConfig
	camera     *coords.Camera
	maps       []config.Map
}

func (Game) Name() string {
	return "Game"
}

func (g *Game) Update(gs *scene.GameState) error {
	if gs == nil {
		return fmt.Errorf("nil gs pointer")
	}
	if gs.Keyboard.IsActionOn(controls.ZoomIn) {
		g.camera.AddZoom(0.01)
	}
	if gs.Keyboard.IsActionOn(controls.ZoomOut) {
		g.camera.AddZoom(-0.01)
	}
	if gs.Keyboard.IsActionOn(controls.Up) {
		g.camera.MovingY(10)
	}
	if gs.Keyboard.IsActionOn(controls.Down) {
		g.camera.MovingY(-10)
	}
	if gs.Keyboard.IsActionOn(controls.Left) {
		g.camera.MovingX(10)
	}
	if gs.Keyboard.IsActionOn(controls.Right) {
		g.camera.MovingX(-10)
	}
	if gs.Keyboard.IsActionOn(controls.ZoomReset) {
		g.camera.ResetZoom()
	}
	if gs.Keyboard.DidActionsJustReleased(controls.Up, controls.Down) {
		g.camera.MovingY(0)
	}
	if gs.Keyboard.DidActionsJustReleased(controls.Left, controls.Right) {
		g.camera.MovingX(0)
	}
	g.camera.Update()
	return nil
}

func (g Game) ComputeOffsets(screen *state.Screen, x, y, gridCols int) (int, int) {
	iso := coords.NewIsometric(x, y, g.conf.TileWidth, g.conf.TileHeightWithOffset())
	nw := screen.NormalizeXWithCamera(g.conf.TileWidth, g.conf.RefWidth, g.camera)

	nScreenX, nScreenY := screen.Normalize(iso.X, iso.Y, g.conf.RefWidth, g.conf.RefHeight)
	// Center the grid
	offsetX := screen.Width/2 - (gridCols*nw)/11
	offsetY := 40
	return g.camera.CameraX(nScreenX + offsetX), g.camera.CameraY(nScreenY + offsetY)
}

func (g Game) Draw(screen *ebiten.Image, gs *scene.GameState) {
	nw, nh := gs.Screen.Normalize(g.conf.TileWidth, g.conf.TileHeight, g.conf.RefWidth, g.conf.RefHeight)
	g.cellImage1 = ebiten.NewImage(g.camera.NormalizeSize(nw), g.camera.NormalizeSize(nh))
	op := &ebiten.DrawImageOptions{}
	sw, sh := config.TileSet.Bounds().Dx(), config.TileSet.Bounds().Dy()
	op.GeoM.Scale(float64(g.camera.NormalizeSize(nw))/float64(sw), float64(g.camera.NormalizeSize(nh))/float64(sh))
	g.cellImage1.DrawImage(config.TileSet, op)
	g.cellImage2 = g.cellImage1
	for y, row := range config.Maps[0] {
		gridCols := len(row)
		for x, tileID := range row {
			if tileID == 0 {
				continue
			}
			realX, realY := g.ComputeOffsets(&gs.Screen, x, y, gridCols)
			op := images.TranslatedImage(realX, realY)
			if ((y*gridCols)+x)%2 == 0 {
				screen.DrawImage(g.cellImage1, op)
			} else {
				screen.DrawImage(g.cellImage2, op)
			}
		}
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("scene '%s'", g.Name()))
}

func (g *Game) Load(gs *scene.GameState) error {
	g.camera = coords.NewCamera(0, 0)
	g.conf = config.Scenes.Game
	// g.maps = config.Maps
	return nil
}

func (n *Game) Unload() error {
	return nil
}

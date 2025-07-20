package scenes

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
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
}

var (
	GridCols int = 11
	GridRows int = 11
)

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

func (g Game) ComputeOffsets(screen *state.Screen, x, y int) (int, int) {
	iso := coords.NewIsometric(x, y, g.conf.TileWidth, g.conf.TileHeight)
	nw := screen.NormalizeXWithCamera(g.conf.TileWidth, g.conf.RefWidth, g.camera)

	nScreenX, nScreenY := screen.Normalize(iso.X, iso.Y, g.conf.RefWidth, g.conf.RefHeight)
	// Center the grid
	offsetX := screen.Width/2 - (GridCols*nw)/11
	offsetY := 40
	return g.camera.CameraX(nScreenX + offsetX), g.camera.CameraY(nScreenY + offsetY)
}

func (g Game) Draw(screen *ebiten.Image, gs *scene.GameState) {
	nw, nh := gs.Screen.Normalize(g.conf.TileWidth, g.conf.TileHeight, g.conf.RefWidth, g.conf.RefHeight)
	// g.cellImage1 = drawDiamond(g.camera.NormalizeSize(nw), g.camera.NormalizeSize(nh), color.RGBA{0x66, 0xbb, 0xff, 0xff}, color.Black)
	// g.cellImage2 = drawDiamond(g.camera.NormalizeSize(nw), g.camera.NormalizeSize(nh), color.RGBA{0xff, 0x33, 0x00, 0xff}, color.Black)
	g.cellImage1 = ebiten.NewImage(g.camera.NormalizeSize(nw), g.camera.NormalizeSize(nh))
	op := &ebiten.DrawImageOptions{}
	sw, sh := config.TileSet.Size()
	op.GeoM.Scale(float64(g.camera.NormalizeSize(nw))/float64(sw), float64(g.camera.NormalizeSize(nh))/float64(sh))
	g.cellImage1.DrawImage(config.TileSet, op)
	g.cellImage2 = g.cellImage1
	for y := range GridRows {
		for x := range GridCols {
			realX, realY := g.ComputeOffsets(&gs.Screen, x, y)
			op := images.TranslatedImage(realX, realY)
			if ((y*GridCols)+x)%2 == 0 {
				screen.DrawImage(g.cellImage1, op)
			} else {
				screen.DrawImage(g.cellImage2, op)
			}
		}
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("scene '%s'", g.Name()))
}

func drawDiamond(w, h int, fill color.Color, border color.Color) *ebiten.Image {
	if w <= 0 {
		w = 1
	}
	if h <= 0 {
		h = 1
	}
	img := ebiten.NewImage(w, h)
	path := vector.Path{}
	path.MoveTo(float32(w)/2, 0)
	path.LineTo(float32(w), float32(h)/2)
	path.LineTo(float32(w)/2, float32(h))
	path.LineTo(0, float32(h)/2)
	path.Close()
	vector.DrawFilledPath(img, &path, fill, false, vector.FillRuleEvenOdd)
	// Border (stroke)
	strokeOpts := &vector.StrokeOptions{}
	strokeOpts.Width = 1
	vector.StrokePath(img, &path, border, false, strokeOpts)
	return img
}

func (g *Game) Load(gs *scene.GameState) error {
	g.camera = coords.NewCamera(0, 0)
	g.conf = config.Scenes.Game
	return nil
}

func (n *Game) Unload() error {
	return nil
}

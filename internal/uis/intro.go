package uis

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/monkeydioude/donut/internal/buttons"
	"golang.org/x/image/colornames"
)

func Intro(changeScene func() error) *ebitenui.UI {
	button := buttons.IntroButton("Play", func(args *widget.ButtonClickedEventArgs) {
		fmt.Printf("> Offset: %d, %d\n", args.OffsetX, args.OffsetY)
		_ = changeScene()
	})

	ui := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewNineSliceColor(colornames.Gainsboro),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	ui.AddChild(button)
	return &ebitenui.UI{Container: ui}
}

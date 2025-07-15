package buttons

import (
	"github.com/ebitenui/ebitenui/widget"
	"github.com/monkeydioude/donut/pkg/uiutils"
	"golang.org/x/image/colornames"
)

func IntroButton(label string, onClick func(args *widget.ButtonClickedEventArgs)) *widget.Button {
	return widget.NewButton(
		widget.ButtonOpts.TextLabel(label),
		widget.ButtonOpts.ClickedHandler(onClick),
		widget.ButtonOpts.TextFace(uiutils.DefaultFont()),
		widget.ButtonOpts.TextColor(&widget.ButtonTextColor{
			Idle:    colornames.Gainsboro,
			Hover:   colornames.Gainsboro,
			Pressed: uiutils.Mix(colornames.Gainsboro, colornames.Black, 0.4),
		}),
		widget.ButtonOpts.Image(&widget.ButtonImage{
			Idle:         uiutils.DefaultNineSlice(colornames.Darkslategray),
			Hover:        uiutils.DefaultNineSlice(uiutils.Mix(colornames.Darkslategray, colornames.Mediumseagreen, 0.4)),
			Disabled:     uiutils.DefaultNineSlice(uiutils.Mix(colornames.Darkslategray, colornames.Gainsboro, 0.8)),
			Pressed:      uiutils.PressedNineSlice(uiutils.Mix(colornames.Darkslategray, colornames.Black, 0.4)),
			PressedHover: uiutils.PressedNineSlice(uiutils.Mix(colornames.Darkslategray, colornames.Black, 0.4)),
		}),
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(180, 48),
		),
	)
}

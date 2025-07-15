package scene

import (
	"github.com/ebitenui/ebitenui"
	"github.com/monkeydioude/donut/pkg/controls"
	"github.com/monkeydioude/donut/pkg/state"
)

type GameState struct {
	UI       *ebitenui.UI
	Manager  *Manager
	Screen   state.Screen
	Keyboard *controls.Keyboard
}

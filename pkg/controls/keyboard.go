package controls

import (
	"log/slog"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/monkeydioude/donut/pkg/dtutils"
)

type Keyboard struct {
	PressedKeys    []ebiten.Key
	CurrentActions []string
	Actions        Actions
}

func (k *Keyboard) UpdatePressedKeys() {
	k.PressedKeys = inpututil.AppendJustPressedKeys(k.PressedKeys)
	tmp := []ebiten.Key{}
	for _, key := range k.PressedKeys {
		if !inpututil.IsKeyJustReleased(key) {
			tmp = append(tmp, key)
		}
	}
	k.PressedKeys = tmp
}

func (k *Keyboard) UpdateActions() {
	currActions := []string{}
	for _, key := range k.PressedKeys {
		action, ok := k.Actions[key]
		if !ok {
			slog.Warn(ErrIsNotABoundAction.Error(), "key", key)
			continue
		}
		if slices.Index(currActions, action) == -1 {
			currActions = append(currActions, action)
		}
	}
	k.CurrentActions = currActions
}

func (k *Keyboard) IsActionOn(action string) bool {
	return slices.Index(k.CurrentActions, action) > -1
}

func (k *Keyboard) DidActionJustReleased(action string) bool {
	return slices.ContainsFunc(
		dtutils.MapKeysByValue(k.Actions, action),
		func(key ebiten.Key) bool { return inpututil.IsKeyJustReleased(key) },
	) && !k.IsActionOn(action)
}

func (k *Keyboard) DidActionsJustReleased(actions ...string) bool {
	return slices.ContainsFunc(actions, func(a string) bool { return k.DidActionJustReleased(a) })
}

func (k *Keyboard) UpdateAll() {
	k.UpdatePressedKeys()
	k.UpdateActions()
}

func NewKeyboard(mapping map[string][]string) *Keyboard {
	return &Keyboard{
		Actions: NewActionsFromMapping(mapping),
	}
}

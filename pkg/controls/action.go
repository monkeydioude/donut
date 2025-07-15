package controls

import (
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2"
)

type Actions map[ebiten.Key]string

func (a Actions) ActionNameFromKey(key ebiten.Key) (string, error) {
	if action, ok := a[key]; ok {
		return action, nil
	}
	return "", ErrIsNotABoundAction
}

func NewActionsFromMapping(mapping map[string][]string) Actions {
	actions := make(Actions, len(mapping))
	for action, keys := range mapping {
		for _, key := range keys {
			ebitenKey, ok := KeyNameToEbitenKey[key]
			if !ok {
				slog.Warn("could not map key name to ebiten key", "where", "pkg.controls.NewActionsFromMapping")
				continue
			}
			actions[ebitenKey] = action
		}
	}
	return actions
}

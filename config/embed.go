//go:generate go run ./generator
package config

import (
	_ "embed"
	"runtime"
)

var (
	//go:embed yml/game.config.yml
	gameConfig []byte
	//go:embed yml/window.config.yml
	windowConfig []byte
	//go:embed yml/controls.config.yml
	controlsConfig []byte
	//go:embed img/tile_set.png
	tileSet []byte
	//go:embed yml/tiles.config.yml
	tilesConfig []byte
)

func init() {
	initScenes()
	initWindows()
	initControls()
	initTiles()

	// cleanup
	gameConfig = nil
	windowConfig = nil
	controlsConfig = nil
	tileSet = nil
	tilesConfig = nil
	runtime.GC()
}

package config

import (
	_ "embed"
)

//go:embed yml/game.config.yml
var gameConfig []byte

//go:embed yml/window.config.yml
var windowConfig []byte

//go:embed yml/controls.config.yml
var controlsConfig []byte

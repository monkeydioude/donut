package config

import (
	"runtime"

	"gopkg.in/yaml.v3"
)

func init() {
	if err := yaml.Unmarshal(windowConfig, Window); err != nil {
		panic(err)
	}

	if err := yaml.Unmarshal(gameConfig, Scenes.Game); err != nil {
		panic(err)
	}

	if err := yaml.Unmarshal(controlsConfig, Controls); err != nil {
		panic(err)
	}

	// cleanup
	gameConfig = nil
	windowConfig = nil
	controlsConfig = nil
	runtime.GC()
}

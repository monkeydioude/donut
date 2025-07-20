package config

import (
	"log"

	"gopkg.in/yaml.v3"
)

var Controls *ControlsConfig = &ControlsConfig{
	KeyMapping: make(map[string][]string),
}

type ControlsConfig struct {
	KeyMapping map[string][]string `yaml:"KeyMapping"`
}

func initControls() {
	if err := yaml.Unmarshal(controlsConfig, Controls); err != nil {
		log.Fatal(err)
	}
}

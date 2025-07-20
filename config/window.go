package config

import (
	"log"

	"gopkg.in/yaml.v3"
)

var Window *WindowConfig = &WindowConfig{}

type WindowConfig struct {
	DefaultWidth  int `yaml:"DefaultWidth"`
	DefaultHeight int `yaml:"DefaultHeight"`
}

func initWindows() {
	if err := yaml.Unmarshal(windowConfig, Window); err != nil {
		log.Fatal(err)
	}
}

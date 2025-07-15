package config

var Window *WindowConfig = &WindowConfig{}

type WindowConfig struct {
	DefaultWidth  int `yaml:"DefaultWidth"`
	DefaultHeight int `yaml:"DefaultHeight"`
}

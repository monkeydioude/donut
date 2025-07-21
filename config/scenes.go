package config

import (
	"log"

	"gopkg.in/yaml.v3"
)

var Scenes *ScenesConfig = &ScenesConfig{
	Game: &GameSceneConfig{},
}

type ScenesConfig struct {
	Game *GameSceneConfig
}

type GameSceneConfig struct {
	TileWidth        int `yaml:"TileWidth"`
	TileHeight       int `yaml:"TileHeight"`
	TileHeightOffset int `yaml:"TileHeightOffset"`
	RefWidth         int `yaml:"RefWidth"`
	RefHeight        int `yaml:"RefHeight"`
}

func (gsc GameSceneConfig) TileHeightWithOffset() int {
	return gsc.TileHeight - gsc.TileHeightOffset
}

func initScenes() {
	if err := yaml.Unmarshal(gameConfig, Scenes.Game); err != nil {
		log.Fatal(err)
	}
}

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
	TileWidth  int `yaml:"TileWidth"`
	TileHeight int `yaml:"TileHeight"`
	RefWidth   int `yaml:"RefWidth"`
	RefHeight  int `yaml:"RefHeight"`
}

func initScenes() {
	if err := yaml.Unmarshal(gameConfig, Scenes.Game); err != nil {
		log.Fatal(err)
	}
}

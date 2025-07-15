package config

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

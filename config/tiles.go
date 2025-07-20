package config

import (
	"bytes"
	"image"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"gopkg.in/yaml.v3"
)

var TileSet *ebiten.Image

type Tile struct {
	Name string `yml:"name"`
}

type TilesConfig struct {
	Mapping map[int]Tile
	TileSet *ebiten.Image
}

var Tiles = &TilesConfig{
	Mapping: make(map[int]Tile),
	TileSet: TileSet,
}

func initTiles() {
	img, _, err := image.Decode(bytes.NewReader(tileSet))
	if err != nil {
		log.Fatal(err)
	}
	TileSet = ebiten.NewImageFromImage(img)
	if err := yaml.Unmarshal(tilesConfig, Tiles.Mapping); err != nil {
		log.Fatal(err)
	}
}

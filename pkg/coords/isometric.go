package coords

type IsometricCoords struct {
	X          int
	Y          int
	TileWidth  int
	TileHeight int
}

func (ic IsometricCoords) IntoCartesian() Coords {
	return Coords{
		X: (ic.Y/(ic.TileHeight/2) + ic.X/(ic.TileWidth/2)) / 2,
		Y: (ic.Y/(ic.TileHeight/2) - ic.X/(ic.TileWidth/2)) / 2,
	}
}

func NewIsometric(cartX, cartY, tileWidth, tileHeight int) IsometricCoords {
	return Cartesian{cartX, cartY}.IntoIsometric(tileWidth, tileHeight)
}

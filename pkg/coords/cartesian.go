package coords

type Cartesian = Coords

func (c Cartesian) IntoIsometric(tw, th int) IsometricCoords {
	return IsometricCoords{
		X:          (c.X - c.Y) * (tw / 2),
		Y:          (c.X + c.Y) * (th / 2),
		TileWidth:  tw,
		TileHeight: th,
	}
}

func NewCartesian(x, y int) Cartesian {
	return Cartesian{
		X: x,
		Y: y,
	}
}

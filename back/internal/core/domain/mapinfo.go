package domain

// MapInfo is what a player is told of the map they arrive on (ADR 0012): its name, size and regions, and for a city
// where it stands. Its cells come chunk by chunk.
type MapInfo struct {
	ID       string
	Slug     string
	Name     string
	Width    int
	Height   int
	Wraps    bool
	Regions  []RegionInfo
	StandsOn *Place
}

// RegionInfo is a named area of a map.
type RegionInfo struct {
	ID    int
	Name  string
	Biome string
}

// Grid is the map's shape.
func (m MapInfo) Grid() Grid {
	return Grid{Width: m.Width, Height: m.Height, Wraps: m.Wraps}
}

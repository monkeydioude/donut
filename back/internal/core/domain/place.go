package domain

// Place is where a city stands (ADR 0012): its dome's back cell on another map, with that map's size, which the map
// screen's donut needs to show it.
type Place struct {
	Map    string `db:"map"`
	Col    int    `db:"col"`
	Row    int    `db:"row"`
	Width  int    `db:"width"`
	Height int    `db:"height"`
}

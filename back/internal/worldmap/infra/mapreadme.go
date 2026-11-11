package infra

import (
	"fmt"
	"strings"
)

// readme is the folder's README.md: what each file holds, its exact colors, and how to edit it.
func readme(man manifest) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "# Map `%s`: %s, %d x %d\n\n", man.Slug, man.Name, man.Width, man.Height)
	b.WriteString("A map built into the server (ADR 0007, ADR 0012). Every PNG is the map's size, 1 pixel per cell, column 0 on\n" +
		"the left and row 0 at the top. Each pixel is fully transparent or one of the exact colors below: no antialiasing,\n" +
		"no half transparency. Paint with a pencil, or in indexed mode with `palette.gpl` (GIMP, Aseprite).\n\n" +
		"- `go run ./cmd/mapfile check mapfiles/<slug>`, from `back/`, checks a folder (`make check-world` the world's).\n" +
		"- `make reset-maps`, with the server stopped, replaces every map in the database with its folder.\n" +
		"- `make world` and `make city` generate the world and Doma-1 again: they refuse to overwrite hand edits\n" +
		"  without `FORCE=1`.\n\n" +
		"`map.json` holds the name, size, wrapping and regions, each with its color in `regions.png`, and the domes:\n" +
		"each one's back cell (its 2 x 2 footprint's lowest column and row) and the map it leads to.\n")
	for _, l := range []layer{groundLayer(), heightLayer(), regionsLayer(man.Regions), objectsLayer()} {
		fmt.Fprintf(&b, "\n## `%s`\n\n| Color | Means |\n|-------|-------|\n", l.file)
		if l.transparent != "" {
			fmt.Fprintf(&b, "| transparent | %s |\n", l.transparent)
		}
		for _, c := range l.colors {
			fmt.Fprintf(&b, "| `%s` | %s |\n", hexColor(c.rgb), c.name)
		}
	}
	return []byte(b.String())
}

// gimpPalette is every color of the folder as a GIMP palette, which Aseprite reads too.
func gimpPalette(man manifest) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "GIMP Palette\nName: Donut map %s\nColumns: 8\n#\n", man.Slug)
	for _, l := range []layer{groundLayer(), heightLayer(), regionsLayer(man.Regions), objectsLayer()} {
		for _, c := range l.colors {
			fmt.Fprintf(&b, "%3d %3d %3d\t%s: %s\n", c.rgb.R, c.rgb.G, c.rgb.B, l.file, c.name)
		}
	}
	return []byte(b.String())
}

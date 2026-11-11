package infra

import (
	"fmt"
	"strings"
)

// readme is the folder's README.md: what each file holds, its exact colors, and how to edit it.
func readme(man manifest) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "# Map `%s`: %s, %d x %d\n\n", man.Slug, man.Name, man.Width, man.Height)
	b.WriteString("The starter map (ADR 0007). Every PNG is the map's size, 1 pixel per cell, column 0 on the left and row 0 at\n" +
		"the top. Each pixel is fully transparent or one of the exact colors below: no antialiasing, no half\n" +
		"transparency. Paint with a pencil, or in indexed mode with `palette.gpl` (GIMP, Aseprite).\n\n" +
		"- `make check-world` checks the folder.\n" +
		"- `make reset-world`, with the server stopped, replaces the world in the database with it.\n" +
		"- `make world` generates it again from its seed: it refuses to overwrite hand edits without `FORCE=1`.\n\n" +
		"`map.json` holds the name, size, wrapping and regions, each with its color in `regions.png`.\n")
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

package infra

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io/fs"

	core "donut/internal/core/domain"
	"donut/internal/worldmap/domain"
)

// The map folder (ADR 0007): map.json, then a PNG per layer, 1 pixel per cell, column 0 on the left and row 0 at the
// top. A README and a palette file help to edit it by hand.
const (
	mapFileFormat = 1
	manifestFile  = "map.json"
	groundFile    = "ground.png"
	heightFile    = "height.png"
	regionsFile   = "regions.png"
	objectsFile   = "objects.png"
	readmeFile    = "README.md"
	paletteFile   = "palette.gpl"
)

// MapFile is a map folder's files, by name.
type MapFile map[string][]byte

// manifest is map.json: the map without its cells, and its domes (ADR 0012).
type manifest struct {
	Format  int              `json:"format"`
	Slug    string           `json:"slug"`
	Name    string           `json:"name"`
	Width   int              `json:"width"`
	Height  int              `json:"height"`
	Wraps   bool             `json:"wraps"`
	Regions []manifestRegion `json:"regions"`
	Domes   []manifestDome   `json:"domes,omitempty"`
}

// manifestDome is a dome: its back cell, and the slug of the map it leads to.
type manifestDome struct {
	Col int    `json:"col"`
	Row int    `json:"row"`
	Map string `json:"map"`
}

// manifestRegion is a region, with the color it's painted in regions.png.
type manifestRegion struct {
	ID    int          `json:"id"`
	Name  string       `json:"name"`
	Biome domain.Biome `json:"biome"`
	Color string       `json:"color"`
	rgb   color.NRGBA
}

// EncodeMapFile is a map as a folder. Regions 1 to 12 are painted in the map screen's shades, the others in colors
// made from their id, so every region has its own.
func EncodeMapFile(m domain.Map) (MapFile, error) {
	man := manifestOf(m)
	data, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("EncodeMapFile: %w", err)
	}
	file := MapFile{manifestFile: append(data, '\n')}
	for _, encode := range []struct {
		layer   layer
		valueAt func(core.Cell) uint8
	}{
		{groundLayer(), func(c core.Cell) uint8 { return uint8(m.GroundAt(c)) }},
		{heightLayer(), func(c core.Cell) uint8 { return uint8(int(m.HeightAt(c)) - domain.MinHeight + 1) }},
		{regionsLayer(man.Regions), regionValues(m, man.Regions)},
		{objectsLayer(), rockValues(m)},
	} {
		if file[encode.layer.file], err = encode.layer.encode(m.Width, m.Height, encode.valueAt); err != nil {
			return nil, fmt.Errorf("EncodeMapFile: %w", err)
		}
	}
	file[readmeFile] = readme(man)
	file[paletteFile] = gimpPalette(man)
	return file, nil
}

func manifestOf(m domain.Map) manifest {
	man := manifest{Format: mapFileFormat, Slug: m.Slug, Name: m.Name, Width: m.Width, Height: m.Height, Wraps: m.Wraps, Regions: []manifestRegion{}}
	for _, region := range m.Regions {
		rgb := regionColor(region.ID)
		man.Regions = append(man.Regions, manifestRegion{ID: region.ID, Name: region.Name, Biome: region.Biome, Color: hexColor(rgb), rgb: rgb})
	}
	for _, object := range m.Objects {
		if object.Kind == domain.ObjectDome {
			man.Domes = append(man.Domes, manifestDome{Col: object.Col, Row: object.Row, Map: object.LeadsTo})
		}
	}
	return man
}

// regionColor is a region's default color in regions.png: the map screen's shade up to 12, then one made from its id.
func regionColor(id int) color.NRGBA {
	if id >= 1 && id <= len(domain.RegionShades) {
		return domain.RegionShades[id-1]
	}
	return color.NRGBA{R: uint8(id), G: 0x99, B: 0x77, A: 0xff}
}

func regionValues(m domain.Map, regions []manifestRegion) func(core.Cell) uint8 {
	values := map[uint8]uint8{}
	for i, region := range regions {
		values[uint8(region.ID)] = uint8(i + 1)
	}
	return func(c core.Cell) uint8 { return values[m.RegionAt(c)] }
}

func rockValues(m domain.Map) func(core.Cell) uint8 {
	rocks := map[core.Cell]bool{}
	for _, object := range m.Objects {
		rocks[object.Cell()] = object.Kind == domain.ObjectRock
	}
	return func(c core.Cell) uint8 {
		if rocks[c] {
			return 1
		}
		return 0
	}
}

// DecodeMapFile reads a map folder. It refuses the whole map at the first problem, naming the file, and the pixel
// for a layer. It checks what the game needs, not the generator's own rules: a hand-made cliff is fine.
func DecodeMapFile(folder fs.FS) (domain.Map, error) {
	man, err := readManifest(folder)
	if err != nil {
		return domain.Map{}, err
	}
	m := domain.NewMap(man.summary(), domain.TileVoid)
	m.Regions = man.regions()
	ground, err := groundLayer().decode(folder, m.Width, m.Height)
	if err != nil {
		return domain.Map{}, err
	}
	heights, err := heightLayer().decode(folder, m.Width, m.Height)
	if err != nil {
		return domain.Map{}, err
	}
	regions, err := regionsLayer(man.Regions).decode(folder, m.Width, m.Height)
	if err != nil {
		return domain.Map{}, err
	}
	for i := range ground {
		cell := core.Cell{Col: i % m.Width, Row: i / m.Width}
		m.SetGround(cell, domain.TileKind(ground[i]))
		m.SetHeight(cell, heightOf(heights[i]))
		m.SetRegion(cell, regionID(man.Regions, regions[i]))
	}
	if m.Objects, err = readRocks(folder, m); err != nil {
		return domain.Map{}, err
	}
	m.Objects = append(m.Objects, man.domes()...)
	if err := m.Validate(); err != nil {
		return domain.Map{}, fmt.Errorf("map file: %w", err)
	}
	return m, nil
}

// CheckDomes: each dome of a map leads to a map whose folder is in `maps`, the folder of map folders (ADR 0012). The
// map's own checks have made sure each one leads to a slug, so it names a folder of `maps`, never a path out of it.
func CheckDomes(m domain.Map, maps fs.FS) error {
	for _, object := range m.Objects {
		if object.Kind != domain.ObjectDome {
			continue
		}
		if _, err := fs.Stat(maps, object.LeadsTo+"/"+manifestFile); err != nil {
			return fmt.Errorf("map file: %s: the dome at (%d, %d) leads to %q, which has no map folder", manifestFile, object.Col, object.Row, object.LeadsTo)
		}
	}
	return nil
}

// readManifest reads map.json, and checks its format, its size and regions, and its colors, before any layer is read.
func readManifest(folder fs.FS) (manifest, error) {
	data, err := fs.ReadFile(folder, manifestFile)
	if err != nil {
		return manifest{}, fmt.Errorf("map file: %w", err)
	}
	var man manifest
	if err := json.Unmarshal(data, &man); err != nil {
		return manifest{}, fmt.Errorf("map file: %s: %w", manifestFile, err)
	}
	if man.Format != mapFileFormat {
		return manifest{}, fmt.Errorf("map file: %s: format %d, expected %d", manifestFile, man.Format, mapFileFormat)
	}
	if err := (domain.Map{Description: domain.Description{Summary: man.summary(), Regions: man.regions()}}).Validate(); err != nil {
		return manifest{}, fmt.Errorf("map file: %s: %w", manifestFile, err)
	}
	return man, man.parseColors()
}

// parseColors reads each region's color. Two regions can't share one.
func (man *manifest) parseColors() error {
	owners := map[color.NRGBA]string{}
	for i, region := range man.Regions {
		rgb, err := parseHexColor(region.Color)
		if err != nil {
			return fmt.Errorf("map file: %s: region %d: %w", manifestFile, region.ID, err)
		}
		if owner, taken := owners[rgb]; taken {
			return fmt.Errorf("map file: %s: regions %s and %d have the same color %s", manifestFile, owner, region.ID, region.Color)
		}
		owners[rgb] = fmt.Sprint(region.ID)
		man.Regions[i].rgb = rgb
	}
	return nil
}

func (man manifest) summary() domain.Summary {
	return domain.Summary{Slug: man.Slug, Name: man.Name, Width: man.Width, Height: man.Height, Wraps: man.Wraps}
}

func (man manifest) domes() []domain.Object {
	domes := []domain.Object{}
	for _, dome := range man.Domes {
		domes = append(domes, domain.Object{Kind: domain.ObjectDome, Col: dome.Col, Row: dome.Row, LeadsTo: dome.Map})
	}
	return domes
}

func (man manifest) regions() []domain.Region {
	regions := []domain.Region{}
	for _, region := range man.Regions {
		regions = append(regions, domain.Region{ID: region.ID, Name: region.Name, Biome: region.Biome})
	}
	return regions
}

// regionID is the region of a regions layer's value: 0 for none.
func regionID(regions []manifestRegion, value uint8) uint8 {
	if value == 0 {
		return 0
	}
	return uint8(regions[value-1].ID)
}

// readRocks reads the objects layer, row by row. A rock stands only on grass.
func readRocks(folder fs.FS, m domain.Map) ([]domain.Object, error) {
	objects, err := objectsLayer().decode(folder, m.Width, m.Height)
	if err != nil {
		return nil, err
	}
	rocks := []domain.Object{}
	for i, value := range objects {
		if value == 0 {
			continue
		}
		cell := core.Cell{Col: i % m.Width, Row: i / m.Width}
		if ground := m.GroundAt(cell); ground != domain.TileGrass {
			return nil, fmt.Errorf("map file: %s (%d, %d): a rock on %v, rocks stand on grass only", objectsFile, cell.Col, cell.Row, ground)
		}
		rocks = append(rocks, domain.Object{Kind: domain.ObjectRock, Col: cell.Col, Row: cell.Row})
	}
	return rocks, nil
}

// Command mapfile makes and checks map folders (ADR 0007), from back/:
//
//	go run ./cmd/mapfile generate -seed 42 -out mapfiles/world [-force]
//	go run ./cmd/mapfile city -out mapfiles/doma-1 [-force]
//	go run ./cmd/mapfile check mapfiles/world
//
// generate writes the starter world from its seed, city Doma-1's map (ADR 0012). Both refuse to overwrite a folder
// that has files without -force: they may hold hand edits. check reads a folder as the server would, and names its
// first problem: a dome must lead to a map folder beside it.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"time"

	core "donut/internal/core/domain"
	"donut/internal/worldmap/domain"
	"donut/internal/worldmap/infra"
)

var errUsage = errors.New("usage: mapfile generate -seed <n> -out <folder> [-force] | mapfile city -out <folder> [-force] | mapfile check <folder>")

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	if err := run(os.Args[1:], log); err != nil {
		log.Error("mapfile failed", "error", err)
		os.Exit(1)
	}
}

func run(args []string, log *slog.Logger) error {
	if len(args) == 0 {
		return errUsage
	}
	switch args[0] {
	case "generate":
		return generate(args[1:], log)
	case "city":
		return city(args[1:], log)
	case "check":
		if len(args) != 2 {
			return errUsage
		}
		return check(args[1], log)
	}
	return errUsage
}

func generate(args []string, log *slog.Logger) error {
	flags := flag.NewFlagSet("generate", flag.ContinueOnError)
	seed := flags.Uint64("seed", 42, "the seed the world is generated from")
	out := flags.String("out", "", "the map folder to write")
	force := flags.Bool("force", false, "overwrite a folder that has files")
	if err := flags.Parse(args); err != nil || *out == "" {
		return errUsage
	}
	if err := refuseToOverwrite(*out, *force); err != nil {
		return err
	}
	start := time.Now()
	world := domain.Generate(domain.StarterWorld(), rand.New(rand.NewPCG(*seed, *seed)))
	logWorld(world, *seed, time.Since(start), log)
	file, err := infra.EncodeMapFile(world)
	if err != nil {
		return err
	}
	return writeFolder(*out, file, log)
}

// city writes Doma-1's map, the first dome city (ADR 0012).
func city(args []string, log *slog.Logger) error {
	flags := flag.NewFlagSet("city", flag.ContinueOnError)
	out := flags.String("out", "", "the map folder to write")
	force := flags.Bool("force", false, "overwrite a folder that has files")
	if err := flags.Parse(args); err != nil || *out == "" {
		return errUsage
	}
	if err := refuseToOverwrite(*out, *force); err != nil {
		return err
	}
	m := domain.GenerateCity(domain.Doma1())
	spawn, err := domain.NewTerrain(m).SpawnPoint()
	log.Info("city generated", "slug", m.Slug, "size", fmt.Sprintf("%dx%d", m.Width, m.Height), "spawn", spawn, "spawn_error", err)
	file, err := infra.EncodeMapFile(m)
	if err != nil {
		return err
	}
	return writeFolder(*out, file, log)
}

// refuseToOverwrite: a folder that has files may hold hand edits, overwritten only with -force.
func refuseToOverwrite(folder string, force bool) error {
	entries, err := os.ReadDir(folder)
	if errors.Is(err, os.ErrNotExist) || force {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("%s has files, which may hold hand edits: add -force to overwrite them", folder)
	}
	return nil
}

func writeFolder(folder string, file infra.MapFile, log *slog.Logger) error {
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return err
	}
	for name, data := range file {
		if err := os.WriteFile(filepath.Join(folder, name), data, 0o644); err != nil {
			return err
		}
		log.Debug("map file written", "file", filepath.Join(folder, name), "bytes", len(data))
	}
	log.Info("map folder written", "folder", folder, "files", len(file))
	return nil
}

// logWorld logs what was generated: cells per level and per ground kind, rocks, and the spawn with the river's
// rows at its column.
func logWorld(m domain.Map, seed uint64, took time.Duration, log *slog.Logger) {
	levels, kinds := map[int8]int{}, map[string]int{}
	for row := range m.Height {
		for col := range m.Width {
			cell := cellAt(col, row)
			levels[m.HeightAt(cell)]++
			kinds[m.GroundAt(cell).String()]++
		}
	}
	spawn, err := domain.NewTerrain(m).SpawnPoint()
	log.Info("world generated", "slug", m.Slug, "seed", seed, "size", fmt.Sprintf("%dx%d", m.Width, m.Height), "took", took,
		"levels", levels, "ground", kinds, "rocks", len(m.Objects), "spawn", spawn, "spawn_error", err, "river_rows_at_spawn", riverRows(m, spawn.Col))
}

// riverRows are the rows of a column that hold water or a bridge.
func riverRows(m domain.Map, col int) []int {
	var rows []int
	for row := range m.Height {
		if kind := m.GroundAt(cellAt(col, row)); kind == domain.TileWater || kind == domain.TileBridge {
			rows = append(rows, row)
		}
	}
	return rows
}

func check(folder string, log *slog.Logger) error {
	start := time.Now()
	m, err := infra.DecodeMapFile(os.DirFS(folder))
	if err != nil {
		return err
	}
	if err := infra.CheckDomes(m, os.DirFS(filepath.Dir(folder))); err != nil {
		return err
	}
	log.Info("map folder checked", "folder", folder, "slug", m.Slug, "size", fmt.Sprintf("%dx%d", m.Width, m.Height),
		"regions", len(m.Regions), "rocks", len(m.Objects), "took", time.Since(start))
	return nil
}

func cellAt(col, row int) core.Cell {
	return core.Cell{Col: col, Row: row}
}

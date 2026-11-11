// Command spritesheet makes a sprite sheet from picked rows of the Pixel Art Atlas (ADR 0006).
//
//	go run ./cmd/spritesheet -picks cmd/spritesheet/robots.json -out ../client/public/sprites/robots
//
// writes <out>.png and <out>.json. It needs the network only while it runs.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"donut/internal/spritesheet"
)

func main() {
	picksPath := flag.String("picks", "", "the picks file (JSON)")
	out := flag.String("out", "", "where to write, without extension: <out>.png and <out>.json")
	flag.Parse()
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, *picksPath, *out, log); err != nil {
		log.Error("spritesheet failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, picksPath, out string, log *slog.Logger) error {
	if picksPath == "" || out == "" {
		return errors.New("usage: spritesheet -picks <file.json> -out <path without extension>")
	}
	data, err := os.ReadFile(picksPath)
	if err != nil {
		return err
	}
	picks, err := spritesheet.ParsePicks(data)
	if err != nil {
		return err
	}
	// The filter scans the whole dataset: it often takes more than 30 s.
	client := &http.Client{Timeout: 90 * time.Second}
	atlas, err := spritesheet.NewHTTPAtlas(client, spritesheet.DatasetsServer, picks, spritesheet.AtlasOptions{}, log)
	if err != nil {
		return err
	}
	sheet, manifest, err := spritesheet.Make(ctx, picks, atlas, spritesheet.DonutStyle(), log)
	if err != nil {
		return err
	}
	return write(out, sheet, manifest, log)
}

func write(out string, sheet *image.NRGBA, manifest spritesheet.Manifest, log *slog.Logger) error {
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, sheet); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	jsonBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := os.WriteFile(out+".png", pngBytes.Bytes(), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(out+".json", append(jsonBytes, '\n'), 0o644); err != nil {
		return err
	}
	log.Info("sheet written", "png", out+".png", "json", out+".json", "sprites", len(manifest.Sprites), "bytes", pngBytes.Len())
	return nil
}

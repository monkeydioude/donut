package spritesheet

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"log/slog"
)

// Cell is the side of a sprite and of its cell in the sheet, in pixels.
const Cell = 16

// Atlas gives the row of the dataset with that row_id: the row_id it found, and its sprite.
type Atlas interface {
	Row(ctx context.Context, rowID int) (foundID int, sprite image.Image, err error)
}

// Sprite is a picked sprite, ready for the sheet.
type Sprite struct {
	Slug  string
	RowID int
	Image image.Image
}

// Manifest is the sheet's JSON: each sprite's cell in the PNG, by its top-left corner.
type Manifest struct {
	Cell    int      `json:"cell"`
	Sprites []Placed `json:"sprites"`
}

type Placed struct {
	Slug  string `json:"slug"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	RowID int    `json:"rowId"`
}

// Make fetches the picks one at a time, checks each is the expected row, a Cell × Cell sprite and the pixels picked,
// recolors it, and lays the sheet. The first pick that fails stops it, named in the error.
func Make(ctx context.Context, picks Picks, atlas Atlas, style Style, log *slog.Logger) (*image.NRGBA, Manifest, error) {
	sprites := make([]Sprite, 0, len(picks.Sprites))
	for i, pick := range picks.Sprites {
		sprite, err := fetch(ctx, pick, atlas, style)
		if err != nil {
			return nil, Manifest{}, fmt.Errorf("Make: %s: %w", pick.Slug, err)
		}
		log.Info("sprite ready", "slug", pick.Slug, "row_id", pick.RowID, "done", i+1, "of", len(picks.Sprites))
		sprites = append(sprites, sprite)
	}
	sheet, manifest := Lay(sprites)
	return sheet, manifest, nil
}

func fetch(ctx context.Context, pick Pick, atlas Atlas, style Style) (Sprite, error) {
	foundID, img, err := atlas.Row(ctx, pick.RowID)
	if err != nil {
		return Sprite{}, err
	}
	if foundID != pick.RowID {
		return Sprite{}, fmt.Errorf("row_id %d gave row_id %d: the dataset changed", pick.RowID, foundID)
	}
	if size := img.Bounds().Size(); size.X != Cell || size.Y != Cell {
		return Sprite{}, fmt.Errorf("the sprite is %dx%d, not %dx%d", size.X, size.Y, Cell, Cell)
	}
	if hash := PixelHash(img); hash != pick.SHA256 {
		return Sprite{}, fmt.Errorf("row_id %d gave other pixels (sha256 %s, not %s): the API's image cache may be stale, try later", pick.RowID, hash, pick.SHA256)
	}
	return Sprite{Slug: pick.Slug, RowID: foundID, Image: Recolor(img, style)}, nil
}

// Lay puts the sprites in one row of Cell × Cell cells, in order.
func Lay(sprites []Sprite) (*image.NRGBA, Manifest) {
	sheet := image.NewNRGBA(image.Rect(0, 0, Cell*len(sprites), Cell))
	manifest := Manifest{Cell: Cell, Sprites: make([]Placed, len(sprites))}
	for i, sprite := range sprites {
		at := image.Pt(i*Cell, 0)
		draw.Draw(sheet, image.Rectangle{Min: at, Max: at.Add(image.Pt(Cell, Cell))}, sprite.Image, sprite.Image.Bounds().Min, draw.Src)
		manifest.Sprites[i] = Placed{Slug: sprite.Slug, X: at.X, Y: at.Y, RowID: sprite.RowID}
	}
	return sheet, manifest
}

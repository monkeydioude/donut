// Package spritesheet makes a sprite sheet from rows of the Pixel Art Atlas (ADR 0006): it fetches each picked
// sprite, brings it to Donut's style, and lays them all in one row, with a JSON index of where each one is.
package spritesheet

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	presence "donut/internal/presence/domain"
)

// Picks is what goes in a sheet: the dataset's rows to take, in order.
type Picks struct {
	Dataset string `json:"dataset"`
	Config  string `json:"config"`
	Split   string `json:"split"`
	Sprites []Pick `json:"sprites"`
}

// Pick is one sprite: its slug, the dataset's own id of its row, and the PixelHash of the sprite picked. The API's
// image cache can serve another row's sprite: the hash catches it.
type Pick struct {
	Slug   string `json:"slug"`
	RowID  int    `json:"rowId"`
	SHA256 string `json:"sha256"`
}

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ParsePicks reads a picks file and checks it before anything is downloaded: a dataset, config and split, at least
// one sprite, each slug one the server accepts as a robot, no slug twice, no negative row id, and each pick's
// sha256 in lowercase hex.
func ParsePicks(data []byte) (Picks, error) {
	var picks Picks
	if err := json.Unmarshal(data, &picks); err != nil {
		return Picks{}, fmt.Errorf("ParsePicks: %w", err)
	}
	if picks.Dataset == "" || picks.Config == "" || picks.Split == "" {
		return Picks{}, errors.New("ParsePicks: dataset, config and split are required")
	}
	if len(picks.Sprites) == 0 {
		return Picks{}, errors.New("ParsePicks: no sprites")
	}
	seen := map[string]bool{}
	for _, pick := range picks.Sprites {
		if err := checkPick(pick, seen); err != nil {
			return Picks{}, fmt.Errorf("ParsePicks: %w", err)
		}
		seen[pick.Slug] = true
	}
	return picks, nil
}

func checkPick(pick Pick, seen map[string]bool) error {
	if _, err := presence.CleanRobot(pick.Slug); err != nil || pick.Slug == "" {
		return fmt.Errorf("slug %q: 1 to %d lowercase letters, digits or dashes", pick.Slug, presence.MaxRobotLength)
	}
	if seen[pick.Slug] {
		return fmt.Errorf("slug %q is there twice", pick.Slug)
	}
	if pick.RowID < 0 {
		return fmt.Errorf("slug %q: negative row id %d", pick.Slug, pick.RowID)
	}
	if !sha256Hex.MatchString(pick.SHA256) {
		return fmt.Errorf("slug %q: sha256 is not 64 lowercase hex characters", pick.Slug)
	}
	return nil
}

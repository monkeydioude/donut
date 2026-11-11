// Package domain holds the rules of the players on a map: who they are, how they move, what they are told.
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	core "donut/internal/core/domain"
	"donut/pkg/errd"
)

const (
	MaxNameLength = 24
	// MaxRobotLength bounds a robot's slug (ADR 0006).
	MaxRobotLength = 32
	// StepInterval is the time of one step: everyone walks at this pace.
	StepInterval = 200 * time.Millisecond
	// TickInterval is how often a room moves its walkers.
	TickInterval = 50 * time.Millisecond
)

// robotSlug is a robot's slug: lowercase letters, digits and dashes. The server doesn't know the list of robots.
var robotSlug = regexp.MustCompile(fmt.Sprintf(`^[a-z0-9-]{1,%d}$`, MaxRobotLength))

var (
	ErrNotNeighbor = errors.New("the next cell is not one of the 8 cells around the player")
	ErrNotWalkable = errors.New("the next cell is not walkable")
)

// Profile is who someone is, as the others see them: their public id, name and robot.
type Profile struct {
	ID   string
	Name string
	// Robot is the slug of the robot they're drawn as; "" for the default one (ADR 0006).
	Robot string
}

// Player is someone on a map. Its ID is the public id of the guest id the client keeps.
type Player struct {
	ID    string
	Name  string
	Robot string
	Cell  core.Cell
}

// PublicID is the id other players see. The guest id is a secret: whoever sends it plays that player,
// so it never leaves the server. A hash can't be turned back into the guest id.
func PublicID(guestID string) string {
	sum := sha256.Sum256([]byte(guestID))
	return hex.EncodeToString(sum[:16])
}

// ParseGuestID accepts a UUID and gives it in its canonical form.
func ParseGuestID(raw string) (string, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid player id: %w", errd.ErrBadRequest)
	}
	return id.String(), nil
}

// CleanName trims a player's name, which must then be 1 to MaxNameLength printable characters.
func CleanName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	length := utf8.RuneCountInString(name)
	if length < 1 || length > MaxNameLength || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("a name has 1 to %d printable characters: %w", MaxNameLength, errd.ErrBadRequest)
	}
	return name, nil
}

// CleanRobot checks a robot's slug: empty, or 1 to MaxRobotLength lowercase letters, digits or dashes.
func CleanRobot(raw string) (string, error) {
	if raw != "" && !robotSlug.MatchString(raw) {
		return "", fmt.Errorf("a robot is 1 to %d lowercase letters, digits or dashes: %w", MaxRobotLength, errd.ErrBadRequest)
	}
	return raw, nil
}

// CheckStep tells why a walker on `from` may not step to `to` now, nil when it may. A step may cross the seams of a
// wrapping map. Paths are checked step by step as they are walked: the map may change under a walker (traps, NPCs).
func CheckStep(grid core.Grid, from, to core.Cell, isWalkable func(core.Cell) bool) error {
	switch {
	case !grid.AreNeighbors(from, to):
		return ErrNotNeighbor
	case !isWalkable(to):
		return ErrNotWalkable
	}
	return nil
}

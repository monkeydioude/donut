// Package logs builds the server's logger. It is passed to every module: no global logger.
package logs

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// New is a text logger writing to w from `level` on: debug, info, warn or error.
func New(level string, w io.Writer) (*slog.Logger, error) {
	var l slog.Level
	if err := l.UnmarshalText([]byte(strings.TrimSpace(level))); err != nil {
		return nil, fmt.Errorf("LOG_LEVEL %q: use debug, info, warn or error", level)
	}
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: l})), nil
}

// Discard is a logger that writes nothing, for tests.
func Discard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

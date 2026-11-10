// Package db opens the Postgres pool.
package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
)

// Open connects to url and checks the database answers, so a wrong DB_URL stops the boot.
// Every query is logged at debug level, with its arguments and how long it took.
func Open(ctx context.Context, url string, log *slog.Logger) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("db.Open: %w", err)
	}
	config.ConnConfig.Tracer = &tracelog.TraceLog{Logger: queryLogger(log), LogLevel: tracelog.LogLevelDebug}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("db.Open: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db.Open: ping: %w", err)
	}
	return pool, nil
}

// queryLogger writes pgx's query traces to the logger, under "db".
func queryLogger(log *slog.Logger) tracelog.Logger {
	log = log.With("component", "db")
	return tracelog.LoggerFunc(func(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
		attrs := make([]any, 0, 2*len(data))
		for key, value := range data {
			attrs = append(attrs, key, value)
		}
		log.Log(ctx, slogLevel(level), "db "+msg, attrs...)
	})
}

// slogLevel keeps queries at debug (pgx reports them as info, which would flood an info log); warnings and
// errors keep their level.
func slogLevel(level tracelog.LogLevel) slog.Level {
	switch level {
	case tracelog.LogLevelError:
		return slog.LevelError
	case tracelog.LogLevelWarn:
		return slog.LevelWarn
	default:
		return slog.LevelDebug
	}
}

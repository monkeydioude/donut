// Donut API server: the maps and their layers over HTTP, the players on a map over a websocket.
//
//	donut                    serves
//	donut reset-map <slug>   replaces a map in the database with its file (ADR 0007), the server stopped
//	donut reset-maps         replaces every map built into the server with its file (ADR 0012), the server stopped
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"syscall"
	"time"

	"github.com/calqs/gopkg/env"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/oklog/run"

	"donut/internal/worldmap"
	"donut/mapfiles"
	"donut/migrations"
	"donut/pkg/db"
	"donut/pkg/logs"
	"donut/pkg/migrate"
)

type Config struct {
	APIPort string `env:"API_PORT,?:8400"`
	DBURL   string `env:"DB_URL,?postgres://donut:donut@localhost:5434/donut?search_path=donut"`
	// FrontOrigin is where the client is served from: browsers there may open the websocket.
	FrontOrigin string `env:"FRONT_ORIGIN,?http://localhost:5180"`
	// LogLevel is debug (every action, every query), info, warn or error.
	LogLevel string `env:"LOG_LEVEL,?debug"`
}

// RedactedDBURL is DB_URL with its password hidden, for the logs.
func (c Config) RedactedDBURL() string {
	u, err := url.Parse(c.DBURL)
	if err != nil {
		return "(unreadable DB_URL)"
	}
	return u.Redacted()
}

// OriginPatterns are the hosts browsers may open the websocket from: FRONT_ORIGIN's.
func (c Config) OriginPatterns() ([]string, error) {
	u, err := url.Parse(c.FrontOrigin)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("FRONT_ORIGIN %q is not a URL like http://localhost:5180", c.FrontOrigin)
	}
	return []string{u.Host}, nil
}

func main() {
	// .env lives at the repo root; missing is fine, every value has a default.
	_ = godotenv.Load("../.env", ".env")
	config, err := env.ParseEnv[Config]()
	if err != nil {
		log.Fatal(err.Error())
	}
	logger, err := logs.New(config.LogLevel, os.Stdout)
	if err != nil {
		log.Fatal(err.Error())
	}
	slog.SetDefault(logger) // for libraries that log on their own
	if err := runCommand(context.Background(), os.Args[1:], config, logger); err != nil {
		log.Fatal(err.Error())
	}
}

// runCommand serves without arguments, or replaces maps with their files for `reset-map <slug>` and `reset-maps`.
func runCommand(ctx context.Context, args []string, config Config, log *slog.Logger) error {
	switch {
	case len(args) == 0:
		return serve(ctx, config, log)
	case len(args) == 2 && args[0] == "reset-map":
		return resetMaps(ctx, config, args[1:], log)
	case len(args) == 1 && args[0] == "reset-maps":
		return resetMaps(ctx, config, mapfiles.Slugs(), log)
	}
	return errors.New("usage: donut [reset-map <slug> | reset-maps]")
}

// serve opens the database, migrates it, saves the map files it doesn't have yet, then answers until SIGINT or SIGTERM.
func serve(ctx context.Context, config Config, log *slog.Logger) error {
	log.Info("donut API booting", "port", config.APIPort, "db", config.RedactedDBURL(), "front_origin", config.FrontOrigin,
		"log_level", config.LogLevel)
	pool, err := openDatabase(ctx, config, log)
	if err != nil {
		return err
	}
	defer pool.Close()
	handler, maps, err := routing(ctx, pool, config, log)
	if err != nil {
		return err
	}
	if _, err := maps.Service.EnsureMaps(ctx, mapfiles.Slugs(), mapfiles.Read); err != nil {
		return err
	}
	return listen(ctx, &http.Server{
		Addr:    config.APIPort,
		Handler: handler,
		// no read/write timeouts: they would cut the websockets, which stay open
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}, log)
}

// resetMaps replaces maps in the database with the files built into the server, each in its own transaction. Run it
// with the server stopped: an open room keeps the map it loaded.
func resetMaps(ctx context.Context, config Config, slugs []string, log *slog.Logger) error {
	log.Info("donut reset-maps", "slugs", slugs, "db", config.RedactedDBURL())
	pool, err := openDatabase(ctx, config, log)
	if err != nil {
		return err
	}
	defer pool.Close()
	return worldmap.NewModule(pool, log).Service.ResetMaps(ctx, slugs, mapfiles.Read)
}

// openDatabase connects and migrates. The caller closes the pool.
func openDatabase(ctx context.Context, config Config, log *slog.Logger) (*pgxpool.Pool, error) {
	pool, err := db.Open(ctx, config.DBURL, log)
	if err != nil {
		return nil, err
	}
	log.Debug("database connected")
	if err := migrateAtBoot(ctx, pool, log); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func migrateAtBoot(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	defer conn.Release()
	return migrate.Apply(ctx, conn, migrations.Files, migrations.Schema, log)
}

func listen(ctx context.Context, server *http.Server, log *slog.Logger) error {
	g := &run.Group{}
	g.Add(func() error {
		log.Info("donut API listening", "port", server.Addr)
		return server.ListenAndServe()
	}, func(err error) {
		log.Info("donut API stopping", "reason", err)
		_ = server.Close()
	})
	g.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))
	if err := g.Run(); err != nil && !errors.Is(err, run.ErrSignal) {
		return err
	}
	return nil
}

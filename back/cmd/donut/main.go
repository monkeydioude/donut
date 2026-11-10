// Donut API server: the maps and their layers over HTTP, the players on a map over a websocket.
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
	// StarterMapSeed generates the starter world, the first time the server boots on an empty database.
	StarterMapSeed int `env:"STARTER_MAP_SEED,?42"`
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

// Seed is STARTER_MAP_SEED, which must not be negative.
func (c Config) Seed() (uint64, error) {
	if c.StarterMapSeed < 0 {
		return 0, fmt.Errorf("STARTER_MAP_SEED %d is negative", c.StarterMapSeed)
	}
	return uint64(c.StarterMapSeed), nil
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
	if err := serve(context.Background(), config, logger); err != nil {
		log.Fatal(err.Error())
	}
}

// serve opens the database, migrates it, makes sure a map exists, then answers until SIGINT or SIGTERM.
func serve(ctx context.Context, config Config, log *slog.Logger) error {
	log.Info("donut API booting", "port", config.APIPort, "db", config.RedactedDBURL(), "front_origin", config.FrontOrigin,
		"starter_map_seed", config.StarterMapSeed, "log_level", config.LogLevel)
	seed, err := config.Seed()
	if err != nil {
		return err
	}
	pool, err := db.Open(ctx, config.DBURL, log)
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Debug("database connected")
	if err := migrateAtBoot(ctx, pool, log); err != nil {
		return err
	}
	handler, maps, err := routing(ctx, pool, config, log)
	if err != nil {
		return err
	}
	if _, err := maps.Service.EnsureStarterMap(ctx, seed); err != nil {
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

# Dev servers started together by `make dev`.
DEV_TARGETS := dev-back dev-front

# air reloads the back on every .go change (config: back/.air.toml).
# Uses the installed air, else runs a pinned one without installing it.
AIR := $(shell command -v air 2>/dev/null || echo "go run github.com/air-verse/air@v1.67.4")

TEST_DB_URL ?= postgres://donut:donut@localhost:5434/donut

.PHONY: help dev dev-back dev-front db test

help:
	@echo "make dev   - database + back (port 8400, reloads with air) + front (port 5180); ctrl-c stops all"
	@echo "make db    - start Donut's Postgres (docker, port 5434); the back migrates it at boot"
	@echo "make test  - back tests (with the database) and front tests"

# Runs every dev server in parallel; Ctrl+C stops them all.
dev: db
	@$(MAKE) --no-print-directory -j $(words $(DEV_TARGETS)) $(DEV_TARGETS)

dev-back:
	cd back && $(AIR)

dev-front: client/node_modules
	cd client && npm run dev

db:
	@docker compose up -d --wait db

test: db client/node_modules
	cd back && DONUT_TEST_DATABASE_URL=$(TEST_DB_URL) go test ./...
	cd client && npm run typecheck && npm test

# Reinstalls front dependencies only when the package files change.
client/node_modules: client/package.json client/package-lock.json
	cd client && npm install
	@touch $@

# Dev servers started together by `make dev`.
DEV_TARGETS := dev-back dev-front

# air reloads the back on every .go change (config: back/.air.toml).
# Uses the installed air, else runs a pinned one without installing it.
AIR := $(shell command -v air 2>/dev/null || echo "go run github.com/air-verse/air@v1.67.4")

TEST_DB_URL ?= postgres://donut:donut@localhost:5434/donut

.PHONY: help dev dev-back dev-front db test sprites world city check-world reset-world reset-maps

help:
	@echo "make dev   - database + back (port 8400, reloads with air) + front (port 5180); ctrl-c stops all"
	@echo "make db    - start Donut's Postgres (docker, port 5434); the back migrates it at boot"
	@echo "make test  - back tests (with the database) and front tests"
	@echo "make sprites - rebuild the robot sprite sheet from the Pixel Art Atlas (needs the network)"
	@echo "make world  - generate the starter map folder back/mapfiles/world; FORCE=1 overwrites hand edits"
	@echo "make city   - generate Doma-1's map folder back/mapfiles/doma-1; FORCE=1 overwrites hand edits"
	@echo "make check-world - check back/mapfiles/world as the server would, without a database"
	@echo "make reset-world - replace the world in the database with back/mapfiles/world (stop the server first)"
	@echo "make reset-maps - replace every map in the database with its folder in back/mapfiles (stop the server first)"

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

# The robot sheet (ADR 0006): downloads the picked atlas rows, recolors them, writes client/public/sprites/robots.*
sprites:
	cd back && go run ./cmd/spritesheet -picks cmd/spritesheet/robots.json -out ../client/public/sprites/robots

# Reinstalls front dependencies only when the package files change.
client/node_modules: client/package.json client/package-lock.json
	cd client && npm install
	@touch $@

# The starter map (ADR 0007): generated once, then edited by hand. Overwriting it needs FORCE=1.
WORLD_SEED ?= 42

world:
	cd back && go run ./cmd/mapfile generate -seed $(WORLD_SEED) -out mapfiles/world $(if $(FORCE),-force)

# Doma-1, the first dome city (ADR 0012): generated once, then edited by hand, like the world.
city:
	cd back && go run ./cmd/mapfile city -out mapfiles/doma-1 $(if $(FORCE),-force)

check-world:
	cd back && go run ./cmd/mapfile check mapfiles/world

# Rooms keep the map they loaded: run them with the server stopped.
reset-world: db
	cd back && go run ./cmd/donut reset-map world

# Every map folder of back/mapfiles, each in its own transaction (ADR 0012).
reset-maps: db
	cd back && go run ./cmd/donut reset-maps

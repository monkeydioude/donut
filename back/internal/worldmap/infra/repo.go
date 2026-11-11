// Package infra is the worldmap module persistence.
package infra

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	core "donut/internal/core/domain"
	coreinfra "donut/internal/core/infra"
	"donut/internal/worldmap/domain"
	"donut/pkg/pg"
)

type Repo struct {
	coreinfra.Repository
}

func NewRepo(db coreinfra.Requester) *Repo {
	return &Repo{Repository: coreinfra.Repository{DB: db}}
}

const summaryColumns = `id, slug, name, width, height, wraps`

// ListMaps is every map without its contents, by name.
func (r *Repo) ListMaps(ctx context.Context) ([]domain.Summary, error) {
	rows, err := r.DB.Query(ctx, `SELECT `+summaryColumns+` FROM maps ORDER BY name, id`)
	return pg.CollectMany[domain.Summary](rows, err, "ListMaps")
}

func (r *Repo) CountMaps(ctx context.Context) (int, error) {
	var count int
	err := r.DB.QueryRow(ctx, `SELECT count(*) FROM maps`).Scan(&count)
	return count, pg.Wrap(err, "CountMaps")
}

// DescriptionBySlug is a map's size and regions, without its cells.
func (r *Repo) DescriptionBySlug(ctx context.Context, slug string) (domain.Description, error) {
	rows, err := r.DB.Query(ctx, `SELECT `+summaryColumns+` FROM maps WHERE slug = $1`, slug)
	summary, err := pg.CollectOne[domain.Summary](rows, err, "DescriptionBySlug")
	if err != nil {
		return domain.Description{}, err
	}
	rows, err = r.DB.Query(ctx, `SELECT id, name, biome FROM map_regions WHERE map_id = $1 ORDER BY id`, summary.ID)
	regions, err := pg.CollectMany[domain.Region](rows, err, "DescriptionBySlug: regions")
	return domain.Description{Summary: summary, Regions: regions}, err
}

// MapBySlug is a whole map: its description, its chunks and its objects.
func (r *Repo) MapBySlug(ctx context.Context, slug string) (domain.Map, error) {
	description, err := r.DescriptionBySlug(ctx, slug)
	if err != nil {
		return domain.Map{}, fmt.Errorf("MapBySlug: %w", err)
	}
	m := domain.Map{Description: description}
	if m.Chunks, err = r.chunksOf(ctx, m.ID); err != nil {
		return domain.Map{}, err
	}
	rows, err := r.DB.Query(ctx, `SELECT id, kind, col, row FROM map_objects WHERE map_id = $1 ORDER BY row, col`, m.ID)
	m.Objects, err = pg.CollectMany[domain.Object](rows, err, "MapBySlug: objects")
	return m, err
}

func (r *Repo) chunksOf(ctx context.Context, mapID string) (map[core.ChunkKey]*domain.Chunk, error) {
	rows, err := r.DB.Query(ctx, `SELECT cx, cy, version, data FROM map_chunks WHERE map_id = $1`, mapID)
	if err != nil {
		return nil, pg.Wrap(err, "MapBySlug: chunks")
	}
	chunks := map[core.ChunkKey]*domain.Chunk{}
	var key core.ChunkKey
	var version int
	var data []byte
	_, err = pgx.ForEachRow(rows, []any{&key.CX, &key.CY, &version, &data}, func() error {
		chunk, err := DecodeChunk(key, version, data)
		chunks[key] = chunk
		return err
	})
	return chunks, pg.Wrap(err, "MapBySlug: chunks")
}

// InsertMap saves a whole map in one transaction, and returns its id.
func (r *Repo) InsertMap(ctx context.Context, m domain.Map) (string, error) {
	var id string
	err := r.RunInTransaction(ctx, func(ctx context.Context, tx coreinfra.Requester) error {
		var err error
		id, err = insertMap(ctx, tx, m)
		return err
	})
	return id, err
}

// ReplaceMap deletes the map of m's slug, with its chunks, regions and rocks, and saves m instead, in one
// transaction. It returns m's new id, and whether a map was deleted.
func (r *Repo) ReplaceMap(ctx context.Context, m domain.Map) (string, bool, error) {
	var id string
	var replaced bool
	err := r.RunInTransaction(ctx, func(ctx context.Context, tx coreinfra.Requester) error {
		tag, err := tx.Exec(ctx, `DELETE FROM maps WHERE slug = $1`, m.Slug)
		if err != nil {
			return pg.Wrap(err, "ReplaceMap")
		}
		replaced = tag.RowsAffected() > 0
		id, err = insertMap(ctx, tx, m)
		return err
	})
	return id, replaced, err
}

func insertMap(ctx context.Context, tx coreinfra.Requester, m domain.Map) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `INSERT INTO maps (slug, name, width, height, wraps) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		m.Slug, m.Name, m.Width, m.Height, m.Wraps).Scan(&id)
	if err != nil {
		return "", pg.Wrap(err, "InsertMap")
	}
	for _, insert := range []func(context.Context, coreinfra.Requester, string, domain.Map) error{insertRegions, insertChunks, insertObjects} {
		if err := insert(ctx, tx, id, m); err != nil {
			return "", err
		}
	}
	return id, nil
}

func insertRegions(ctx context.Context, tx coreinfra.Requester, mapID string, m domain.Map) error {
	ids, names, biomes := make([]int, len(m.Regions)), make([]string, len(m.Regions)), make([]string, len(m.Regions))
	for i, region := range m.Regions {
		ids[i], names[i], biomes[i] = region.ID, region.Name, string(region.Biome)
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO map_regions (map_id, id, name, biome)
		SELECT $1, id, name, biome FROM unnest($2::smallint[], $3::text[], $4::text[]) AS r(id, name, biome)`,
		mapID, ids, names, biomes)
	return pg.Wrap(err, "InsertMap: regions")
}

func insertChunks(ctx context.Context, tx coreinfra.Requester, mapID string, m domain.Map) error {
	var cxs, cys, versions []int
	var data [][]byte
	for key, chunk := range m.Chunks {
		cxs, cys, versions = append(cxs, key.CX), append(cys, key.CY), append(versions, chunk.Version)
		data = append(data, EncodeChunk(chunk))
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO map_chunks (map_id, cx, cy, version, data)
		SELECT $1, cx, cy, version, data FROM unnest($2::int[], $3::int[], $4::int[], $5::bytea[]) AS c(cx, cy, version, data)`,
		mapID, cxs, cys, versions, data)
	return pg.Wrap(err, "InsertMap: chunks")
}

func insertObjects(ctx context.Context, tx coreinfra.Requester, mapID string, m domain.Map) error {
	kinds, cols, rows := make([]string, len(m.Objects)), make([]int, len(m.Objects)), make([]int, len(m.Objects))
	for i, object := range m.Objects {
		kinds[i], cols[i], rows[i] = string(object.Kind), object.Col, object.Row
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO map_objects (map_id, kind, col, row)
		SELECT $1, kind, col, row FROM unnest($2::text[], $3::int[], $4::int[]) AS o(kind, col, row)`,
		mapID, kinds, cols, rows)
	return pg.Wrap(err, "InsertMap: objects")
}

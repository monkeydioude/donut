-- A donut-shaped world in chunks (ADR 0003): maps lose their whole-map ground for chunks of 32 x 32 cells, may wrap
-- around, and are cut into named regions. The maps made so far were dev data: they go, and the server generates the
-- new starter world at its next boot.
SET search_path TO donut;

DELETE FROM maps; -- their rocks go with them

-- dropping the column drops the check on its length too
ALTER TABLE maps DROP COLUMN ground;
ALTER TABLE maps ADD COLUMN wraps BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE maps DROP CONSTRAINT maps_width_check, DROP CONSTRAINT maps_height_check;
ALTER TABLE maps
  ADD CONSTRAINT maps_width_check CHECK (width BETWEEN 32 AND 4096 AND width % 32 = 0),
  ADD CONSTRAINT maps_height_check CHECK (height BETWEEN 32 AND 4096 AND height % 32 = 0),
  -- a wrapping map is at least 5 chunks on each side
  ADD CONSTRAINT maps_wraps_size_check CHECK (NOT wraps OR (width >= 160 AND height >= 160));

CREATE TABLE map_regions (
  map_id UUID NOT NULL REFERENCES maps (id) ON DELETE CASCADE,
  id     SMALLINT NOT NULL CHECK (id BETWEEN 1 AND 255),
  name   TEXT NOT NULL CHECK (name <> ''),
  biome  TEXT NOT NULL,
  PRIMARY KEY (map_id, id)
);

-- A chunk with no row is all void.
CREATE TABLE map_chunks (
  map_id     UUID NOT NULL REFERENCES maps (id) ON DELETE CASCADE,
  cx         INT NOT NULL CHECK (cx >= 0),
  cy         INT NOT NULL CHECK (cy >= 0),
  -- goes up each time the chunk changes
  version    INT NOT NULL DEFAULT 1 CHECK (version >= 1),
  -- format 1: 1 format byte, then 1,024 ground codes, 1,024 signed heights and 1,024 region ids, row by row
  data       BYTEA NOT NULL CHECK (length(data) = 3073 AND get_byte(data, 0) = 1),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (map_id, cx, cy)
);

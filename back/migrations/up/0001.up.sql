-- Maps and their layers (ADR 0001): the ground grid on the map, one row per object standing on a cell.
SET search_path TO donut;

CREATE TABLE maps (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug       TEXT NOT NULL UNIQUE,
  name       TEXT NOT NULL,
  width      INT NOT NULL CHECK (width BETWEEN 1 AND 256),
  height     INT NOT NULL CHECK (height BETWEEN 1 AND 256),
  -- one tile kind per cell, row by row: width × height entries
  ground     JSONB NOT NULL CHECK (jsonb_typeof(ground) = 'array' AND jsonb_array_length(ground) = width * height),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE map_objects (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  map_id     UUID NOT NULL REFERENCES maps (id) ON DELETE CASCADE,
  kind       TEXT NOT NULL,
  col        INT NOT NULL CHECK (col >= 0),
  row        INT NOT NULL CHECK (row >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (map_id, col, row)
);

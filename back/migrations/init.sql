-- Donut schema and the migrations' bookkeeping. Tables come in up/*.sql, one file per change.
--
-- The schema is created only when it is missing: in production an admin may create it beforehand and give the
-- app's user only USAGE + CREATE on it, and a plain CREATE SCHEMA IF NOT EXISTS checks the database right first.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'donut') THEN
    CREATE SCHEMA donut;
  END IF;
END
$$;
SET search_path TO donut;

CREATE TABLE IF NOT EXISTS schema_migrations (
  filename   TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Doma-1 (ADR 0012): a dome on the world map leads to its city's map. Only domes lead somewhere. Nothing is deleted:
-- `make reset-maps` replaces the maps with their files.
SET search_path TO donut;

ALTER TABLE map_objects
  ADD COLUMN leads_to TEXT,
  ADD CONSTRAINT map_objects_only_domes_lead CHECK ((kind = 'dome') = (leads_to IS NOT NULL));

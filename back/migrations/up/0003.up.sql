-- Terrain (ADR 0007): chunks get water, holes and bridges (ground codes 2, 3 and 4), and heights from -3 to 3.
-- The starter world now comes from its map file: the maps made so far go, and the server loads the file at its next
-- boot.
SET search_path TO donut;

DELETE FROM maps; -- their chunks, regions and rocks go with them

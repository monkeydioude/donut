-- Continents, islands and an ocean (ADR 0008): the starter world is 2048 x 1024 with a new ground code 5, ocean.
-- The maps made so far go, and the server loads the new map file at its next boot.
SET search_path TO donut;

DELETE FROM maps; -- their chunks, regions and rocks go with them

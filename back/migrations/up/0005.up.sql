-- Mountains and glaciers (ADR 0009): the starter world gets the ground codes 6, mountain, and 7, glacier.
-- The maps made so far go, and the server loads the new map file at its next boot.
SET search_path TO donut;

DELETE FROM maps; -- their chunks, regions and rocks go with them

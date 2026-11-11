-- Massifs and the glacier ring (ADR 0011): heights go up to +6, and the starter world is generated again.
-- The maps made so far go, and the server loads the new map file at its next boot.
SET search_path TO donut;

DELETE FROM maps; -- their chunks, regions and rocks go with them

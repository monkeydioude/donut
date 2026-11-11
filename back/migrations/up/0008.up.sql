-- Doma-1 (ADR 0012): a city's description looks for the dome leading to it, among a world's tens of thousands of
-- rocks. Only domes lead somewhere, so the index holds only them.
SET search_path TO donut;

CREATE INDEX map_objects_leads_to ON map_objects (leads_to) WHERE leads_to IS NOT NULL;

-- A contest can be hidden from the archive list. Direct URLs keep working.
-- SQLite requires a non-NULL default on a NOT NULL column added by ALTER TABLE;
-- existing contests therefore stay listed.
ALTER TABLE contests ADD COLUMN unlisted BOOLEAN NOT NULL DEFAULT FALSE;

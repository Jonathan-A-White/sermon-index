-- A source's `by` (source.toml `by`): the name its sermons are by, for a row
-- that names no preacher (a church's name, for a source that is one church's
-- sermons). NULL for a source that sets none.
--
-- It is served with the source (/v1/sources) and never written into a row's
-- preacher, so search and the tutor never take it for a preacher. `sermons
-- load` writes it: run `sermons load --all sources/` after `sermons migrate`.

ALTER TABLE sources ADD COLUMN by text CHECK (by IS NULL OR btrim(by) <> '');

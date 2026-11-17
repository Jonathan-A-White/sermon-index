-- A source's `text` (source.toml `text`): "public-domain" for a source whose
-- sermons are public-domain text that /v1/text may fetch from the source's
-- page on demand and serve. NULL for a source that sets none (its words are
-- never served).
--
-- `sermons load` writes it: run `sermons load --all sources/` after `sermons
-- migrate`.

ALTER TABLE sources ADD COLUMN text text CHECK (text IS NULL OR text = 'public-domain');

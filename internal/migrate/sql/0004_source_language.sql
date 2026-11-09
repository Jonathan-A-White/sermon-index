-- A source's language (source.toml `language`, a two-letter ISO 639-1 code
-- from a short list, internal/sources). 'en' is the default, so every source
-- loaded before this migration, and every one that does not say, is English.
--
-- `sermons load` writes it. Stored and checked only; nothing filters or shows
-- it yet. Run `sermons load --all sources/` after `sermons migrate` so that
-- calvin-pt-logos says 'pt'.

ALTER TABLE sources ADD COLUMN language text NOT NULL DEFAULT 'en' CHECK (language ~ '^[a-z]{2}$');

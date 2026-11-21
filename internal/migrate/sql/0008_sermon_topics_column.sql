-- A sermon row's `topics` (rows.jsonl `topics`): the free subject labels of a
-- sermon, for one with no passage to hang a theme on (the Luther sermons whose
-- heading prints no text). NULL for a row that has none. It is not
-- sermon_topics, which links a sermon to a topic id of a theme source.
--
-- `sermons load` writes it: run `sermons load --all sources/` after `sermons
-- migrate`.

ALTER TABLE sermons ADD COLUMN topics text[];

-- Direct links from a sermon to a topic (story mw-qtu3zvz.59.1): a sermon
-- source's optional sermon-topics.jsonl, one (row, theme source, topic) a
-- line, chosen by a person (or a model reading the sermon's notes) rather than
-- found through the sermon's preaching text. The theme tier of /v1/sermons
-- answers them first, marked "direct": true (docs/api.md).
--
-- No foreign key to themes, on purpose: `sermons load --all` goes by directory
-- name, so a sermon source (brookfield-first) loads before the theme sources
-- (themes-*) its links name, and a theme source's reload must never silently
-- delete a link a person chose. A link to a topic that is not loaded matches
-- nothing; `sermons check` makes sure the topic is in the theme source's
-- files. The key to sermons stays: deleting a sermon deletes its links.
--
-- `sermons load` writes the table. A source without the file has no links, so
-- no reload is needed after `sermons migrate` until a source gains the file.

CREATE TABLE sermon_topics (
    source_id       text NOT NULL,
    sermon_id       text NOT NULL,
    theme_source_id text NOT NULL,
    theme_id        text NOT NULL,
    PRIMARY KEY (source_id, sermon_id, theme_source_id, theme_id),
    FOREIGN KEY (source_id, sermon_id) REFERENCES sermons (source_id, id) ON DELETE CASCADE
);
CREATE INDEX sermon_topics_theme ON sermon_topics (theme_source_id, theme_id);

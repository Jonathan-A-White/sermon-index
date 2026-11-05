-- Search (story A2): a generated tsvector on sermons (title, then preacher)
-- and on themes (label), each with a GIN index, for /v1/search's
-- websearch_to_tsquery. The columns are generated, so `sermons load` writes
-- nothing new and a reload keeps them right.
--
-- The 'english' configuration stems ("graces" finds "Grace") and drops stop
-- words. A theme's part labels ("Of God") are in another table and are not
-- searched.

ALTER TABLE sermons ADD COLUMN search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('english', title), 'A') ||
    setweight(to_tsvector('english', coalesce(preacher, '')), 'B')
) STORED;
CREATE INDEX sermons_search ON sermons USING gin (search);

ALTER TABLE themes ADD COLUMN search tsvector GENERATED ALWAYS AS (
    to_tsvector('english', label)
) STORED;
CREATE INDEX themes_search ON themes USING gin (search);

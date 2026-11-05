-- Schema v1 of the sermon index. docs/schema.md draws these tables and says
-- why each choice was made. Every row here is loaded from a file under
-- sources/ by `sermons load`; the files are the truth and the database is
-- their index, so keys are the files' own text ids, never surrogate numbers.
--
-- Conventions: an absent optional value is NULL, never '' (a CHECK keeps one
-- way to say "none"); a passage is an int4range of verse ordinals
-- (book*1000000 + chapter*1000 + verse, Lampas's book order) in canonical
-- form [first, last+1).

-- A source: one directory under sources/, a sermon list or a theme system.
CREATE TABLE sources (
    id            text PRIMARY KEY CHECK (id ~ '^[a-z0-9][a-z0-9-]*$'),
    name          text NOT NULL CHECK (name <> ''),
    kind          text NOT NULL CHECK (kind IN ('sermons', 'themes')),
    licence       text NOT NULL CHECK (btrim(licence) <> ''),
    licence_note  text CHECK (licence_note <> ''),
    credit        text NOT NULL CHECK (credit <> ''),
    homepage      text NOT NULL CHECK (homepage LIKE 'https://%'),
    terms_url     text NOT NULL CHECK (terms_url LIKE 'https://%'),
    added         date NOT NULL,
    on_by_default boolean NOT NULL,
    loaded_at     timestamptz NOT NULL DEFAULT now(),
    -- The target of the kind-checked foreign keys below.
    UNIQUE (id, kind)
);

-- The theme sources, as a view: one sources table holds both kinds, so a
-- licence and a credit are written once and read the same way.
CREATE VIEW theme_sources AS
    SELECT id, name, licence, licence_note, credit, homepage, terms_url, added, on_by_default, loaded_at
    FROM sources WHERE kind = 'themes';

-- A sermon: one row of a sermon source's rows.jsonl.
CREATE TABLE sermons (
    source_id   text NOT NULL,
    -- Always 'sermons': with the foreign key it keeps a sermon out of a
    -- theme source.
    source_kind text NOT NULL DEFAULT 'sermons' CHECK (source_kind = 'sermons'),
    id          text NOT NULL CHECK (id ~ '^[A-Za-z0-9._-]+$'),
    title       text NOT NULL CHECK (title <> ''),
    preacher    text CHECK (preacher <> ''),
    url         text NOT NULL CHECK (url LIKE 'https://%'),
    alt_url     text CHECK (alt_url LIKE 'https://%'),
    kind        text NOT NULL DEFAULT 'link' CHECK (kind IN ('link', 'hosted', 'exposition')),
    -- ISO 8601 at the precision known: '1855', '1855-03' or '1855-03-18'.
    date        text CHECK (date ~ '^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$'),
    duration_s  integer CHECK (duration_s > 0),
    notes       text CHECK (notes <> ''),
    PRIMARY KEY (source_id, id),
    FOREIGN KEY (source_id, source_kind) REFERENCES sources (id, kind) ON DELETE CASCADE
);

-- A sermon's passages: its preaching text ('text') and the references it
-- cites ('cited'), each as a range of verse ordinals.
CREATE TABLE sermon_passages (
    source_id text NOT NULL,
    sermon_id text NOT NULL,
    ref_osis  text NOT NULL CHECK (ref_osis <> ''),
    role      text NOT NULL CHECK (role IN ('text', 'cited')),
    span      int4range NOT NULL CHECK (NOT isempty(span) AND NOT lower_inf(span) AND NOT upper_inf(span)),
    PRIMARY KEY (source_id, sermon_id, role, ref_osis),
    FOREIGN KEY (source_id, sermon_id) REFERENCES sermons (source_id, id) ON DELETE CASCADE
);
CREATE INDEX sermon_passages_span ON sermon_passages USING gist (span);

-- A topic of a theme source (topics.jsonl). see holds the ids of related
-- topics: a bare id is a topic of the same source, '<source>:<id>' another's.
CREATE TABLE themes (
    source_id   text NOT NULL,
    source_kind text NOT NULL DEFAULT 'themes' CHECK (source_kind = 'themes'),
    id          text NOT NULL CHECK (id ~ '^[A-Za-z0-9._-]+$'),
    label       text NOT NULL CHECK (label <> ''),
    see         text[] NOT NULL DEFAULT '{}',
    PRIMARY KEY (source_id, id),
    FOREIGN KEY (source_id, source_kind) REFERENCES sources (id, kind) ON DELETE CASCADE
);

-- A topic's numbered parts ("(4) Of God"), which a link may name.
CREATE TABLE theme_parts (
    theme_source_id text NOT NULL,
    theme_id        text NOT NULL,
    n               integer NOT NULL CHECK (n > 0),
    label           text NOT NULL CHECK (label <> ''),
    PRIMARY KEY (theme_source_id, theme_id, n),
    FOREIGN KEY (theme_source_id, theme_id) REFERENCES themes (source_id, id) ON DELETE CASCADE
);

-- A topic joined to a passage (links.jsonl). via says where the link came
-- from: the source itself, a reading of the Bible, or a sermon (from_ref
-- names it as '<source>:<sermon id>').
CREATE TABLE theme_links (
    source_id text NOT NULL,
    theme_id  text NOT NULL,
    part_n    integer,
    ref_osis  text NOT NULL CHECK (ref_osis <> ''),
    span      int4range NOT NULL CHECK (NOT isempty(span) AND NOT lower_inf(span) AND NOT upper_inf(span)),
    via       text NOT NULL DEFAULT 'source' CHECK (via IN ('source', 'bible', 'sermon')),
    from_ref  text CHECK (from_ref <> ''),
    CHECK (via <> 'sermon' OR from_ref IS NOT NULL),
    FOREIGN KEY (source_id, theme_id) REFERENCES themes (source_id, id) ON DELETE CASCADE,
    -- MATCH SIMPLE: a link with no part (part_n NULL) is not checked here.
    FOREIGN KEY (source_id, theme_id, part_n) REFERENCES theme_parts (theme_source_id, theme_id, n) ON DELETE CASCADE,
    -- The link's natural key. NULLS NOT DISTINCT (Postgres 15+) makes a
    -- link with no part or no from_ref a key like any other, so a reload's
    -- ON CONFLICT finds it.
    CONSTRAINT theme_links_key UNIQUE NULLS NOT DISTINCT (source_id, theme_id, part_n, ref_osis, via, from_ref)
);
CREATE INDEX theme_links_span ON theme_links USING gist (span);

-- The theme tier (story A5): a topic's width, the verses its links cover
-- together, so a query can leave out a whole-subject topic (Nave's puts 2,735
-- sermons on Romans 5) without counting verses from ordinals in SQL.
--
-- `sermons load` writes it (internal/themes), from the book table. NULL means
-- the topic has not been loaded since this migration: the theme tier leaves
-- such a topic out, so run `sermons load --all sources/` after `sermons migrate`.

ALTER TABLE themes ADD COLUMN verses integer CHECK (verses >= 0);

package sources

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Jonathan-A-White/sermon-index/internal/migrate"
	"github.com/Jonathan-A-White/sermon-index/internal/themes"
)

// Counts is what a load left in the database for one source.
type Counts struct {
	Source string
	Kind   string
	// A sermon source: its rows and their passages (preaching texts and
	// cited references), and its direct links to topics
	// (sermon-topics.jsonl; HasTopicLinks is whether it has the file).
	Rows, Passages int
	TopicLinks     int
	HasTopicLinks  bool
	// A theme source: its topics, their parts, and links.
	Topics, Parts, Links int
	// Removed: rows of the source in the database that are no longer in its
	// files (for a sermon source its sermons; for a theme source its topics
	// and links).
	Removed int
}

func (c Counts) String() string {
	if c.Kind == KindThemes {
		return fmt.Sprintf("%s: %d topics, %d parts, %d links, %d removed", c.Source, c.Topics, c.Parts, c.Links, c.Removed)
	}
	links := ""
	if c.HasTopicLinks {
		links = fmt.Sprintf(", %d topic links", c.TopicLinks)
	}
	return fmt.Sprintf("%s: %d rows, %d passages%s, %d removed", c.Source, c.Rows, c.Passages, links, c.Removed)
}

// Load checks a source directory and, when it has no fault, makes the
// database hold exactly it, in one transaction: the source and each row are
// upserted by their key and the source's rows no longer in the files are
// deleted. Loading the same files twice changes nothing. A directory with
// faults is refused with Faults and the database is not touched.
func Load(ctx context.Context, db *pgxpool.Pool, dir string) (Counts, error) {
	d, faults := checked(dir)
	if len(faults) > 0 {
		return Counts{}, Faults(faults)
	}
	if err := needSchema(ctx, db); err != nil {
		return Counts{}, err
	}
	return load(ctx, db, d)
}

// LoadAll checks every source directory under root (skipping those whose
// names begin with '_' or '.') and, when none has a fault, loads each in
// its own transaction.
func LoadAll(ctx context.Context, db *pgxpool.Pool, root string) ([]Counts, error) {
	dirs, faults := CheckAll(root)
	if len(faults) > 0 {
		return nil, Faults(faults)
	}
	if err := needSchema(ctx, db); err != nil {
		return nil, err
	}
	var out []Counts
	for _, dir := range dirs {
		d, faults := checked(dir)
		if len(faults) > 0 { // changed on disk since CheckAll
			return out, Faults(faults)
		}
		c, err := load(ctx, db, d)
		if err != nil {
			return out, fmt.Errorf("%s: %w", filepath.Base(dir), err)
		}
		out = append(out, c)
	}
	return out, nil
}

func needSchema(ctx context.Context, db *pgxpool.Pool) error {
	pending, err := migrate.Pending(ctx, db)
	if err != nil {
		return err
	}
	if len(pending) > 0 {
		return fmt.Errorf("the database lacks migrations %s: run `sermons migrate` first", strings.Join(pending, ", "))
	}
	return nil
}

func load(ctx context.Context, db *pgxpool.Pool, d *Dir) (Counts, error) {
	c := Counts{Source: d.Source.ID, Kind: d.Source.Kind}
	err := pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		if err := upsertSource(ctx, tx, d.Source); err != nil {
			return err
		}
		switch d.Source.Kind {
		case KindSermons:
			return writeSermons(ctx, tx, d, &c)
		case KindThemes:
			tc, err := themes.Write(ctx, tx, d.Source.ID, d.Topics, d.Links)
			c.Topics, c.Parts, c.Links, c.Removed = tc.Topics, tc.Parts, tc.Links, tc.Removed
			return err
		}
		return fmt.Errorf("kind %q", d.Source.Kind)
	})
	if err != nil {
		return Counts{}, err
	}
	return c, nil
}

func nullable(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func upsertSource(ctx context.Context, tx pgx.Tx, s Source) error {
	_, err := tx.Exec(ctx, `INSERT INTO sources
			(id, name, kind, licence, licence_note, credit, homepage, terms_url, added, on_by_default, language)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name, kind = EXCLUDED.kind, licence = EXCLUDED.licence,
			licence_note = EXCLUDED.licence_note, credit = EXCLUDED.credit, homepage = EXCLUDED.homepage,
			terms_url = EXCLUDED.terms_url, added = EXCLUDED.added, on_by_default = EXCLUDED.on_by_default,
			language = EXCLUDED.language, loaded_at = now()`,
		s.ID, s.Name, s.Kind, s.Licence, nullable(s.LicenceNote), s.Credit, s.Homepage, s.TermsURL,
		s.Added, s.OnByDefault, s.LanguageOrDefault())
	if err != nil {
		return fmt.Errorf("writing the source: %w", err)
	}
	return nil
}

// Passage roles.
const (
	RoleText  = "text"
	RoleCited = "cited"
)

func writeSermons(ctx context.Context, tx pgx.Tx, d *Dir, c *Counts) error {
	id := d.Source.ID
	ids := make([]string, 0, len(d.Rows))
	b := &pgx.Batch{}
	for _, r := range d.Rows {
		ids = append(ids, r.ID)
		b.Queue(`INSERT INTO sermons (source_id, id, title, preacher, url, alt_url, kind, date, duration_s, notes)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (source_id, id) DO UPDATE SET
				title = EXCLUDED.title, preacher = EXCLUDED.preacher, url = EXCLUDED.url,
				alt_url = EXCLUDED.alt_url, kind = EXCLUDED.kind, date = EXCLUDED.date,
				duration_s = EXCLUDED.duration_s, notes = EXCLUDED.notes
			WHERE (sermons.title, sermons.preacher, sermons.url, sermons.alt_url, sermons.kind,
			       sermons.date, sermons.duration_s, sermons.notes)
			      IS DISTINCT FROM (EXCLUDED.title, EXCLUDED.preacher, EXCLUDED.url, EXCLUDED.alt_url,
			       EXCLUDED.kind, EXCLUDED.date, EXCLUDED.duration_s, EXCLUDED.notes)`,
			id, r.ID, r.Title, r.Preacher, r.URL, r.AltURL, r.KindOrDefault(), r.Date, r.DurationS, r.Notes)
	}
	if err := tx.SendBatch(ctx, b).Close(); err != nil {
		return fmt.Errorf("writing sermons: %w", err)
	}

	tag, err := tx.Exec(ctx, `DELETE FROM sermons WHERE source_id = $1 AND id <> ALL($2::text[])`, id, ids)
	if err != nil {
		return fmt.Errorf("removing sermons: %w", err)
	}
	c.Removed = int(tag.RowsAffected())

	// The passages, one per (sermon, role, reference); a reference given
	// twice in a row is kept once.
	type key struct{ sermon, role, ref string }
	seen := map[key]bool{}
	var kSermon, kRole, kRef, kSpan []string
	addPassage := func(sermon, role, ref, span string) {
		k := key{sermon, role, ref}
		if seen[k] {
			return
		}
		seen[k] = true
		kSermon, kRole, kRef, kSpan = append(kSermon, sermon), append(kRole, role), append(kRef, ref), append(kSpan, span)
	}
	for _, r := range d.Rows {
		if r.IsSeries() {
			continue // a series is kept as a sermon but has no passage: nothing a passage asks for matches it
		}
		if r.text != nil {
			addPassage(r.ID, RoleText, r.text.OSIS, r.text.PG())
		}
		for _, rg := range r.cited {
			addPassage(r.ID, RoleCited, rg.OSIS, rg.PG())
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM sermon_passages p WHERE p.source_id = $1 AND NOT EXISTS (
			SELECT 1 FROM unnest($2::text[], $3::text[], $4::text[]) AS k(sermon, role, ref)
			WHERE k.sermon = p.sermon_id AND k.role = p.role AND k.ref = p.ref_osis)`,
		id, kSermon, kRole, kRef); err != nil {
		return fmt.Errorf("removing passages: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO sermon_passages (source_id, sermon_id, role, ref_osis, span)
			SELECT $1, k.sermon, k.role, k.ref, k.span::int4range
			FROM unnest($2::text[], $3::text[], $4::text[], $5::text[]) AS k(sermon, role, ref, span)
		ON CONFLICT (source_id, sermon_id, role, ref_osis) DO UPDATE SET span = EXCLUDED.span
			WHERE sermon_passages.span IS DISTINCT FROM EXCLUDED.span`,
		id, kSermon, kRole, kRef, kSpan); err != nil {
		return fmt.Errorf("writing passages: %w", err)
	}

	// The direct links to topics: the file's, and none when there is no
	// file. No key to themes: a topic not loaded (yet) matches nothing.
	var tSermon, tTheme, tTopic []string
	for _, st := range d.SermonTopics {
		tSermon, tTheme, tTopic = append(tSermon, st.Row), append(tTheme, st.Theme), append(tTopic, st.Topic)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM sermon_topics s WHERE s.source_id = $1 AND NOT EXISTS (
			SELECT 1 FROM unnest($2::text[], $3::text[], $4::text[]) AS k(sermon, theme, topic)
			WHERE k.sermon = s.sermon_id AND k.theme = s.theme_source_id AND k.topic = s.theme_id)`,
		id, tSermon, tTheme, tTopic); err != nil {
		return fmt.Errorf("removing topic links: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO sermon_topics (source_id, sermon_id, theme_source_id, theme_id)
			SELECT $1, k.sermon, k.theme, k.topic FROM unnest($2::text[], $3::text[], $4::text[]) AS k(sermon, theme, topic)
		ON CONFLICT DO NOTHING`,
		id, tSermon, tTheme, tTopic); err != nil {
		return fmt.Errorf("writing topic links: %w", err)
	}

	c.Rows, c.Passages = len(d.Rows), len(kSermon)
	c.TopicLinks, c.HasTopicLinks = len(d.SermonTopics), d.HasSermonTopics
	return nil
}

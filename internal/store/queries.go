package store

// The API's reads (story A2). Each function is one question /v1 asks, and
// its row type is the JSON the API answers (docs/api.md): the json tags
// here are the contract Lampas codes against, so a change to one is a
// change to the API.

import (
	"context"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
)

// Totals is what /v1/health counts.
type Totals struct {
	Sources int `json:"sources"`
	Sermons int `json:"sermons"`
}

// Count counts the sources (both kinds) and the sermons.
func Count(ctx context.Context, db *pgxpool.Pool) (Totals, error) {
	var t Totals
	err := db.QueryRow(ctx, `SELECT (SELECT count(*) FROM sources), (SELECT count(*) FROM sermons)`).Scan(&t.Sources, &t.Sermons)
	return t, err
}

// Source is a row of /v1/sources. Rows is a sermon source's sermons or a
// theme source's topics.
type Source struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Licence     string `json:"licence"`
	Credit      string `json:"credit"`
	Homepage    string `json:"homepage"`
	OnByDefault bool   `json:"on_by_default"`
	Rows        int    `json:"rows"`
}

// Sources lists every source, sermon sources first, then by id.
func Sources(ctx context.Context, db *pgxpool.Pool) ([]Source, error) {
	rows, err := db.Query(ctx, `
		SELECT s.id, s.name, s.kind, s.licence, s.credit, s.homepage, s.on_by_default,
		       CASE s.kind WHEN 'sermons' THEN (SELECT count(*) FROM sermons m WHERE m.source_id = s.id)
		                   ELSE (SELECT count(*) FROM themes t WHERE t.source_id = s.id) END
		FROM sources s
		ORDER BY s.kind, s.id`)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Source, error) {
		var s Source
		err := r.Scan(&s.ID, &s.Name, &s.Kind, &s.Licence, &s.Credit, &s.Homepage, &s.OnByDefault, &s.Rows)
		return s, err
	})
	if out == nil {
		out = []Source{}
	}
	return out, err
}

// SermonQuery is /v1/sermons's question: the sermons whose passages overlap
// Span, from Sources only when it is not empty.
type SermonQuery struct {
	Span          passage.Range
	Sources       []string
	Limit, Offset int
}

// SermonRow is a row of /v1/sermons: a sermon and the one passage of it
// that answered (its preaching text when that overlaps, else the narrowest
// reference it cites that does). Width is that passage's verses.
type SermonRow struct {
	Source   string  `json:"source"`
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Preacher *string `json:"preacher"`
	Date     *string `json:"date"`
	URL      string  `json:"url"`
	AltURL   *string `json:"alt_url"`
	Kind     string  `json:"kind"`
	Ref      string  `json:"ref"`
	Role     string  `json:"role"`
	Width    int     `json:"width"`
}

// The order of a passage answer, in SQL over a row with role, lo and hi
// (the span's bounds): the preaching text before a cited reference, then
// narrower before wider by the ordinal width (so a passage within one
// chapter comes before any that crosses a chapter, docs/schema.md), then
// the source, then the id with a number as a number ('9' before '10').
const naturalID = `CASE WHEN %[1]s ~ '^[0-9]{1,9}$' THEN %[1]s::int END NULLS LAST, %[1]s`

// best is each sermon's one answering passage; it takes $1 the span, $2
// whether to filter by source and $3 the sources.
const bestPassage = `
	SELECT DISTINCT ON (p.source_id, p.sermon_id)
	       p.source_id, p.sermon_id, p.ref_osis, p.role, lower(p.span) AS lo, upper(p.span) AS hi
	FROM sermon_passages p
	WHERE p.span && $1::int4range AND (NOT $2::bool OR p.source_id = ANY($3::text[]))
	ORDER BY p.source_id, p.sermon_id, p.role = 'text' DESC, upper(p.span) - lower(p.span), p.ref_osis`

// Sermons answers q: how many sermons overlap it, and the page of them
// Limit and Offset name, in the order above. Rows is never nil.
func Sermons(ctx context.Context, db *pgxpool.Pool, q SermonQuery) (int, []SermonRow, error) {
	filter := len(q.Sources) > 0
	srcs := q.Sources
	if srcs == nil {
		srcs = []string{}
	}
	var total int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM (`+bestPassage+`) b`, q.Span.PG(), filter, srcs).Scan(&total); err != nil {
		return 0, nil, err
	}
	rows, err := db.Query(ctx, `
		WITH b AS (`+bestPassage+`)
		SELECT s.source_id, s.id, s.title, s.preacher, s.date, s.url, s.alt_url, s.kind, b.ref_osis, b.role, b.lo, b.hi
		FROM b JOIN sermons s ON s.source_id = b.source_id AND s.id = b.sermon_id
		ORDER BY b.role = 'text' DESC, b.hi - b.lo, s.source_id, `+fmt.Sprintf(naturalID, "s.id")+`
		LIMIT $4 OFFSET $5`, q.Span.PG(), filter, srcs, q.Limit, q.Offset)
	if err != nil {
		return 0, nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (SermonRow, error) {
		var s SermonRow
		var lo, hi int
		err := r.Scan(&s.Source, &s.ID, &s.Title, &s.Preacher, &s.Date, &s.URL, &s.AltURL, &s.Kind, &s.Ref, &s.Role, &lo, &hi)
		s.Width = Verses(lo, hi-1)
		return s, err
	})
	if out == nil {
		out = []SermonRow{}
	}
	return total, out, err
}

// ThemeRow is a row of /v1/themes: a topic (or one numbered part of it)
// with a link overlapping the passage asked about; Ref is the narrowest
// such link. See is never nil.
type ThemeRow struct {
	Source    string   `json:"source"`
	Topic     string   `json:"topic"`
	Label     string   `json:"label"`
	Part      *int     `json:"part"`
	PartLabel *string  `json:"part_label"`
	Ref       string   `json:"ref"`
	See       []string `json:"see"`
}

// Themes answers the topics whose links overlap span: grouped by theme
// source (by its id), narrower first within a source, then by label and
// part. A topic with links to two of its parts is two rows.
func Themes(ctx context.Context, db *pgxpool.Pool, span passage.Range) ([]ThemeRow, error) {
	rows, err := db.Query(ctx, `
		WITH b AS (
			SELECT DISTINCT ON (l.source_id, l.theme_id, l.part_n)
			       l.source_id, l.theme_id, l.part_n, l.ref_osis, upper(l.span) - lower(l.span) AS w
			FROM theme_links l
			WHERE l.span && $1::int4range
			ORDER BY l.source_id, l.theme_id, l.part_n, upper(l.span) - lower(l.span), l.ref_osis)
		SELECT b.source_id, b.theme_id, t.label, b.part_n, p.label, b.ref_osis, t.see
		FROM b
		JOIN themes t ON t.source_id = b.source_id AND t.id = b.theme_id
		LEFT JOIN theme_parts p ON p.theme_source_id = b.source_id AND p.theme_id = b.theme_id AND p.n = b.part_n
		ORDER BY b.source_id, b.w, t.label, b.theme_id, b.part_n NULLS FIRST`, span.PG())
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (ThemeRow, error) {
		var t ThemeRow
		var part *int32
		err := r.Scan(&t.Source, &t.Topic, &t.Label, &part, &t.PartLabel, &t.Ref, &t.See)
		if part != nil {
			n := int(*part)
			t.Part = &n
		}
		if t.See == nil {
			t.See = []string{}
		}
		return t, err
	})
	if out == nil {
		out = []ThemeRow{}
	}
	return out, err
}

// SearchLimit is the most rows /v1/search answers.
const SearchLimit = 50

// Hit is a row of /v1/search: a sermon (Ref its preaching text, if it has
// one) or a theme (Title its label; no Ref or URL). Rank is ts_rank to
// four places.
type Hit struct {
	Kind   string  `json:"kind"`
	Source string  `json:"source"`
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Ref    *string `json:"ref"`
	URL    *string `json:"url"`
	Rank   float64 `json:"rank"`
}

// Search is Postgres full-text search of q (websearch_to_tsquery, 'english')
// over the sermons' titles and preachers and the themes' labels: the best
// SearchLimit, by rank, then sermons before themes, the source and the id.
// A q of stop words alone answers nothing. Rows is never nil.
func Search(ctx context.Context, db *pgxpool.Pool, q string) ([]Hit, error) {
	rows, err := db.Query(ctx, `
		WITH q AS (SELECT websearch_to_tsquery('english', $1) AS q),
		h AS (
			SELECT 'sermon' AS kind, s.source_id, s.id, s.title,
			       (SELECT p.ref_osis FROM sermon_passages p
			        WHERE p.source_id = s.source_id AND p.sermon_id = s.id AND p.role = 'text'
			        ORDER BY lower(p.span), p.ref_osis LIMIT 1) AS ref,
			       s.url, ts_rank(s.search, q.q) AS rank
			FROM sermons s, q WHERE s.search @@ q.q
			UNION ALL
			SELECT 'theme', t.source_id, t.id, t.label, NULL, NULL, ts_rank(t.search, q.q)
			FROM themes t, q WHERE t.search @@ q.q)
		SELECT kind, source_id, id, title, ref, url, round(rank::numeric, 4)::float8 AS r
		FROM h
		ORDER BY r DESC, kind, source_id, `+fmt.Sprintf(naturalID, "id")+`
		LIMIT $2`, q, SearchLimit)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Hit, error) {
		var h Hit
		err := r.Scan(&h.Kind, &h.Source, &h.ID, &h.Title, &h.Ref, &h.URL, &h.Rank)
		return h, err
	})
	if out == nil {
		out = []Hit{}
	}
	return out, err
}

// Verses counts the verses from the ordinal lo to the ordinal hi, both
// inclusive, by the book table (so Rom.5.12-Rom.6.4 is 14: Romans 5 ends
// at verse 21). An ordinal outside the table counts as its difference.
func Verses(lo, hi int) int {
	books := passage.Books()
	b, c, v := lo/1000000, lo/1000%1000, lo%1000
	eb, ec, ev := hi/1000000, hi/1000%1000, hi%1000
	valid := func(b, c int) bool { return b >= 1 && b <= len(books) && c >= 1 && c <= len(books[b-1].Last) }
	if hi < lo || !valid(b, c) || !valid(eb, ec) {
		return int(math.Abs(float64(hi-lo))) + 1
	}
	n := 0
	for b != eb || c != ec {
		n += books[b-1].Last[c-1] - v + 1
		c, v = c+1, 1
		if c > len(books[b-1].Last) {
			b, c = b+1, 1
		}
	}
	return n + ev - v + 1
}

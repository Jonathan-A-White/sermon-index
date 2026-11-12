package store

// The API's reads (story A2). Each function is one question /v1 asks, and
// its row type is the JSON the API answers (docs/api.md): the json tags
// here are the contract Lampas codes against, so a change to one is a
// change to the API.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

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
// such link. See is never nil. Sermons is how many sermons are on the whole
// topic by their preaching text (any part, any sermon source) and Verses how
// many verses the topic's links cover together; a topic over ThemeMaxVerses
// is not used by the theme tier, whatever its sermon count.
type ThemeRow struct {
	Source    string   `json:"source"`
	Topic     string   `json:"topic"`
	Label     string   `json:"label"`
	Part      *int     `json:"part"`
	PartLabel *string  `json:"part_label"`
	Ref       string   `json:"ref"`
	See       []string `json:"see"`
	Sermons   int      `json:"sermons"`
	Verses    int      `json:"verses"`
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
			ORDER BY l.source_id, l.theme_id, l.part_n, upper(l.span) - lower(l.span), l.ref_osis),
		n AS (
			SELECT k.source_id, k.theme_id,
			       (SELECT count(DISTINCT (p.source_id, p.sermon_id))
			        FROM theme_links l JOIN sermon_passages p ON p.role = 'text' AND p.span && l.span
			        WHERE l.source_id = k.source_id AND l.theme_id = k.theme_id) AS sermons
			FROM (SELECT DISTINCT source_id, theme_id FROM b) k)
		SELECT b.source_id, b.theme_id, t.label, b.part_n, p.label, b.ref_osis, t.see, n.sermons, coalesce(t.verses, 0)
		FROM b
		JOIN themes t ON t.source_id = b.source_id AND t.id = b.theme_id
		JOIN n ON n.source_id = b.source_id AND n.theme_id = b.theme_id
		LEFT JOIN theme_parts p ON p.theme_source_id = b.source_id AND p.theme_id = b.theme_id AND p.n = b.part_n
		ORDER BY b.source_id, b.w, t.label, b.theme_id, b.part_n NULLS FIRST`, span.PG())
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (ThemeRow, error) {
		var t ThemeRow
		var part *int32
		err := r.Scan(&t.Source, &t.Topic, &t.Label, &part, &t.PartLabel, &t.Ref, &t.See, &t.Sermons, &t.Verses)
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

// ThemeMaxVerses is the widest topic the theme tier uses: a topic whose
// links cover more verses than this is left out of a passage's own themes
// (a topic named by ThemeQuery.Themes is used whatever its width). The probe that set it
// (Lampas docs/sources/sermon-index.md, section 4) found Nave's whole-subject
// topics (JESUS, THE CHRIST 4,821 verses, GOD 4,051, SIN 916) put 2,735
// sermons on Romans 5 and the topics of 300 verses or fewer still 799; the
// topics of Romans 5:8 under 300 put 170, a set a reader can look through.
const ThemeMaxVerses = 300

// ThemeQuery is the theme tier's question: the sermons linked directly to a
// topic that links Span (sermon-topics.jsonl), then the sermons whose
// preaching text shares a topic with Span. Sources limits the sermon sources
// when not empty and Themes the topics, each "<theme source>:<topic>", when
// not empty; DirectOnly leaves out the sermons reached through their text.
// PassageTier is whether the passage tier is answered beside it (tier=theme):
// then a named topic over the whole Bible does not keep the passage rows
// either, so the two tiers are different sermons.
type ThemeQuery struct {
	Span          passage.Range
	Sources       []string
	Themes        []string
	DirectOnly    bool
	PassageTier   bool
	Limit, Offset int
}

// ThemeRef is the topic (and the numbered part of it, when the link names
// one) a theme-tier row came in by.
type ThemeRef struct {
	Source    string  `json:"source"`
	Topic     string  `json:"topic"`
	Label     string  `json:"label"`
	Part      *int    `json:"part"`
	PartLabel *string `json:"part_label"`
}

// ThemeSermonRow is a row of the theme tier: a sermon row and the topic.
// A row reached through its text has Ref its preaching text that met the
// topic and Role "text". A Direct row (a sermon linked to the topic in
// sermon-topics.jsonl) has Ref the sermon's preaching text, or Role "" when
// it has none: its ref, role and width are then null in the JSON. A direct
// row's JSON has "direct": true; a row reached through its text has no
// "direct" key, so its JSON is the same as before direct links.
type ThemeSermonRow struct {
	SermonRow
	Direct bool
	Theme  ThemeRef
}

// MarshalJSON writes the row as docs/api.md says: the sermon row's fields,
// ref, role and width null on a direct row with no preaching text, then
// "direct" on a direct row only, then "theme".
func (r ThemeSermonRow) MarshalJSON() ([]byte, error) {
	out := struct {
		SermonRow
		Ref    *string  `json:"ref"`
		Role   *string  `json:"role"`
		Width  *int     `json:"width"`
		Direct bool     `json:"direct,omitempty"`
		Theme  ThemeRef `json:"theme"`
	}{SermonRow: r.SermonRow, Direct: r.Direct, Theme: r.Theme}
	if r.Role != "" {
		out.Ref, out.Role, out.Width = &r.Ref, &r.Role, &r.Width
	}
	// '<', '>' and '&' as themselves, as the API's encoder writes the rest
	// (json.Marshal would escape them, and the encoder cannot undo it).
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// themeTier is each sermon's one theme-tier row; it takes $1 the span, $2
// whether to filter by sermon source, $3 the sources, $4 whether to filter by
// topic, $5 the topics ("source:topic"), $6 the cap, $7 whether to answer
// direct rows only and $8 whether to keep the sermons with a passage on the
// span. A sermon with a passage overlapping the span is a passage row and is
// never in it, unless $8 (a named topic asked over the whole Bible with the
// theme tier alone, where every sermon is a passage row and nothing else
// would be left).
//
// The cap $6 applies to a topic the span's own themes offer; a topic named
// ($4) is used however wide it is: a person chose it.
//
// d, the direct rows: a sermon linked in sermon_topics to a topic that links
// the span, whatever the topic's size (a person chose the link). Each sermon
// keeps its narrowest such topic (a topic with no verse count last), then the
// theme source and topic; its ref is its preaching text, or null.
//
// x, the rows reached through their text: a preaching text that overlaps a
// link of a topic that also links the span and is at most $6 verses, of a
// sermon not in d. Each sermon keeps its narrowest topic (then the theme
// source and topic), and of that topic the narrowest link its text meets
// (then the part).
const themeTier = `
	WITH d AS (
		SELECT DISTINCT ON (st.source_id, st.sermon_id)
		       st.source_id, st.sermon_id, t.source_id AS theme_source, t.id AS theme_id, t.label, t.verses
		FROM sermon_topics st
		JOIN themes t ON t.source_id = st.theme_source_id AND t.id = st.theme_id
		WHERE (NOT $2::bool OR st.source_id = ANY($3::text[]))
		  AND (NOT $4::bool OR t.source_id || ':' || t.id = ANY($5::text[]))
		  AND EXISTS (SELECT 1 FROM theme_links l WHERE l.source_id = t.source_id AND l.theme_id = t.id AND l.span && $1::int4range)
		  AND ($8::bool OR NOT EXISTS (SELECT 1 FROM sermon_passages q
		                  WHERE q.source_id = st.source_id AND q.sermon_id = st.sermon_id AND q.span && $1::int4range))
		ORDER BY st.source_id, st.sermon_id, t.verses NULLS LAST, t.source_id, t.id),
	topics AS (
		SELECT t.source_id, t.id, t.label, t.verses
		FROM themes t
		WHERE NOT $7::bool AND ($4::bool OR t.verses <= $6)
		  AND (NOT $4::bool OR t.source_id || ':' || t.id = ANY($5::text[]))
		  AND EXISTS (SELECT 1 FROM theme_links l WHERE l.source_id = t.source_id AND l.theme_id = t.id AND l.span && $1::int4range)),
	x AS (
		SELECT DISTINCT ON (p.source_id, p.sermon_id)
		       p.source_id, p.sermon_id, p.ref_osis, lower(p.span) AS lo, upper(p.span) AS hi,
		       t.source_id AS theme_source, t.id AS theme_id, t.label, t.verses, l.part_n, pt.label AS part_label
		FROM sermon_passages p
		JOIN theme_links l ON l.span && p.span
		JOIN topics t ON t.source_id = l.source_id AND t.id = l.theme_id
		LEFT JOIN theme_parts pt ON pt.theme_source_id = l.source_id AND pt.theme_id = l.theme_id AND pt.n = l.part_n
		WHERE p.role = 'text' AND (NOT $2::bool OR p.source_id = ANY($3::text[]))
		  AND ($8::bool OR NOT EXISTS (SELECT 1 FROM sermon_passages q
		                  WHERE q.source_id = p.source_id AND q.sermon_id = p.sermon_id AND q.span && $1::int4range))
		  AND NOT EXISTS (SELECT 1 FROM d WHERE d.source_id = p.source_id AND d.sermon_id = p.sermon_id)
		ORDER BY p.source_id, p.sermon_id, t.verses, t.source_id, t.id, upper(l.span) - lower(l.span), l.part_n NULLS FIRST, p.ref_osis)
	SELECT d.source_id, d.sermon_id, true AS direct, tp.ref_osis, lower(tp.span) AS lo, upper(tp.span) AS hi,
	       d.theme_source, d.theme_id, d.label, d.verses, NULL::integer AS part_n, NULL::text AS part_label
	FROM d
	LEFT JOIN LATERAL (SELECT p.ref_osis, p.span FROM sermon_passages p
	                   WHERE p.source_id = d.source_id AND p.sermon_id = d.sermon_id AND p.role = 'text'
	                   ORDER BY lower(p.span), p.ref_osis LIMIT 1) tp ON true
	UNION ALL
	SELECT x.source_id, x.sermon_id, false, x.ref_osis, x.lo, x.hi,
	       x.theme_source, x.theme_id, x.label, x.verses, x.part_n, x.part_label
	FROM x`

// wholeBible is whether span is Genesis 1:1 to Revelation 22:21, an ask in
// which every sermon with a passage is a passage row.
func wholeBible(span passage.Range) bool {
	return span.Lo == passage.Ordinal(1, 1, 1) && span.Hi == passage.Ordinal(len(passage.Books()), 22, 21)
}

// ThemeSermons answers the theme tier for q: how many sermons, and the page
// of them Limit and Offset name. Every direct row comes first, ordered by its
// topic's size ascending (a topic with no count last), then the theme source,
// the topic, the sermon's source and its id (a number as a number); then the
// rows reached through their text, ordered by the topic's size ascending,
// then the sermon's source and id. Rows is never nil.
func ThemeSermons(ctx context.Context, db *pgxpool.Pool, q ThemeQuery) (int, []ThemeSermonRow, error) {
	srcs, themes := q.Sources, q.Themes
	if srcs == nil {
		srcs = []string{}
	}
	if themes == nil {
		themes = []string{}
	}
	args := []any{q.Span.PG(), len(q.Sources) > 0, srcs, len(q.Themes) > 0, themes, ThemeMaxVerses, q.DirectOnly, len(q.Themes) > 0 && !q.PassageTier && wholeBible(q.Span)}
	var total int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM (`+themeTier+`) b`, args...).Scan(&total); err != nil {
		return 0, nil, err
	}
	rows, err := db.Query(ctx, `
		WITH b AS (`+themeTier+`)
		SELECT s.source_id, s.id, s.title, s.preacher, s.date, s.url, s.alt_url, s.kind, b.direct, b.ref_osis, b.lo, b.hi,
		       b.theme_source, b.theme_id, b.label, b.part_n, b.part_label
		FROM b JOIN sermons s ON s.source_id = b.source_id AND s.id = b.sermon_id
		ORDER BY b.direct DESC, b.verses NULLS LAST,
		         CASE WHEN b.direct THEN b.theme_source END, CASE WHEN b.direct THEN b.theme_id END,
		         s.source_id, `+fmt.Sprintf(naturalID, "s.id")+`
		LIMIT $9 OFFSET $10`, append(args, q.Limit, q.Offset)...)
	if err != nil {
		return 0, nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (ThemeSermonRow, error) {
		var s ThemeSermonRow
		var ref *string
		var lo, hi *int
		var part *int32
		err := r.Scan(&s.Source, &s.ID, &s.Title, &s.Preacher, &s.Date, &s.URL, &s.AltURL, &s.Kind, &s.Direct, &ref, &lo, &hi,
			&s.Theme.Source, &s.Theme.Topic, &s.Theme.Label, &part, &s.Theme.PartLabel)
		if ref != nil && lo != nil && hi != nil {
			s.Ref, s.Role, s.Width = *ref, "text", Verses(*lo, *hi-1)
		}
		if part != nil {
			n := int(*part)
			s.Theme.Part = &n
		}
		return s, err
	})
	if out == nil {
		out = []ThemeSermonRow{}
	}
	return total, out, err
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
// inclusive (passage.Verses).
func Verses(lo, hi int) int { return passage.Verses(lo, hi) }

// Package themes holds a theme source's records (a topic, its parts, a link
// from a topic to a passage) and writes them to the database.
//
// The files are read and checked by package sources; this package only
// knows the records and their tables (themes, theme_parts, theme_links).
package themes

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
)

// Via values: where a link came from.
const (
	ViaSource = "source" // the theme source itself (the default)
	ViaBible  = "bible"  // a theme noticed in reading the Bible
	ViaSermon = "sermon" // a theme a sermon carries; From names the sermon
)

// Part is a numbered part of a topic, "(4) Of God".
type Part struct {
	N     int    `json:"n"`
	Label string `json:"label"`
}

// Topic is one line of topics.jsonl.
type Topic struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	Parts []Part   `json:"parts,omitempty"`
	See   []string `json:"see,omitempty"`

	Line int `json:"-"` // the line of topics.jsonl it came from
}

// Link is one line of links.jsonl.
type Link struct {
	Topic string  `json:"topic"`
	Part  *int    `json:"part,omitempty"`
	Ref   string  `json:"ref"`
	Via   *string `json:"via,omitempty"`
	From  *string `json:"from,omitempty"`

	Line int           `json:"-"` // the line of links.jsonl it came from
	Span passage.Range `json:"-"` // Ref parsed, set by the check
}

// ViaOrDefault is the link's via, ViaSource when it is not given.
func (l Link) ViaOrDefault() string {
	if l.Via == nil {
		return ViaSource
	}
	return *l.Via
}

// Counts is what a write left in the database for the source.
type Counts struct {
	Topics, Parts, Links int
	Removed              int // topics and links in the database no longer in the files
}

// Write makes the source's topics, parts and links in the database exactly
// those given, inside tx: it upserts each by its key and deletes those of
// the source no longer given. The links' spans must already be parsed.
func Write(ctx context.Context, tx pgx.Tx, sourceID string, topics []Topic, links []Link) (Counts, error) {
	var c Counts
	topicIDs := make([]string, 0, len(topics))
	var partTopic []string
	var partN []int32

	b := &pgx.Batch{}
	for _, t := range topics {
		see := t.See
		if see == nil {
			see = []string{}
		}
		topicIDs = append(topicIDs, t.ID)
		b.Queue(`INSERT INTO themes (source_id, id, label, see) VALUES ($1, $2, $3, $4)
			ON CONFLICT (source_id, id) DO UPDATE SET label = EXCLUDED.label, see = EXCLUDED.see
			WHERE (themes.label, themes.see) IS DISTINCT FROM (EXCLUDED.label, EXCLUDED.see)`,
			sourceID, t.ID, t.Label, see)
		for _, p := range t.Parts {
			partTopic = append(partTopic, t.ID)
			partN = append(partN, int32(p.N))
			b.Queue(`INSERT INTO theme_parts (theme_source_id, theme_id, n, label) VALUES ($1, $2, $3, $4)
				ON CONFLICT (theme_source_id, theme_id, n) DO UPDATE SET label = EXCLUDED.label
				WHERE theme_parts.label IS DISTINCT FROM EXCLUDED.label`,
				sourceID, t.ID, p.N, p.Label)
		}
	}
	if err := tx.SendBatch(ctx, b).Close(); err != nil {
		return c, fmt.Errorf("writing topics: %w", err)
	}

	// The links' keys, as parallel arrays for unnest.
	var kTopic, kRef, kVia []string
	var kPart []*int32
	var kFrom []*string
	for _, l := range links {
		kTopic = append(kTopic, l.Topic)
		kRef = append(kRef, l.Span.OSIS)
		kVia = append(kVia, l.ViaOrDefault())
		kFrom = append(kFrom, l.From)
		if l.Part == nil {
			kPart = append(kPart, nil)
		} else {
			n := int32(*l.Part)
			kPart = append(kPart, &n)
		}
	}
	tag, err := tx.Exec(ctx, `DELETE FROM theme_links l WHERE l.source_id = $1 AND NOT EXISTS (
			SELECT 1 FROM unnest($2::text[], $3::int[], $4::text[], $5::text[], $6::text[]) AS k(topic, part, ref, via, src)
			WHERE k.topic = l.theme_id AND k.part IS NOT DISTINCT FROM l.part_n AND k.ref = l.ref_osis
			  AND k.via = l.via AND k.src IS NOT DISTINCT FROM l.from_ref)`,
		sourceID, kTopic, kPart, kRef, kVia, kFrom)
	if err != nil {
		return c, fmt.Errorf("removing links: %w", err)
	}
	c.Removed += int(tag.RowsAffected())

	if _, err := tx.Exec(ctx, `DELETE FROM theme_parts p WHERE p.theme_source_id = $1 AND NOT EXISTS (
			SELECT 1 FROM unnest($2::text[], $3::int[]) AS k(topic, n) WHERE k.topic = p.theme_id AND k.n = p.n)`,
		sourceID, partTopic, partN); err != nil {
		return c, fmt.Errorf("removing parts: %w", err)
	}
	tag, err = tx.Exec(ctx, `DELETE FROM themes WHERE source_id = $1 AND id <> ALL($2::text[])`, sourceID, topicIDs)
	if err != nil {
		return c, fmt.Errorf("removing topics: %w", err)
	}
	c.Removed += int(tag.RowsAffected())

	b = &pgx.Batch{}
	for _, l := range links {
		b.Queue(`INSERT INTO theme_links (source_id, theme_id, part_n, ref_osis, span, via, from_ref)
			VALUES ($1, $2, $3, $4, $5::int4range, $6, $7)
			ON CONFLICT ON CONSTRAINT theme_links_key DO UPDATE SET span = EXCLUDED.span
			WHERE theme_links.span IS DISTINCT FROM EXCLUDED.span`,
			sourceID, l.Topic, l.Part, l.Span.OSIS, l.Span.PG(), l.ViaOrDefault(), l.From)
	}
	if err := tx.SendBatch(ctx, b).Close(); err != nil {
		return c, fmt.Errorf("writing links: %w", err)
	}

	c.Topics, c.Parts, c.Links = len(topics), len(partN), len(links)
	return c, nil
}

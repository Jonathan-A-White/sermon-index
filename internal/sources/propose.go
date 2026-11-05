package sources

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/Jonathan-A-White/sermon-index/prompts"
)

// Story is what prompts/add-source.md is filled with.
type Story struct {
	ID, Name, Credit string
	// The fields a Mayor chooses, taken from the source here.
	Preacher     string // the preacher the rows name most, else the credit
	CatalogueURL string // the source's homepage
	TermsURL     string
	Size         string // S, M or L by the rows
	// What the source has.
	Rows, WithText, Series int
	RowsText, WithTextText string // "20 rows", "19 with a preaching text"
	Licence, LicenceNote   string
	LicenceQuoted          string // the licence, a quoted block
}

// Size limits: a source of up to SizeS rows is a small story, up to SizeM a
// medium one, and more is large.
const (
	SizeS = 300
	SizeM = 2000
)

func sizeOf(rows int) string {
	switch {
	case rows <= SizeS:
		return "S"
	case rows <= SizeM:
		return "M"
	}
	return "L"
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

// Propose checks the sermon source in dir and prints the story for adding
// it, prompts/add-source.md filled from its source.toml and its rows, so a
// Builder that has written a source can hand a Mayor the next step. A
// directory with a fault, or a theme source, is refused.
func Propose(dir string) (string, error) {
	d, faults := checked(dir)
	if len(faults) > 0 {
		return "", Faults(faults)
	}
	if d.Source.Kind != KindSermons {
		return "", errors.New("propose is for a sermon source: " + d.Source.ID + " is a themes source")
	}
	s := d.Source
	st := Story{
		ID: s.ID, Name: s.Name, Credit: s.Credit, CatalogueURL: s.Homepage, TermsURL: s.TermsURL,
		Rows: len(d.Rows), Licence: strings.TrimSpace(s.Licence), LicenceNote: strings.TrimSpace(s.LicenceNote),
	}
	preachers := map[string]int{}
	for _, r := range d.Rows {
		if r.text != nil {
			st.WithText++
		}
		if r.IsSeries() {
			st.Series++
		}
		if r.Preacher != nil {
			preachers[*r.Preacher]++
		}
	}
	st.Preacher = s.Credit
	if len(preachers) > 0 {
		names := make([]string, 0, len(preachers))
		for n := range preachers {
			names = append(names, n)
		}
		sort.Slice(names, func(i, j int) bool {
			if preachers[names[i]] != preachers[names[j]] {
				return preachers[names[i]] > preachers[names[j]]
			}
			return names[i] < names[j]
		})
		st.Preacher = names[0]
	}
	st.Size = sizeOf(st.Rows)
	st.RowsText = plural(st.Rows, "row", "rows")
	st.WithTextText = fmt.Sprintf("%d with a preaching text, %s", st.WithText, plural(st.Series, "series row", "series rows"))
	st.LicenceQuoted = "> " + strings.ReplaceAll(st.Licence, "\n", "\n> ")

	tmpl, err := template.New("add-source").Option("missingkey=error").Parse(prompts.AddSource)
	if err != nil {
		return "", fmt.Errorf("prompts/add-source.md: %w", err)
	}
	var b bytes.Buffer
	if err := tmpl.Execute(&b, st); err != nil {
		return "", fmt.Errorf("prompts/add-source.md: %w", err)
	}
	return b.String(), nil
}

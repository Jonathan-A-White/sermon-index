package sources

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
)

// David Clarkson's Practical Works loads with the counts REPORT.md gives: the
// 34 sermons of the three volumes' contents pages, each with its preaching
// text, a leaf of the Internet Archive's scan as url and notes that say only
// where in the edition it is: no sentence of a sermon is kept.
func TestLoadClarksonWorks(t *testing.T) {
	const dir = "../../sources/clarkson-works"
	if fs := Check(dir); len(fs) != 0 {
		t.Fatalf("check: %v", Faults(fs))
	}
	ctx := context.Background()
	db := migrated(t)
	if _, err := Load(ctx, db, dir); err != nil {
		t.Fatal(err)
	}
	const src = "select count(*) from sermons where source_id = 'clarkson-works'"
	if n := count(t, db, src); n != 34 {
		t.Errorf("%d sermons, want 34", n)
	}
	if n := count(t, db, "select count(*) from sermon_passages where role = 'text' and sermon_id in (select id from sermons where source_id = 'clarkson-works')"); n != 34 {
		t.Errorf("%d preaching texts, want 34", n)
	}
	for vol, want := range map[string]int{"01": 13, "02": 13, "03": 8} {
		if n := count(t, db, src+" and url ~ ('^https://archive\\.org/details/practicalworksof' || $1 || 'clar/page/n[0-9]+$') and preacher = 'David Clarkson' and date is null and alt_url is null", vol); n != want {
			t.Errorf("volume %s: %d sermons with an Archive leaf, want %d", vol, n, want)
		}
	}
	// Two sermons on Hebrews 11:13 ("Of Dying in Faith" and "Of Living as
	// Strangers") answer a reader on the verse, none a reader on 2 Peter 1:4.
	at := func(ref string) int {
		rg, err := passage.Parse(ref)
		if err != nil {
			t.Fatal(err)
		}
		return count(t, db, "select count(*) from sermon_passages p join sermons s on s.id = p.sermon_id where s.source_id = 'clarkson-works' and p.role = 'text' and p.span && $1::int4range", rg.PG())
	}
	if n := at("Heb.11.13"); n != 2 {
		t.Errorf("Hebrews 11:13: %d sermons, want 2", n)
	}
	report, err := os.ReadFile(dir + "/REPORT.md")
	if err != nil {
		t.Fatal(err)
	}
	// The Governor's question: a row on 2 Peter 1:4, or the report says none.
	if at("2Pet.1.4") == 0 && !strings.Contains(string(report), "None: no sermon's text is 2 Peter 1:4") {
		t.Error("no row is on 2 Peter 1:4 and REPORT.md does not say none")
	}
}

// Facts only: a row of clarkson-works holds a title of the contents pages, a
// reference, the leaf's address and where in the edition it is, in words the
// tool writes; nothing of the scan's text.
func TestClarksonRowsHoldFactsOnly(t *testing.T) {
	d, faults := Read("../../sources/clarkson-works")
	if len(faults) != 0 {
		t.Fatalf("read: %v", faults)
	}
	notes := regexp.MustCompile(`^Volume [123], page [0-9]+( \(the contents page prints [0-9]+\))?\.( It begins with no heading of its own\.)?$`)
	for _, r := range d.Rows {
		if r.Notes == nil || !notes.MatchString(*r.Notes) {
			t.Errorf("row %s: notes %v are not the tool's own words", r.ID, r.Notes)
		}
		if len(r.Title) > 90 || strings.ContainsAny(r.Title, "\n") {
			t.Errorf("row %s: title %q is longer than a title", r.ID, r.Title)
		}
		if len(r.Topics) != 0 || r.ExpectTitle != nil || r.Cited != nil {
			t.Errorf("row %s: fields the tool does not write", r.ID)
		}
	}
}

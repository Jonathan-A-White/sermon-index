package sources

import (
	"context"
	"strings"
	"testing"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
)

// Life Bible Ministry's Through the Bible loads with the counts REPORT.md
// gives: 995 rows, 993 with a passage (2 whole-book series have none), and
// every row has its page as url, a YouTube watch address as alt_url (so
// Lampas's videoOf() plays it) and Pastor Paul LeBoutillier as preacher.
// Isaiah 1-4:1 (isaiah/1) answers a reader on Isaiah 2.
func TestLoadLifeBibleTTB(t *testing.T) {
	const dir = "../../sources/lifebible-ttb"
	if fs := Check(dir); len(fs) != 0 {
		t.Fatalf("check: %v", Faults(fs))
	}
	ctx := context.Background()
	db := migrated(t)
	if _, err := Load(ctx, db, dir); err != nil {
		t.Fatal(err)
	}
	const src = "select count(*) from sermons where source_id = 'lifebible-ttb'"
	if n := count(t, db, src); n != 995 {
		t.Errorf("%d rows, want 995", n)
	}
	if n := count(t, db, "select count(*) from sermon_passages where role = 'text' and sermon_id in (select id from sermons where source_id = 'lifebible-ttb')"); n != 993 {
		t.Errorf("%d preaching texts, want 993", n)
	}
	if n := count(t, db, src+" and (preacher is distinct from 'Pastor Paul LeBoutillier' or url !~ '^https://www.lifebibleministry.com/through-the-bible/[a-z0-9-]+/.+$' or alt_url !~ '^https://www.youtube.com/watch\\?v=[A-Za-z0-9_-]{11}$' or date is not null)"); n != 0 {
		t.Errorf("%d rows with another preacher, url or alt_url than expected, or a date", n)
	}
	var title, url, alt string
	if err := db.QueryRow(ctx, "select title, url, alt_url from sermons where source_id = 'lifebible-ttb' and url = 'https://www.lifebibleministry.com/through-the-bible/isaiah/1'").Scan(&title, &url, &alt); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(title, "Isaiah 1-4:1") || alt != "https://www.youtube.com/watch?v=ynUYAm1-W6k" {
		t.Errorf("isaiah/1: %q %q", title, alt)
	}
	isa2, err := passage.Parse("Isa.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, "select count(*) from sermon_passages p join sermons s on s.id = p.sermon_id where s.source_id = 'lifebible-ttb' and p.role = 'text' and s.url like '%/isaiah/1' and p.span && $1::int4range", isa2.PG()); n != 1 {
		t.Errorf("Isaiah 2:3 finds isaiah/1 %d times, want 1", n)
	}
}

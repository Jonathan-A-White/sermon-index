package main

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestBodyTextIsTheLessonOverviewOnly(t *testing.T) {
	page, err := os.ReadFile("testdata/post-body-passage.html")
	if err != nil {
		t.Fatal(err)
	}
	got := BodyText(page)
	if !strings.Contains(got, "Isaiah 53:1-12") || !strings.Contains(got, "Romans 4:25") {
		t.Errorf("body text %q lacks the overview's passages", got)
	}
	// the share buttons and the previous/next videos are not the post's text
	for _, no := range []string{"Genesis 1:1", "Psalm 23", "Romans 8", "<p>"} {
		if strings.Contains(got, no) {
			t.Errorf("body text has %q: %q", no, got)
		}
	}
}

func TestRefsInText(t *testing.T) {
	for _, c := range []struct {
		in   string
		bare bool
		want []string
	}{
		{"A walk through Isaiah 53:1-12 and Romans 4:25.", false, []string{"Isa.53.1-Isa.53.12", "Rom.4.25"}},
		{"see John 3:16, then 1 John 4:8 and 2 Cor. 5:17-21", false, []string{"John.3.16", "1John.4.8", "2Cor.5.17-2Cor.5.21"}},
		{"Psalm 23 and Genesis 1:26-2:3", true, []string{"Ps.23", "Gen.1.26-Gen.2.3"}},
		{"Psalm 23 is loved", false, nil},                    // a bare chapter only where the caller allows it
		{"Mark 5 minutes later, Job 2 years on", false, nil}, // prose, not passages
		{"mark 3:4 and gen 1:1", false, nil},                 // a book is written with a capital
		{"Romans 99:1 and Jude 1:99", true, nil},             // a chapter or verse the book does not have
		{"Mark 5 minutes later", true, nil},                  // not a chapter followed by a word of time
		{"Rom 8:1 and Matt 5:3-12", false, []string{"Rom.8.1", "Matt.5.3-Matt.5.12"}},
	} {
		if got := RefsInText(c.in, c.bare); !reflect.DeepEqual(got, c.want) {
			t.Errorf("RefsInText(%q, %v) = %v, want %v", c.in, c.bare, got, c.want)
		}
	}
}

func TestRefsFromSlugAnyBook(t *testing.T) {
	for slug, want := range map[string][]string{
		"the-suffering-servant-isaiah-531-12": {"Isa.53.1-Isa.53.12"},
		"love-1-john-48":                      {"1John.4.8"},
		"jesus-in-the-law-no-passage-here":    nil,
		"what-is-john-about":                  nil,
		"the-good-shepherd-psalm-23":          {"Ps.2.3", "Ps.23"},
	} {
		if got := RefsFromSlugAnyBook(slug); !reflect.DeepEqual(got, want) {
			t.Errorf("RefsFromSlugAnyBook(%q) = %v, want %v", slug, got, want)
		}
	}
}

func TestCategoriesAreTheEightTheGovernorNamed(t *testing.T) {
	var got []string
	for _, c := range Categories {
		got = append(got, c.Slug)
	}
	want := []string{"1-peter", "hebrews", "mark", "romans",
		"how-to-find-jesus-in-the-old-testament", "wisdom-in-the-word", "understanding-the-old-testament-law", "past-livestreams-and-misc-topical-teachings"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("categories = %v, want %v", got, want)
	}
	for _, c := range Categories[:4] {
		if c.Book == "" {
			t.Errorf("%s: a book series with no book", c.Slug)
		}
	}
	for _, c := range Categories[4:] {
		if c.Book != "" {
			t.Errorf("%s: a topical category with book %s", c.Slug, c.Book)
		}
	}
	for _, c := range Categories {
		for _, no := range NotReadCategories {
			if c.Slug == no {
				t.Errorf("%s is read and in the not-read list", c.Slug)
			}
		}
	}
	for _, no := range []string{"past-qa-sessions", "debates", "interviews", "thinking-biblically-reviews", "worship-songs", "christians-react-to-20-atheist-arguments-against-god"} {
		if !contains(NotReadCategories, no) {
			t.Errorf("%s is not in the not-read list", no)
		}
	}
}

var wisdom = Series{Slug: "wisdom-in-the-word", Name: "Wisdom in the Word"}

func bodies(m map[string]string) func(Episode) string {
	return func(e Episode) string { return m[e.Slug] }
}

func TestTopicalPostWithAPassageInItsBodyIsARow(t *testing.T) {
	page, _ := os.ReadFile("testdata/post-body-passage.html")
	eps := []Episode{{
		PostID: 5, Title: "How the Servant Suffers", URL: "https://biblethinker.org/how-the-servant-suffers/", Slug: "how-the-servant-suffers",
		YouTubeID: "s0eX1P2f8Nc", DurationS: 3000, Categories: []string{"wisdom-in-the-word"},
	}}
	res := BuildSeries(wisdom, eps, map[int]string{5: "2019-01-06"}, bodies(map[string]string{"how-the-servant-suffers": BodyText(page)}))
	if len(res.Rows) != 1 {
		t.Fatalf("rows = %+v skips = %+v", res.Rows, res.Skips)
	}
	var b bytes.Buffer
	WriteRows(&b, res.Rows)
	want := `{"id":"how-the-servant-suffers","title":"How the Servant Suffers","ref":"Isa.53.1-Isa.53.12","url":"https://biblethinker.org/how-the-servant-suffers/","alt_url":"https://www.youtube.com/watch?v=s0eX1P2f8Nc","kind":"link","preacher":"Mike Winger","date":"2019-01-06","duration_s":3000,"notes":"Series: Wisdom in the Word (https://biblethinker.org/wisdom-in-the-word/). The YouTube id is the one embedded in the post. The title and the post's address print no passage; it is the first one the post's own text names."}` + "\n"
	if b.String() != want {
		t.Errorf("row:\n got %s\nwant %s", b.String(), want)
	}
}

func TestTopicalPostWithNoPassageAnywhereIsSkippedAndNamed(t *testing.T) {
	page, _ := os.ReadFile("testdata/post-no-passage.html")
	eps := []Episode{{
		PostID: 6, Title: "Slavery and the Bible", URL: "https://biblethinker.org/slavery-and-the-bible/", Slug: "slavery-and-the-bible",
		YouTubeID: "s0eX1P2f8Nc", Categories: []string{"wisdom-in-the-word"},
	}}
	res := BuildSeries(wisdom, eps, nil, bodies(map[string]string{"slavery-and-the-bible": BodyText(page)}))
	if len(res.Rows) != 0 || len(res.Skips) != 1 {
		t.Fatalf("rows = %+v skips = %+v", res.Rows, res.Skips)
	}
	if r := res.Skips[0].Reason; reasonKind(r) != "no passage" || !strings.Contains(r, "body") {
		t.Errorf("reason %q", r)
	}
}

func TestTopicalPostWhoseBodyNamesAChapterAloneIsARow(t *testing.T) {
	eps := []Episode{{
		PostID: 10, Title: "Melchizedek and Jesus", URL: "https://biblethinker.org/melchizedek/", Slug: "melchizedek",
		YouTubeID: "s0eX1P2f8Nc", Categories: []string{"wisdom-in-the-word"},
	}}
	res := BuildSeries(wisdom, eps, nil, bodies(map[string]string{"melchizedek": "He shows up in Genesis 14, and in Psalm 110 we hear of a priest."}))
	if len(res.Rows) != 1 || *res.Rows[0].Ref != "Gen.14" {
		t.Fatalf("rows = %+v skips = %+v", res.Rows, res.Skips)
	}
}

func TestTopicalTitleBeatsTheBody(t *testing.T) {
	eps := []Episode{{
		PostID: 7, Title: "The Good Shepherd: Psalm 23", URL: "https://biblethinker.org/the-good-shepherd/", Slug: "the-good-shepherd",
		YouTubeID: "s0eX1P2f8Nc", Categories: []string{"wisdom-in-the-word"},
	}}
	res := BuildSeries(wisdom, eps, nil, bodies(map[string]string{"the-good-shepherd": "See Isaiah 40:11."}))
	if len(res.Rows) != 1 || *res.Rows[0].Ref != "Ps.23" {
		t.Fatalf("rows = %+v skips = %+v", res.Rows, res.Skips)
	}
}

func TestBookSeriesEpisodeSkippedByTitleIsRetriedWithTheBody(t *testing.T) {
	eps := []Episode{
		{PostID: 8, Title: "The Key to All the Parables: The Mark Series part 13", URL: "https://biblethinker.org/the-key-to-all-the-parables/", Slug: "the-key-to-all-the-parables",
			YouTubeID: "s0eX1P2f8Nc", Categories: []string{"mark"}},
		{PostID: 9, Title: "Who Wrote It? The Mark Series pt 1", URL: "https://biblethinker.org/who-wrote-it/", Slug: "who-wrote-it",
			YouTubeID: "s0eX1P2f8Nc", Categories: []string{"mark"}},
	}
	got := bodies(map[string]string{
		// a passage in another book is not the passage of a Mark episode; the first one in Mark is
		"the-key-to-all-the-parables": "Isaiah 6:9-10 is quoted in Mark 4:1-20, the parable of the sower.",
		"who-wrote-it":                "Papias and the early church. See Isaiah 40.",
	})
	res := BuildSeries(Series{Slug: "mark", Book: "Mark", Name: "Mark"}, eps, nil, got)
	if len(res.Rows) != 1 || *res.Rows[0].Ref != "Mark.4.1-Mark.4.20" {
		t.Fatalf("rows = %+v skips = %+v", res.Rows, res.Skips)
	}
	if len(res.Skips) != 1 || res.Skips[0].Slug != "who-wrote-it" || reasonKind(res.Skips[0].Reason) != "no passage" {
		t.Errorf("skips = %+v", res.Skips)
	}
}

func TestBodyIsOnlyFetchedWhenTitleAndAddressNameNoPassage(t *testing.T) {
	eps := []Episode{{
		PostID: 7, Title: "The Good Shepherd: Psalm 23", URL: "https://biblethinker.org/the-good-shepherd/", Slug: "the-good-shepherd",
		YouTubeID: "s0eX1P2f8Nc", Categories: []string{"wisdom-in-the-word"},
	}}
	BuildSeries(wisdom, eps, nil, func(e Episode) string { t.Errorf("body asked for %s", e.Slug); return "" })
}

func TestTopicalPostWithNoVideoIsSkipped(t *testing.T) {
	eps := []Episode{{PostID: 7, Title: "Psalm 23", URL: "https://biblethinker.org/p/", Slug: "p", Categories: []string{"wisdom-in-the-word"}}}
	res := BuildSeries(wisdom, eps, nil, nil)
	if len(res.Rows) != 0 || len(res.Skips) != 1 || reasonKind(res.Skips[0].Reason) != "no YouTube id" {
		t.Errorf("rows = %+v skips = %+v", res.Rows, res.Skips)
	}
}

func TestADuplicatePostIsOneRow(t *testing.T) {
	a := Episode{PostID: 1, URL: "https://biblethinker.org/a/", Slug: "a"}
	b := Episode{PostID: 2, URL: "https://biblethinker.org/b/", Slug: "b"}
	seen := map[string]bool{}
	fresh, dups := SplitSeen([]Episode{a, b}, seen)
	if len(fresh) != 2 || len(dups) != 0 {
		t.Fatalf("first category: fresh %d dups %d", len(fresh), len(dups))
	}
	fresh, dups = SplitSeen([]Episode{b, {PostID: 3, URL: "https://biblethinker.org/c/", Slug: "c"}}, seen)
	if len(fresh) != 1 || fresh[0].Slug != "c" || len(dups) != 1 || dups[0].Slug != "b" {
		t.Errorf("second category: fresh %+v dups %+v", fresh, dups)
	}
}

func TestReportCountsEachCategory(t *testing.T) {
	runs := []SeriesRun{
		{Data: &SeriesData{Pages: []string{"p"}, Count: 3}, Res: SeriesResult{
			Series: wisdom, Seen: 3, Dups: 1,
			Rows:  []Row{{ID: "a"}},
			Skips: []Skip{{Slug: "n", Title: "No Passage Here", URL: "https://biblethinker.org/n/", Reason: "no passage in the title, the post's address or the post's text (body)"}},
		}},
	}
	var b bytes.Buffer
	WriteReport(&b, runs, "User-agent: *\nDisallow:\n")
	out := b.String()
	for _, want := range []string{
		"| Wisdom in the Word | https://biblethinker.org/wisdom-in-the-word/ | 1 | 3 | 3 | 1 | 1 | 1 |",
		"No Passage Here (https://biblethinker.org/n/): no passage in the title, the post's address or the post's text (body)",
		"past-qa-sessions",
		"Also in an earlier category",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report lacks %q:\n%s", want, out)
		}
	}
}

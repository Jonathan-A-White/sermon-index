package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func fixtureEpisodes(t *testing.T) []Episode {
	t.Helper()
	var all []Episode
	for _, f := range []string{"testdata/archive-1-peter-1.html", "testdata/archive-1-peter-2.html"} {
		page, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		eps, _, err := ParseArchive(page)
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, eps...)
	}
	return all
}

var fixtureDates = map[int]string{
	8337: "2015-09-06", 8336: "2015-09-13", 8326: "2015-12-06",
	8300: "2016-03-27", 8325: "2015-12-13", 8299: "2016-03-03", 8298: "2016-03-04",
}

func TestBuildSeriesWritesTheRow(t *testing.T) {
	res := BuildSeries(Series{Slug: "1-peter", Book: "1Pet", Name: "1 Peter"}, fixtureEpisodes(t), fixtureDates)

	var got *Row
	for i := range res.Rows {
		if res.Rows[i].ID == "hope-in-grief-born-again-jesus-resurrection-1-peter-13-7" {
			got = &res.Rows[i]
		}
	}
	if got == nil {
		t.Fatalf("no row for the 1 Peter 1:3-7 episode; rows: %+v", res.Rows)
	}
	var b bytes.Buffer
	if err := WriteRows(&b, []Row{*got}); err != nil {
		t.Fatal(err)
	}
	want := `{"id":"hope-in-grief-born-again-jesus-resurrection-1-peter-13-7","title":"Hope in Grief, Born Again, Jesus’ Resurrection: 1 Peter 1:3-7","ref":"1Pet.1.3-1Pet.1.7","url":"https://biblethinker.org/hope-in-grief-born-again-jesus-resurrection-1-peter-13-7/","alt_url":"https://www.youtube.com/watch?v=n-0M1f1iJ5Y","kind":"link","preacher":"Mike Winger","date":"2015-09-13","duration_s":4149,"notes":"Series: 1 Peter (https://biblethinker.org/1-peter/). The YouTube id is the one embedded in the post."}` + "\n"
	if b.String() != want {
		t.Errorf("row:\n got %s\nwant %s", b.String(), want)
	}
}

func TestBuildSeriesSkipsAndCounts(t *testing.T) {
	res := BuildSeries(Series{Slug: "1-peter", Book: "1Pet", Name: "1 Peter"}, fixtureEpisodes(t), fixtureDates)
	if res.Seen != 7 {
		t.Errorf("seen = %d, want 7", res.Seen)
	}
	reasons := map[string]string{}
	for _, s := range res.Skips {
		reasons[s.Slug] = s.Reason
	}
	// no passage in the title (the slug has none either)
	for _, slug := range []string{"modern-apostles-calvinism-foreknowledge-the-trinity-and-an-overview-of-1-peter", "how-to-be-a-husband-the-lost-art-of-biblical-manliness"} {
		if !strings.Contains(reasons[slug], "no passage") {
			t.Errorf("%s: reason %q, want no passage", slug, reasons[slug])
		}
	}
	// a passage but no YouTube embed
	if r := reasons["evaluating-church-leaders-1-of-2-1-peter-51-2"]; !strings.Contains(r, "no YouTube") {
		t.Errorf("reason %q, want no YouTube id", r)
	}
	// not in the series' own category: not a row of it
	if r := reasons["q-and-a-1-peter-21"]; !strings.Contains(r, "not in the 1-peter series") {
		t.Errorf("reason %q, want not in the series", r)
	}
	if len(res.Rows)+len(res.Skips) != res.Seen {
		t.Errorf("rows %d + skips %d != seen %d", len(res.Rows), len(res.Skips), res.Seen)
	}
	if len(res.Rows) != 3 {
		t.Errorf("rows = %d, want 3", len(res.Rows))
	}
	for _, r := range res.Rows {
		if strings.Contains(r.URL, "q-and-a") {
			t.Errorf("a row from a post outside the series: %s", r.URL)
		}
	}
}

func TestBuildSeriesRowsAreInDateOrder(t *testing.T) {
	res := BuildSeries(Series{Slug: "1-peter", Book: "1Pet", Name: "1 Peter"}, fixtureEpisodes(t), fixtureDates)
	for i := 1; i < len(res.Rows); i++ {
		if *res.Rows[i-1].Date > *res.Rows[i].Date {
			t.Errorf("row %d (%s) is before row %d (%s)", i, *res.Rows[i].Date, i-1, *res.Rows[i-1].Date)
		}
	}
}

func TestBuildSeriesUsesSlugWhenTheTitleHasNoPassage(t *testing.T) {
	eps := []Episode{{
		PostID: 1, Title: "Worship in the Bible and Lessons in Theology",
		URL: "https://biblethinker.org/to-god-be-the-glory-romans-1133-36/", Slug: "to-god-be-the-glory-romans-1133-36",
		YouTubeID: "mzyAOaFpnuQ", Categories: []string{"romans"},
	}}
	res := BuildSeries(Series{Slug: "romans", Book: "Rom", Name: "Romans"}, eps, nil)
	if len(res.Rows) != 1 || *res.Rows[0].Ref != "Rom.11.33-Rom.11.36" {
		t.Fatalf("rows = %+v skips = %+v", res.Rows, res.Skips)
	}
	if res.Rows[0].Date != nil {
		t.Errorf("a post with no date has date %q", *res.Rows[0].Date)
	}
	if res.NoDate != 1 {
		t.Errorf("NoDate = %d, want 1", res.NoDate)
	}
}

func TestBuildSeriesNamesATitleAndSlugThatDisagree(t *testing.T) {
	eps := []Episode{{
		PostID: 1, Title: "Spiritual Warfare – 1 Peter 2:11-12",
		URL: "https://biblethinker.org/spiritual-warfare-1-peter-31-2/", Slug: "spiritual-warfare-1-peter-31-2",
		YouTubeID: "N-ypbo5DS0U", Categories: []string{"1-peter"},
	}}
	res := BuildSeries(Series{Slug: "1-peter", Book: "1Pet", Name: "1 Peter"}, eps, nil)
	if len(res.Rows) != 1 || *res.Rows[0].Ref != "1Pet.2.11-1Pet.2.12" {
		t.Fatalf("the title wins: rows = %+v", res.Rows)
	}
	if len(res.Disagree) != 1 {
		t.Errorf("disagree = %v, want one", res.Disagree)
	}
}

func TestParseDuration(t *testing.T) {
	for in, want := range map[string]int{"1:09:09": 4149, "7:21": 441, "58:20": 3500, "": 0, "soon": 0} {
		if got := parseDuration(in); got != want {
			t.Errorf("%q: got %d, want %d", in, got, want)
		}
	}
}

func TestWriteRowsIsOneJSONObjectALine(t *testing.T) {
	res := BuildSeries(Series{Slug: "1-peter", Book: "1Pet", Name: "1 Peter"}, fixtureEpisodes(t), fixtureDates)
	var b bytes.Buffer
	if err := WriteRows(&b, res.Rows); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n")
	if len(lines) != len(res.Rows) {
		t.Fatalf("%d lines for %d rows", len(lines), len(res.Rows))
	}
	for _, l := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Errorf("%q: %v", l, err)
		}
	}
}

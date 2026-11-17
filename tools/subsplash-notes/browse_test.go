package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// Subsplash's sermon list holds only the newest 200 items, so a walk of its
// pages never reaches an older sermon. The church site lists every series
// (its /media/ index, 15 to a page) and each series page lists its sermons,
// twelve a View More: browse reads those too. These pages are shaped like the
// ones the church site serves, with invented titles.

const siteBase = "https://brookfieldfirst.org/media/"

func seriesAddress(code, name string) string {
	return siteBase + "series/" + code + "/" + slugOf(name)
}

// indexPage is the series index's page n (1 is /media/): the series on it, the
// other links such a page has, and the count of all the series.
func indexPage(n, total int, series ...[2]string) Page {
	url := seriesIndexURL
	if n > 1 {
		url += "page/" + itoa(n) + "/"
	}
	p := Page{URL: url, Status: 200, Total: total, Text: "Sermons\nSeries\n", Links: []Link{{Text: "Scripture", Href: siteBase + "scripture"}, {Text: "2", Href: siteBase + "page/2/"}}}
	for _, s := range series {
		p.Links = append(p.Links, Link{Text: s[0], Href: seriesAddress(s[1], s[0])})
		p.Text += s[0] + "\n"
	}
	return p
}

// sermonSeriesPage is the page of a series showing the first n of sermons
// (newest first: title, date line, code), with View More while there are more.
func sermonSeriesPage(name, code string, sermons [][3]string, n int) Page {
	p := Page{URL: seriesAddress(code, name), Status: 200, Total: len(sermons), More: n < len(sermons), Text: name + "\n"}
	for _, e := range sermons[:n] {
		p.Links = append(p.Links, Link{Text: e[0] + "\n" + e[1], Href: siteBase + e[2] + "/" + slugOf(e[0])})
		p.Text += e[0] + "\n" + e[1] + "\n"
	}
	return p
}

func TestParseSeriesIndex(t *testing.T) {
	got, err := ParseSeriesIndex(indexPage(1, 26, [2]string{"Acts", "aaaaaaa"}, [2]string{"Romans", "bbbbbbb"}, [2]string{"Morning RePhil", "4fmk6cd"}))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{seriesAddress("aaaaaaa", "Acts"), seriesAddress("bbbbbbb", "Romans"), seriesAddress("4fmk6cd", "Morning RePhil")}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("got %v, want %v", got, want)
	}
	if _, err := ParseSeriesIndex(indexPage(1, 0)); err == nil {
		t.Error("an index page with no series was taken for one")
	}
}

// olderWorld is a church whose Subsplash list shows only its newest sermon (the
// cap), while the church site's three series, over two index pages, hold the
// newest again and three older ones; one older sermon needs View More.
func olderWorld(t *testing.T) (*fakeBrowser, Config) {
	t.Helper()
	f, cfg := world(t)
	cfg.SeriesIndex = seriesIndexURL
	d := "2024-03-03"
	cfg.Rows = append(cfg.Rows, sources.Row{ID: "ROW4", Title: "The Oldest Sermon", Date: &d})
	f.list = []Page{listPage(1, [2]string{"The Invented Sermon", "3nxf2bb"})}
	// 3 series, 2 to a page.
	f.pages[seriesIndexURL] = indexPage(1, 3, [2]string{"Acts", "aaaaaaa"}, [2]string{"Romans", "bbbbbbb"})
	f.pages[seriesIndexURL+"page/2/"] = indexPage(2, 3, [2]string{"Stand Alone", "ccccccc"}, [2]string{"Morning RePhil", "4fmk6cd"})
	acts := [][3]string{{"The Invented Sermon", "Nov 8, 2026", "3nxf2bb"}, {"The Older Sermon", "Oct 25, 2026", "ooooooo"}}
	f.pages[seriesAddress("aaaaaaa", "Acts")] = sermonSeriesPage("Acts", "aaaaaaa", acts, 2)
	romans := [][3]string{{"The Oldest Sermon", "Mar 3, 2024", "yyyyyyy"}, {"The Very Oldest Sermon", "Feb 25, 2024", "zzzzzzz"}}
	f.moreBy[seriesAddress("bbbbbbb", "Romans")] = []Page{sermonSeriesPage("Romans", "bbbbbbb", romans, 1), sermonSeriesPage("Romans", "bbbbbbb", romans, 2)}
	f.pages[seriesAddress("ccccccc", "Stand Alone")] = sermonSeriesPage("Stand Alone", "ccccccc", [][3]string{{"A Sermon Without Notes", "Nov 1, 2026", "nnnnnnn"}}, 1)
	// The Morning RePhil series is read by its own pass (cfg.Series), not this.
	f.pages[seriesAddress("4fmk6cd", "Morning RePhil")] = Page{URL: seriesAddress("4fmk6cd", "Morning RePhil"), Status: 200, Text: "must not be loaded"}
	f.pages[sermonURL("3nxf2bb", "the-invented-sermon")] = sermonPage("3nxf2bb", "The Invented Sermon", "Nov 8, 2026 • Pastor Example • Acts 1:8", true)
	f.notes[sermonURL("3nxf2bb", "the-invented-sermon")] = notesPage("The Invented Sermon", "Acts 1:8", "November 8, 2026")
	f.pages[sermonURL("yyyyyyy", "the-oldest-sermon")] = sermonPage("yyyyyyy", "The Oldest Sermon", "Mar 3, 2024 • Pastor Example • Romans 8:1-4", true)
	f.notes[sermonURL("yyyyyyy", "the-oldest-sermon")] = notesPage("The Oldest Sermon", "Romans 8:1-4", "March 3, 2024")
	f.pages[sermonURL("zzzzzzz", "the-very-oldest-sermon")] = sermonPage("zzzzzzz", "The Very Oldest Sermon", "Feb 25, 2024 • Pastor Example • Romans 5:1", false)
	return f, cfg
}

func TestBrowseReadsSermonsPastTheListsCap(t *testing.T) {
	f, cfg := olderWorld(t)
	var out bytes.Buffer
	if _, err := Browse(context.Background(), f, cfg, &out); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	links := read(t, cfg.LinksOut)
	for _, w := range []string{
		sermonURL("3nxf2bb", "the-invented-sermon"), // in the list
		sermonURL("ooooooo", "the-older-sermon"),    // in a series only
		sermonURL("yyyyyyy", "the-oldest-sermon"),   // on a series' first View More
		sermonURL("zzzzzzz", "the-very-oldest-sermon"),
		sermonURL("nnnnnnn", "a-sermon-without-notes"), // on the second index page
	} {
		if !strings.Contains(links, w+"\n") {
			t.Errorf("no link kept for %s:\n%s", w, links)
		}
	}
	// A sermon of the list that a series shows again is read once.
	count := map[string]int{}
	for _, l := range f.loads {
		count[l]++
	}
	if n := count[sermonURL("3nxf2bb", "the-invented-sermon")]; n != 1 {
		t.Errorf("the sermon in the list and in a series was loaded %d times", n)
	}
	// The older sermon with notes is recorded; so is the one on View More.
	records := read(t, cfg.Out)
	for _, title := range []string{"The Older Sermon", "The Oldest Sermon"} {
		if !strings.Contains(records, `"title":"`+title+`"`) {
			t.Errorf("no record of %q:\n%s", title, records)
		}
	}
	for _, l := range f.loads {
		if strings.Contains(l, "4fmk6cd") {
			t.Errorf("the Morning RePhil series was loaded by the sweep: %s", l)
		}
	}
	if f.moreAt[seriesAddress("bbbbbbb", "Romans")] != 1 {
		t.Errorf("View More pressed %d times on Romans", f.moreAt[seriesAddress("bbbbbbb", "Romans")])
	}
	for i := 1; i < len(f.times); i++ {
		if gap := f.times[i].Sub(f.times[i-1]); gap < 10*time.Second {
			t.Errorf("load %d (%s) came %v after the one before", i, f.loads[i], gap)
		}
	}
	// A re-run reads no sermon again: only the index and the series pages.
	f.loads, f.times = nil, nil
	if _, err := Browse(context.Background(), f, cfg, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, l := range f.loads {
		if strings.HasPrefix(l, base) || strings.HasPrefix(l, "notes of ") {
			t.Errorf("a re-run loaded %s", l)
		}
	}
}

// With no series index asked for, the list alone is walked, as before.
func TestBrowseWithNoSeriesIndexWalksOnlyTheList(t *testing.T) {
	f, cfg := olderWorld(t)
	cfg.SeriesIndex = ""
	if _, err := Browse(context.Background(), f, cfg, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, l := range f.loads {
		if strings.HasPrefix(l, siteBase) {
			t.Errorf("loaded %s", l)
		}
	}
}

func TestBrowseSeriesIndexStopsOnA429(t *testing.T) {
	f, cfg := olderWorld(t)
	blocked := f.pages[seriesIndexURL+"page/2/"]
	blocked.Status = 429
	f.pages[seriesIndexURL+"page/2/"] = blocked
	_, err := Browse(context.Background(), f, cfg, &bytes.Buffer{})
	var st Stop
	if !errors.As(err, &st) || !strings.Contains(st.Why, "429") {
		t.Fatalf("got %v", err)
	}
	// What was read before the 429 is kept; nothing after it.
	if links := read(t, cfg.LinksOut); strings.Contains(links, "nnnnnnn") {
		t.Errorf("a sermon of the blocked page was read:\n%s", links)
	}
	if _, err := os.Stat(cfg.LinksOut); err != nil {
		t.Errorf("the links before the stop were not kept: %v", err)
	}
}

// An index page that gives no count of its series would end the sweep at the
// first page without a word, so it stops the run instead.
func TestBrowseSeriesIndexWithNoCountStops(t *testing.T) {
	f, cfg := olderWorld(t)
	p := f.pages[seriesIndexURL]
	p.Total = 0
	f.pages[seriesIndexURL] = p
	_, err := Browse(context.Background(), f, cfg, &bytes.Buffer{})
	var st Stop
	if !errors.As(err, &st) || !strings.Contains(st.Why, "no count") {
		t.Fatalf("got %v", err)
	}
}

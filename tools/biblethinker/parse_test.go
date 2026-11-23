package main

import (
	"os"
	"reflect"
	"testing"
)

func TestRefFromTitle(t *testing.T) {
	cases := []struct {
		book, title, want string
		cited             []string
	}{
		{"1Pet", "Hope in Grief, Born Again, Jesus’ Resurrection: 1 Peter 1:3-7", "1Pet.1.3-1Pet.1.7", nil},
		{"1Pet", "Evaluating Church Leaders (1 of 2) 1 Peter 5:1-2", "1Pet.5.1-1Pet.5.2", nil},
		{"Mark", "John the Baptizer: The Mark Series Part 2 (1:1-8)", "Mark.1.1-Mark.1.8", nil},
		{"Mark", "Insights from Jesus and the Demoniac: The Mark Series Part 16 (Mk 5:1-20)", "Mark.5.1-Mark.5.20", nil},
		{"Mark", "Rethinking The Women at The Empty Tomb: The Mark Series pt 66 (15:40-16:8)", "Mark.15.40-Mark.16.8", nil},
		{"Mark", "How An Atheist Scholar Misleads Millions Of People: The Mark Series pt 65 (15:34)", "Mark.15.34", nil},
		{"Mark", "“The Poor You Will Always Have With You”: The Mark Series pt 58 (14:1-11 again)", "Mark.14.1-Mark.14.11", nil},
		{"Mark", "Do Babies Go to Heaven? A Biblical Examination (The Mark Series pt 35, 10:13-16)", "Mark.10.13-Mark.10.16", nil},
		{"Mark", "Does Mark 11:26 Belong in Your Bible: The Mark Series pt 46", "Mark.11.26", nil},
		{"Mark", "We Desperately Need to Hear What Jesus Said in Mark 10: The Mark Series pt 34 (10:1-12)", "Mark.10.1-Mark.10.12", nil},
		{"Rom", "Theology of SIN: Romans 6", "Rom.6", nil},
		{"Rom", "Non-Calvinist Interpretation of Romans 9", "Rom.9", nil},
		{"Rom", "Homosexuality and Romans 1: Controversy and Clarity", "Rom.1", nil},
		{"Rom", "Calvinism, Arminianism, Election & Predestination: Romans 8:29-30, 33", "Rom.8.29-Rom.8.30", []string{"Rom.8.33"}},
		{"Rom", "Witnessing to Jews: Romans 3:28-4:13", "Rom.3.28-Rom.4.13", nil},
		{"Heb", "Jesus is YHWH: The Hebrews Series part 5 (Heb 1:6-2:1)", "Heb.1.6-Heb.2.1", nil},
		// no passage in the title
		{"Mark", "Testing Your Christian Worldview: The Mark Series pt 9", "", nil},
		{"Mark", "Who Wrote Mark’s Gospel? Part 1 of the new Mark series", "", nil},
		{"Rom", "Gospels Compared: JW, Mormon, Catholic or Romans", "", nil},
		{"1Pet", "SLAVERY and the BIBLE!? Explained!", "", nil},
	}
	for _, c := range cases {
		got, cited, err := RefFromTitle(c.book, c.title)
		if err != nil {
			t.Errorf("%q: %v", c.title, err)
			continue
		}
		if got != c.want || !reflect.DeepEqual(cited, c.cited) {
			t.Errorf("%q: got %q %v, want %q %v", c.title, got, cited, c.want, c.cited)
		}
	}
}

func TestRefFromTitleOutOfRange(t *testing.T) {
	for _, title := range []string{"Nowhere (20:1-2)", "Verse too high (1:99)", "Backwards (3:10-4)"} {
		if ref, _, err := RefFromTitle("Mark", title); err == nil {
			t.Errorf("%q: got %q, want an error", title, ref)
		}
	}
}

func TestRefsFromSlug(t *testing.T) {
	cases := []struct {
		book, slug string
		want       []string
	}{
		{"1Pet", "hope-in-grief-born-again-jesus-resurrection-1-peter-13-7", []string{"1Pet.1.3-1Pet.1.7"}},
		{"1Pet", "judgment-day-for-believers-and-holy-living-1-peter-114-25", []string{"1Pet.1.14-1Pet.1.25"}},
		{"1Pet", "evaluating-church-leaders-2-of-2-1-peter-51-4", []string{"1Pet.5.1-1Pet.5.4"}},
		{"Rom", "to-god-be-the-glory-romans-1133-36", []string{"Rom.11.33-Rom.11.36"}},
		{"Rom", "witnessing-to-jews-romans-328-413", []string{"Rom.3.28-Rom.4.13"}},
		{"Rom", "theology-of-sin-romans-6", []string{"Rom.6"}},
		{"Heb", "jesus-is-yhwh-the-hebrews-series-part-5-heb-16-21", []string{"Heb.1.6-Heb.2.1"}},
		{"Heb", "did-god-speak-the-hebrews-series-pt-2-11-2", []string{"Heb.1.1-Heb.1.2"}},
		{"Mark", "john-the-baptizer-the-mark-series-part-2-11-8", []string{"Mark.1.1-Mark.1.8"}},
		{"Mark", "the-poor-you-will-always-have-with-you-the-mark-series-pt-58-141-11-again", []string{"Mark.14.1-Mark.14.11"}},
		{"Mark", "how-an-atheist-the-mark-series-pt-65-1534", []string{"Mark.15.34"}},
		// a part number alone is no passage
		{"Mark", "testing-your-christian-worldview-the-mark-series-pt-9", nil},
		{"Mark", "what-exactly-is-the-unforgivable-sin-the-mark-series-pt-12", nil},
		{"Rom", "the-end-of-romans", nil},
		// ambiguous: 12:1 or 1:21
		{"Rom", "god-wants-a-living-sacrifice-romans-121", []string{"Rom.1.21", "Rom.12.1"}},
	}
	for _, c := range cases {
		got := RefsFromSlug(c.book, c.slug)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q: got %v, want %v", c.slug, got, c.want)
		}
	}
}

func TestParseArchive(t *testing.T) {
	page, err := os.ReadFile("testdata/archive-1-peter-1.html")
	if err != nil {
		t.Fatal(err)
	}
	eps, next, err := ParseArchive(page)
	if err != nil {
		t.Fatal(err)
	}
	if next != "https://biblethinker.org/1-peter/page/2/" {
		t.Errorf("next = %q", next)
	}
	if len(eps) != 3 {
		t.Fatalf("got %d episodes, want 3", len(eps))
	}
	want := Episode{
		PostID:     8336,
		Title:      "Hope in Grief, Born Again, Jesus’ Resurrection: 1 Peter 1:3-7",
		URL:        "https://biblethinker.org/hope-in-grief-born-again-jesus-resurrection-1-peter-13-7/",
		Slug:       "hope-in-grief-born-again-jesus-resurrection-1-peter-13-7",
		YouTubeID:  "n-0M1f1iJ5Y",
		DurationS:  1*3600 + 9*60 + 9,
		Categories: []string{"1-peter"},
	}
	if !reflect.DeepEqual(eps[1], want) {
		t.Errorf("got %+v, want %+v", eps[1], want)
	}
	if got := eps[2].Categories; !reflect.DeepEqual(got, []string{"1-peter", "marriage-divorce-remarriage"}) {
		t.Errorf("categories = %v", got)
	}
	if eps[2].DurationS != 58*60+20 {
		t.Errorf("duration = %d", eps[2].DurationS)
	}

	last, err := os.ReadFile("testdata/archive-1-peter-2.html")
	if err != nil {
		t.Fatal(err)
	}
	eps, next, err = ParseArchive(last)
	if err != nil || next != "" {
		t.Fatalf("last page: next %q, err %v", next, err)
	}
	if eps[2].YouTubeID != "" {
		t.Errorf("an article with no embed has id %q", eps[2].YouTubeID)
	}
	if eps[3].Title != "Q&A on 1 Peter 2:1" {
		t.Errorf("title = %q", eps[3].Title)
	}
}

func TestTitleWhitespaceIsOneLine(t *testing.T) {
	page := []byte(`<article class="post-1 post category-mark entry"><header><h2 class="entry-title"><a class="entry-title-link" rel="bookmark" href="https://biblethinker.org/x-the-mark-series-pt-28-813-26/">When Jesus Warns You, It&#8217;s Best
To Pay Attention: The Mark Series
Pt 28 (8:13-26)</a></h2></header></article>`)
	eps, _, err := ParseArchive(page)
	if err != nil || len(eps) != 1 {
		t.Fatalf("%v %v", eps, err)
	}
	if want := "When Jesus Warns You, It’s Best To Pay Attention: The Mark Series Pt 28 (8:13-26)"; eps[0].Title != want {
		t.Errorf("title = %q", eps[0].Title)
	}
}

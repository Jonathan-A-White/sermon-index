package main

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestParseSitemap(t *testing.T) {
	f, err := os.Open("testdata/sitemap.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := ParseSitemap(f)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"https://www.lifebibleministry.com/through-the-bible",
		"https://www.lifebibleministry.com/through-the-bible/isaiah",
		"https://www.lifebibleministry.com/through-the-bible/isaiah/1",
		"https://www.lifebibleministry.com/through-the-bible/nehemiah/3-4",
		"https://www.lifebibleministry.com/through-the-bible/matthew/in-depth/7-1",
		"https://www.lifebibleministry.com/through-the-bible/romans/ch-5-verse-1-verse-11",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got\n%v\nwant\n%v", got, want)
	}
}

func TestPageSlug(t *testing.T) {
	for u, want := range map[string]string{
		"https://www.lifebibleministry.com/through-the-bible":                      "",
		"https://www.lifebibleministry.com/through-the-bible/isaiah":               "isaiah",
		"https://www.lifebibleministry.com/through-the-bible/isaiah/1":             "isaiah/1",
		"https://www.lifebibleministry.com/through-the-bible/matthew/in-depth/7-1": "matthew/in-depth/7-1",
		"https://www.lifebibleministry.com/through-the-bible/isaiah/1/":            "isaiah/1",
	} {
		if got := slugOf(u); got != want {
			t.Errorf("slugOf(%q) = %q, want %q", u, got, want)
		}
	}
}

func TestDisallowed(t *testing.T) {
	for _, p := range []string{"/api/x", "/auth/login", "/console/", "/locked/a", "/background", "/background/x"} {
		if !Disallowed(p) {
			t.Errorf("%s should be disallowed by robots.txt", p)
		}
	}
	for _, p := range []string{"/through-the-bible/isaiah/1", "/sitemap.xml", "/teachings"} {
		if Disallowed(p) {
			t.Errorf("%s should be allowed", p)
		}
	}
}

func TestParsePage(t *testing.T) {
	for file, want := range map[string]Page{
		"testdata/isaiah-1.html":    {Title: "Isaiah 1-4:1: God confronts the Kingdom of Judah", YouTube: "ynUYAm1-W6k"},
		"testdata/no-video.html":    {Title: "Romans 5 (Part 1) :1-11 • Justified by faith"},
		"testdata/iframe-only.html": {Title: "Matthew 7 (Part 1) :1-5 • Judging Others", YouTube: "5l_txKdvc3s"},
	} {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		got := ParsePage(string(b))
		if got != want {
			t.Errorf("%s: got %+v, want %+v", file, got, want)
		}
	}
}

func TestParseTitle(t *testing.T) {
	cases := []struct {
		title, ref, shown string
		series            bool
	}{
		{"Isaiah 1-4:1: God confronts the Kingdom of Judah", "Isa.1.1-Isa.4.1", "Isaiah 1-4:1: God confronts the Kingdom of Judah", false},
		{"Nehemiah 3–4: Obedience in the Face of Opposition", "Neh.3-Neh.4", "Nehemiah 3–4: Obedience in the Face of Opposition", false},
		{"Matthew 2 • The Magi and the Attempt on Jesus' Life by Herod: The Magi and the Attempt on Jesus' Life by Herod", "Matt.2", "Matthew 2 • The Magi and the Attempt on Jesus' Life by Herod", false},
		{"Matthew 7 (Part 1) :1-5 • Judging Others", "Matt.7.1-Matt.7.5", "Matthew 7 (Part 1) :1-5 • Judging Others", false},
		{"2 Corinthians 5 (Part 3) :18-21: Reconciled to God", "2Cor.5.18-2Cor.5.21", "2 Corinthians 5 (Part 3) :18-21: Reconciled to God", false},
		{"1 Corinthians 2: The natural and the spiritual man", "1Cor.2", "1 Corinthians 2: The natural and the spiritual man", false},
		{"Psalm 23: The Lord is my shepherd", "Ps.23", "Psalm 23: The Lord is my shepherd", false},
		{"John 3:16: God so loved", "John.3.16", "John 3:16: God so loved", false},
		{"Genesis 1:1-2:3: Creation", "Gen.1.1-Gen.2.3", "Genesis 1:1-2:3: Creation", false},
		{"Genesis 28–Genesis 30 (Part 1): Jacob gets a taste", "Gen.28-Gen.30", "Genesis 28–Genesis 30 (Part 1): Jacob gets a taste", false},
		{"Exodus 34 (Part 2)–Exodus 36: Covenant renewed", "Exod.34-Exod.36", "Exodus 34 (Part 2)–Exodus 36: Covenant renewed", false},
		{"Romans 15 (Part 2) and Romans 16 (Part 1): Final Words", "Rom.15-Rom.16", "Romans 15 (Part 2) and Romans 16 (Part 1): Final Words", false},
		{"Isaiah chapter 4:2 thru chapter 6: \"I saw the Lord\"", "Isa.4.2-Isa.6", "Isaiah chapter 4:2 thru chapter 6: \"I saw the Lord\"", false},
		{"Colossians 3 (Part 1) 1-17: Living in the Power", "Col.3.1-Col.3.17", "Colossians 3 (Part 1) 1-17: Living in the Power", false},
		{"Ezra 7–8 Ezra Sent To Teach The People", "Ezra.7-Ezra.8", "Ezra 7–8 Ezra Sent To Teach The People", false},
		{"Jeremiah 22–25", "Jer.22-Jer.25", "Jeremiah 22–25", false},
		{"Ezekiel 1–Ezekiel 3:7: Visions of Hope", "Ezek.1.1-Ezek.3.7", "Ezekiel 1–Ezekiel 3:7: Visions of Hope", false},
		{"Obadiah: God's Judgment on Edom", "Obad.1", "Obadiah: God's Judgment on Edom", false},
		{"Nahum: A prophesy against Assyria", "Nah", "Nahum: A prophesy against Assyria", false},
		{"Lamentations: Crying Out in Our Brokenness", "Lam.1", "Lamentations: Crying Out in Our Brokenness", true},
	}
	for _, c := range cases {
		ref, shown, series, ok := ParseTitle(c.title)
		if !ok || series != c.series || ref != c.ref || shown != c.shown {
			t.Errorf("ParseTitle(%q) = %q, %q, series %v, %v; want %q, %q, series %v", c.title, ref, shown, series, ok, c.ref, c.shown, c.series)
		}
	}
	for _, bad := range []string{"Genesis: Through the Bible", "A Word on Prayer", "", "Romans (Part 1) :1-17: Paul, a servant", "Notabook 3: x", "Isaiah 99 : x"} {
		if ref, _, _, ok := ParseTitle(bad); ok {
			t.Errorf("ParseTitle(%q) = %q, want no passage", bad, ref)
		}
	}
}

func TestStripSite(t *testing.T) {
	if got := siteTitle("Matthew 2 • The Magi &amp; Herod | Life Bible Ministry"); got != "Matthew 2 • The Magi & Herod" {
		t.Errorf("got %q", got)
	}
	if !strings.Contains(userAgent, "sermon-index") {
		t.Error("User-Agent does not name the project")
	}
}

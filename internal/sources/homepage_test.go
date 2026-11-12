package sources

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// A Logos source's credit link (its homepage) opens that archive in Logos, not
// logos.com: https://ref.ly/logosres/<resource>, the resource its rows' urls carry.
func TestLogosSourceHomepageIsItsResource(t *testing.T) {
	dirs, err := filepath.Glob("../../sources/*-logos")
	if err != nil || len(dirs) == 0 {
		t.Fatalf("no *-logos sources found (%v)", err)
	}
	homepage := regexp.MustCompile(`^https://ref\.ly/logosres/(LLS:[^/?#&]+)$`)
	rowRes := regexp.MustCompile(`https://ref\.ly/logosres/(LLS:[^/?#&"]+)`)
	for _, dir := range dirs {
		id := filepath.Base(dir)
		var s Source
		if _, err := toml.DecodeFile(filepath.Join(dir, "source.toml"), &s); err != nil {
			t.Errorf("%s: %v", id, err)
			continue
		}
		m := homepage.FindStringSubmatch(s.Homepage)
		if m == nil {
			t.Errorf("%s: homepage %q is not a https://ref.ly/logosres/LLS:<resource> link", id, s.Homepage)
			continue
		}
		rows, err := os.ReadFile(filepath.Join(dir, "rows.jsonl"))
		if err != nil {
			t.Errorf("%s: %v", id, err)
			continue
		}
		used := map[string]bool{}
		for _, r := range rowRes.FindAllStringSubmatch(string(rows), -1) {
			used[r[1]] = true
		}
		if !used[m[1]] {
			t.Errorf("%s: homepage resource %s is not one its rows use", id, m[1])
		}
	}
}

// Brookfield First's credit link is the church's Subsplash media page, and the
// source is not named for YouTube.
func TestBrookfieldFirstHomepageIsSubsplash(t *testing.T) {
	var s Source
	if _, err := toml.DecodeFile("../../sources/brookfield-first/source.toml", &s); err != nil {
		t.Fatal(err)
	}
	if s.Homepage != "https://subsplash.com/u/brookfieldfirst/media" {
		t.Errorf("homepage %q", s.Homepage)
	}
	if strings.Contains(s.Name, "YouTube") || !strings.Contains(s.Name, "Subsplash") {
		t.Errorf("name %q", s.Name)
	}
}

// No source's credit link is a bare site root (https://archive.org,
// https://www.youtube.com, https://logos.com): it must open the thing the
// source is made from. Henry's is the Archive's Miscellaneous Works, the book
// of 44 of its 51 rows.
func TestNoSourceHomepageIsABareSiteRoot(t *testing.T) {
	files, err := filepath.Glob("../../sources/*/source.toml")
	if err != nil || len(files) == 0 {
		t.Fatalf("no source.toml found (%v)", err)
	}
	for _, f := range files {
		id := filepath.Base(filepath.Dir(f))
		var s Source
		if _, err := toml.DecodeFile(f, &s); err != nil {
			t.Errorf("%s: %v", id, err)
			continue
		}
		u, err := url.Parse(s.Homepage)
		if err != nil || u.Host == "" {
			t.Errorf("%s: homepage %q is not a URL", id, s.Homepage)
			continue
		}
		if p := strings.Trim(u.Path, "/"); p == "" && u.RawQuery == "" && u.Fragment == "" {
			t.Errorf("%s: homepage %q is a bare site root", id, s.Homepage)
		}
	}
}

func TestHenryHomepageIsMiscellaneousWorks(t *testing.T) {
	var s Source
	if _, err := toml.DecodeFile("../../sources/henry-sermons/source.toml", &s); err != nil {
		t.Fatal(err)
	}
	if want := "https://archive.org/details/miscellaneouswor00henr"; s.Homepage != want {
		t.Errorf("homepage %q, want %q", s.Homepage, want)
	}
}

package sources

import (
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

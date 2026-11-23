package main

import (
	"os"
	"path/filepath"
	"testing"
)

// A refresh from YouTube keeps what tools/subsplash-notes browse added to
// data/morning-rephil.jsonl: the share link and notes file of an episode, and
// the episodes only Subsplash has.
func TestKeepRephilKeepsWhatBrowseAdded(t *testing.T) {
	fresh := []RephilRow{
		{Number: 2, Title: "Morning RePhil #2", Date: "2026-11-09", URL: "https://www.youtube.com/watch?v=b"},
		{Number: 3, Title: "Morning RePhil #3", Date: "2026-11-13", URL: "https://www.youtube.com/watch?v=c"},
	}
	kept := []RephilRow{
		{Number: 1, Title: "Morning RePhil #1", Date: "2026-11-02", Share: "https://brookfieldfirst.subspla.sh/aaaaaaa"},
		{Number: 2, Title: "Morning RePhil #2", Date: "2026-11-09", URL: "https://www.youtube.com/watch?v=b", Share: "https://brookfieldfirst.subspla.sh/bbbbbbb", NotesFile: "rephil-2.txt"},
	}
	got := KeepRephil(fresh, kept)
	want := []RephilRow{
		kept[0],
		kept[1],
		fresh[1],
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestReadRephilMissingFileHasNone(t *testing.T) {
	rows, err := ReadRephil(filepath.Join(t.TempDir(), "none.jsonl"))
	if err != nil || rows != nil {
		t.Errorf("got %v, %v", rows, err)
	}
	bad := filepath.Join(t.TempDir(), "bad.jsonl")
	os.WriteFile(bad, []byte("not json\n"), 0o644)
	if _, err := ReadRephil(bad); err == nil {
		t.Error("a bad line was read")
	}
}

// An episode Subsplash had before YouTube did (a line with a share and no url)
// is one row once YouTube lists it: the share and notes file move onto the
// YouTube row, matched on the number, and the url-less line is not kept twice.
func TestKeepRephilMergesAnEpisodeSubsplashHadFirst(t *testing.T) {
	fresh := []RephilRow{
		{Number: 478, Title: "Morning RePhil #478", Date: "2026-11-10", URL: "https://www.youtube.com/watch?v=a"},
		{Number: 479, Title: "Morning RePhil #479", Date: "2026-11-13", URL: "https://www.youtube.com/watch?v=b"},
	}
	kept := []RephilRow{
		{Number: 478, Title: "Morning RePhil #478", Date: "2026-11-10", URL: "https://www.youtube.com/watch?v=a", Share: "https://brookfieldfirst.subspla.sh/aaaaaaa"},
		{Number: 479, Title: "Morning RePhil #479", Date: "2026-11-13", URL: "https://www.youtube.com/watch?v=b"},
		{Number: 479, Title: "Morning RePhil #479", Date: "2026-11-13", Share: "https://brookfieldfirst.subspla.sh/fnnh79c", NotesFile: "n.txt"},
		{Number: 480, Title: "Morning RePhil #480", Date: "2026-11-16", Share: "https://brookfieldfirst.subspla.sh/ccccccc"},
	}
	got := KeepRephil(fresh, kept)
	want := []RephilRow{
		kept[0],
		{Number: 479, Title: "Morning RePhil #479", Date: "2026-11-13", URL: "https://www.youtube.com/watch?v=b", Share: "https://brookfieldfirst.subspla.sh/fnnh79c", NotesFile: "n.txt"},
		kept[3],
	}
	if len(got) != len(want) {
		t.Fatalf("got %d rows: %+v", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: %+v, want %+v", i, got[i], want[i])
		}
	}
}

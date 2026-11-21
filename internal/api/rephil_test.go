package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const rephilFixture = "../../sources/_fixtures/rephil/morning-rephil.jsonl"

type gotRephil struct {
	Number int     `json:"number"`
	Title  string  `json:"title"`
	Date   string  `json:"date"`
	Share  *string `json:"share"`
	Video  *string `json:"video"`
}

func rephilServer(t *testing.T, file string) *testServer {
	t.Helper()
	return newTestServerWith(t, loadedDB(t), func(c *Config) { c.RephilFile = file })
}

func getRephil(t *testing.T, ts *testServer) []gotRephil {
	t.Helper()
	res := ts.do("GET", "/v1/rephil")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /v1/rephil: %d %s", res.StatusCode, body(t, res))
	}
	var rows []gotRephil
	if err := json.Unmarshal([]byte(body(t, res)), &rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

func str(s string) *string { return &s }

// Every episode, oldest first, one for each number: an episode the file
// holds on two lines (the channel's video, Subsplash's share link) is one
// row with both; the video is the id alone, and what is not known is null.
func TestRephilListsEveryEpisodeOldestFirst(t *testing.T) {
	rows := getRephil(t, rephilServer(t, rephilFixture))
	want := []gotRephil{
		{300, "Morning RePhil #300", "2024-01-07", str("https://brookfieldfirst.subspla.sh/hpp9pq4"), nil},
		{302, "Morning RePhil #302", "2024-01-14", str("https://brookfieldfirst.subspla.sh/br2bczk"), str("aGjZ0tgU47I")},
		{303, "Morning RePhil #303", "2024-01-21", str("https://brookfieldfirst.subspla.sh/thd5c6d"), str("LZEwbd652-Y")},
		{304, "Morning RePhil #304", "2024-01-28", nil, str("yGY16nh3_ow")},
	}
	if len(rows) != len(want) {
		t.Fatalf("%d rows, want %d: %+v", len(rows), len(want), rows)
	}
	for i, w := range want {
		g := rows[i]
		if g.Number != w.Number || g.Title != w.Title || g.Date != w.Date || !eq(g.Share, w.Share) || !eq(g.Video, w.Video) {
			t.Errorf("row %d: %+v, want %+v", i, g, w)
		}
	}
}

func eq(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// The 30-day rule: the id is served from the file as it is now, so a
// refresh that drops a video drops its id from the next answer.
func TestRephilIsReadFromTheFileEachTime(t *testing.T) {
	f := filepath.Join(t.TempDir(), "morning-rephil.jsonl")
	line := `{"number":1,"title":"Morning RePhil #1","date":"2026-01-01","url":"https://www.youtube.com/watch?v=abcDEF12345","share":"https://brookfieldfirst.subspla.sh/aaaaaaa"}` + "\n"
	if err := os.WriteFile(f, []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := rephilServer(t, f)
	if rows := getRephil(t, ts); len(rows) != 1 || rows[0].Video == nil || *rows[0].Video != "abcDEF12345" {
		t.Fatalf("first read: %+v", rows)
	}
	line = strings.Replace(line, `"url":"https://www.youtube.com/watch?v=abcDEF12345",`, "", 1)
	if err := os.WriteFile(f, []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	if rows := getRephil(t, ts); len(rows) != 1 || rows[0].Video != nil {
		t.Fatalf("after the refresh dropped the video: %+v", rows)
	}
}

func TestRephilWithNoFileIs503(t *testing.T) {
	for name, file := range map[string]string{"unset": "", "missing": filepath.Join(t.TempDir(), "none.jsonl")} {
		ts := rephilServer(t, file)
		res := ts.do("GET", "/v1/rephil")
		if res.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("%s: status %d", name, res.StatusCode)
		}
		if code, _ := errorBody(t, res); code != "unavailable" {
			t.Errorf("%s: error %q", name, code)
		}
	}
}

func TestRephilWithABadLineIs500(t *testing.T) {
	f := filepath.Join(t.TempDir(), "morning-rephil.jsonl")
	if err := os.WriteFile(f, []byte("not json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := rephilServer(t, f).do("GET", "/v1/rephil")
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("status %d", res.StatusCode)
	}
}

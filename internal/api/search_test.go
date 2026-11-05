package api

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func TestSearch(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	res := ts.do("GET", "/v1/search?q="+url.QueryEscape("spurgeon grace"))
	if res.StatusCode != 200 {
		t.Fatalf("status %d", res.StatusCode)
	}
	var s struct {
		Q    string `json:"q"`
		Rows []struct {
			Kind   string   `json:"kind"`
			Source string   `json:"source"`
			ID     string   `json:"id"`
			Title  string   `json:"title"`
			Ref    *string  `json:"ref"`
			URL    *string  `json:"url"`
			Rank   *float64 `json:"rank"`
		} `json:"rows"`
	}
	if err := json.NewDecoder(res.Body).Decode(&s); err != nil {
		t.Fatal(err)
	}
	if s.Q != "spurgeon grace" || len(s.Rows) == 0 {
		t.Fatalf("%+v", s)
	}
	var titles []string
	for _, r := range s.Rows {
		titles = append(titles, r.Title)
		if r.Kind != "sermon" || r.Source != "spurgeon-20" || r.URL == nil || r.Rank == nil || *r.Rank <= 0 {
			t.Errorf("row %+v", r)
		}
	}
	if !strings.Contains(strings.Join(titles, "|"), "Justification by Grace") {
		t.Errorf("spurgeon grace: %q", titles)
	}

	// Themes are found by their labels.
	res = ts.do("GET", "/v1/search?q=atonement")
	b := body(t, res)
	if !strings.Contains(b, `"kind": "theme"`) || !strings.Contains(b, `"title": "Atonement"`) {
		t.Errorf("atonement: %s", b)
	}

	// An empty q is 400; so is one past 200 characters.
	for _, q := range []string{"", "q=", "q=++", "q=" + strings.Repeat("a", 201)} {
		res := ts.do("GET", "/v1/search?"+q)
		if res.StatusCode != 400 {
			t.Errorf("%q: %d", q, res.StatusCode)
		}
		if code, _ := errorBody(t, res); code != "bad_query" {
			t.Errorf("%q: error %q", q, code)
		}
	}
	// A query that matches nothing is 200 and no rows, not null.
	if b := body(t, ts.do("GET", "/v1/search?q=zzyzx")); b != "{\n  \"q\": \"zzyzx\",\n  \"rows\": []\n}\n" {
		t.Errorf("zzyzx: %q", b)
	}
}

func TestThemesByRef(t *testing.T) {
	ts := newTestServer(t, loadedDB(t))
	res := ts.do("GET", "/v1/themes?ref=rom.5.8")
	var s struct {
		Ref   string `json:"ref"`
		Total int    `json:"total"`
		Rows  []struct {
			Source    string   `json:"source"`
			Topic     string   `json:"topic"`
			Label     string   `json:"label"`
			Part      *int     `json:"part"`
			PartLabel *string  `json:"part_label"`
			Ref       string   `json:"ref"`
			See       []string `json:"see"`
		} `json:"rows"`
	}
	if err := json.NewDecoder(res.Body).Decode(&s); err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 || s.Ref != "Rom.5.8" || s.Total != 2 || len(s.Rows) != 2 {
		t.Fatalf("%d %+v", res.StatusCode, s)
	}
	love, atone := s.Rows[0], s.Rows[1]
	if love.Topic != "love" || love.Part == nil || *love.Part != 1 || *love.PartLabel != "Of God" || love.Ref != "Rom.5.8" ||
		strings.Join(love.See, ",") != "grace" {
		t.Errorf("love: %+v", love)
	}
	if atone.Topic != "atonement" || atone.Part != nil || atone.PartLabel != nil || atone.See == nil {
		t.Errorf("atonement: %+v", atone)
	}
}

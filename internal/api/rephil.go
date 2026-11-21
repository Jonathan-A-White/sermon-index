package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
)

// rephilRow is an episode of Morning RePhil, as /v1/rephil lists it. Share
// is the Subsplash link Lampas opens; Video is the YouTube video id alone.
// What is not known is null.
type rephilRow struct {
	Number int     `json:"number"`
	Title  string  `json:"title"`
	Date   string  `json:"date"`
	Share  *string `json:"share"`
	Video  *string `json:"video"`
}

// rephilLine is a line of data/morning-rephil.jsonl; the file's other
// fields (notes_file) are not served.
type rephilLine struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Date   string `json:"date"`
	URL    string `json:"url"`
	Share  string `json:"share"`
}

// rephil is GET /v1/rephil: every Morning RePhil episode, oldest first.
//
// The episodes are not in the database. They are read from the file
// (Config.RephilFile) on every request, so the YouTube data in an answer is
// never older than the file: YouTube's terms let the rig keep it 30 days,
// and scripts/refresh-brookfield.sh refreshes the file, and so the next
// answer, inside that.
func (s *server) rephil(w http.ResponseWriter, r *http.Request) {
	if s.rephilFile == "" {
		fail(w, r, http.StatusServiceUnavailable, "unavailable", "the episodes' file is not set: SERMONS_REPHIL")
		return
	}
	f, err := os.Open(s.rephilFile)
	if err != nil {
		s.errLog.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		fail(w, r, http.StatusServiceUnavailable, "unavailable", "the episodes' file cannot be read; try again shortly")
		return
	}
	defer f.Close()
	rows, err := readRephil(f)
	if err != nil {
		s.errLog.Printf("%s %s: %s: %v", r.Method, r.URL.Path, s.rephilFile, err)
		fail(w, r, http.StatusInternalServerError, "internal", "the episodes' file is not well formed; it is logged")
		return
	}
	reply(w, r, rows)
}

// readRephil reads the JSON lines into one row for each episode number: the
// file may hold an episode on two lines (the channel's video, and Subsplash's
// share link, before a refresh joins them), and the row has what either
// knows. Rows are oldest first, by date and then number; the list is never
// null.
func readRephil(f *os.File) ([]rephilRow, error) {
	byNumber := map[int]*rephilRow{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var l rephilLine
		if err := json.Unmarshal([]byte(line), &l); err != nil {
			return nil, fmt.Errorf("line %d: %w", n, err)
		}
		row := byNumber[l.Number]
		if row == nil {
			row = &rephilRow{Number: l.Number, Title: l.Title, Date: l.Date}
			byNumber[l.Number] = row
		}
		if row.Title == "" {
			row.Title = l.Title
		}
		if row.Date == "" {
			row.Date = l.Date
		}
		if l.Share != "" && row.Share == nil {
			share := l.Share
			row.Share = &share
		}
		if id := videoID(l.URL); id != "" && row.Video == nil {
			row.Video = &id
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	rows := make([]rephilRow, 0, len(byNumber))
	for _, r := range byNumber {
		rows = append(rows, *r)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Date != rows[j].Date {
			return rows[i].Date < rows[j].Date
		}
		return rows[i].Number < rows[j].Number
	})
	return rows, nil
}

// videoID is the YouTube video id of a watch link (youtube.com/watch?v=ID
// or youtu.be/ID); empty when u is neither.
func videoID(u string) string {
	if u == "" {
		return ""
	}
	p, err := url.Parse(u)
	if err != nil {
		return ""
	}
	switch strings.TrimPrefix(p.Hostname(), "www.") {
	case "youtube.com", "m.youtube.com":
		return p.Query().Get("v")
	case "youtu.be":
		return strings.Trim(p.Path, "/")
	}
	return ""
}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// compareSpan is a reference that holds every passage of the Bible, so that one
// query asks the service for every row of a source that has a passage.
const compareSpan = "Gen.1.1-Rev.22.21"

// comparePage is the page size asked for (the service's largest).
const comparePage = 200

// compareRetries is how many times a page is asked again after a 429, and
// compareSleep how it waits (a test replaces it).
const compareRetries = 5

var compareSleep = time.Sleep

// servedRow is a row of GET /v1/sermons as the service answers it.
type servedRow struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Preacher *string `json:"preacher"`
	Date     *string `json:"date"`
	URL      string  `json:"url"`
	AltURL   *string `json:"alt_url"`
	Ref      string  `json:"ref"`
	Role     string  `json:"role"`
}

// fetchServed is every row the service serves for source, by row id: one
// query for the whole Bible, paged. The service serves a row only through a
// passage, so a row with no ref, or a series, is not among them. A row with
// several passages answers once for each; the one with role "text" is the
// row's own ref.
func fetchServed(client *http.Client, base, source string) (map[string]servedRow, error) {
	got := map[string]servedRow{}
	for offset, total := 0, 1; offset < total; {
		q := url.Values{"ref": {compareSpan}, "source": {source}, "limit": {strconv.Itoa(comparePage)}, "offset": {strconv.Itoa(offset)}}
		req, err := http.NewRequest(http.MethodGet, strings.TrimRight(base, "/")+"/v1/sermons?"+q.Encode(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "sermon-index-compare")
		var body []byte
		for try := 0; ; try++ {
			resp, err := client.Do(req)
			if err != nil {
				return nil, err
			}
			body, err = io.ReadAll(io.LimitReader(resp.Body, 64<<20))
			resp.Body.Close()
			if err != nil {
				return nil, err
			}
			if resp.StatusCode == http.StatusTooManyRequests && try < compareRetries {
				// The service's rate limit: wait as it asks and ask again.
				wait := 5
				if n, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && n > 0 {
					wait = n
				}
				compareSleep(time.Duration(min(wait, 60)) * time.Second)
				continue
			}
			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("%s answered %s", req.URL.Redacted(), resp.Status)
			}
			break
		}
		var page struct {
			Total int         `json:"total"`
			Rows  []servedRow `json:"rows"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("%s: %w", req.URL.Redacted(), err)
		}
		if len(page.Rows) == 0 {
			break
		}
		for _, r := range page.Rows {
			if old, seen := got[r.ID]; !seen || (r.Role == "text" && old.Role != "text") {
				got[r.ID] = r
			}
		}
		total = page.Total
		offset += len(page.Rows)
	}
	return got, nil
}

func orEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// Differing is the ids of the rows (those with a passage, not a series, which
// is all the service serves: a row with a ref, or with cited references only)
// whose title, ref, date, preacher, url or alt_url
// in rows is not what served has, in the order of the rows file, then any id
// the service serves that rows lacks (sorted). n is
// the rows compared.
func Differing(rows []sources.Row, served map[string]servedRow) (ids []string, n int) {
	seen := map[string]bool{}
	for _, r := range rows {
		hasRef := r.Ref != nil && *r.Ref != ""
		if (!hasRef && len(r.Cited) == 0) || (r.Series != nil && *r.Series) {
			continue
		}
		n++
		seen[r.ID] = true
		s, ok := served[r.ID]
		// A row with a ref of its own is served with that ref as its text; a
		// row with cited references only is served by one of them, so its ref
		// is not compared.
		refDiffers := hasRef && (s.Role != "text" || s.Ref != *r.Ref)
		if !ok || s.Title != r.Title || refDiffers || orEmpty(s.Date) != orEmpty(r.Date) ||
			orEmpty(s.Preacher) != orEmpty(r.Preacher) || s.URL != r.URL || orEmpty(s.AltURL) != orEmpty(r.AltURL) {
			ids = append(ids, r.ID)
		}
	}
	var extra []string
	for id := range served {
		if !seen[id] {
			extra = append(extra, id)
		}
	}
	sort.Strings(extra)
	return append(ids, extra...), n
}

// compareMaxNamed is how many row ids the report names.
const compareMaxNamed = 5

// compareService reads the rows file and the service and writes the report:
//
//	compare: N of M rows differ from the service
//	ids: a b c d e
//
// (the second line only when N is not 0, and at most five ids). It reports
// success when it could compare, whether or not the rows differ.
func compareService(rowsPath, base, source string, stdout io.Writer) error {
	rows, err := readRows(rowsPath)
	if err != nil {
		return err
	}
	served, err := fetchServed(&http.Client{Timeout: 60 * time.Second}, base, source)
	if err != nil {
		return err
	}
	ids, n := Differing(rows, served)
	fmt.Fprintf(stdout, "compare: %d of %d rows differ from the service\n", len(ids), n)
	if len(ids) > 0 {
		if len(ids) > compareMaxNamed {
			ids = ids[:compareMaxNamed]
		}
		fmt.Fprintf(stdout, "ids: %s\n", strings.Join(ids, " "))
	}
	return nil
}

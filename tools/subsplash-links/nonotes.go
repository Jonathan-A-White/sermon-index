package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// A sermon with no notes. Some sermons of Brookfield First have no notes in
// the church's app, so no notes mail will ever come for them. The Governor
// reads such a sermon in the app and sends it as a message in Postern: a title,
// an optional series line, a line with the date, the preacher and the passage,
// and a line that begins "No notes":
//
//	The Running Man
//	Easter Sunday 2026
//	Apr 5, 2026 Phil Morgan Luke 24:33-46
//	(No notes)
//
// The shape varies (the preacher or the passage may be missing, the date, the
// preacher and the passage may each sit on a line of their own, a passage may
// be broken over two lines, and one message may hold several sermons), so the
// reader works from the date line: the line or two above it are the title and
// the series, what follows it up to the next title is the preacher and the
// passage. An entry it cannot place it reports by the message's txid and
// writes nothing for; it never guesses.
//
// RunNoNotes writes each sermon it reads as a line of passages.tsv (the church's
// list, which tools/brookfield-youtube merges into the rows) with the id of the
// row it belongs to, and as a line of no-notes.tsv (date, row, the message's
// txid), which the job feed reads to count the sermon's mail as done. A second
// run adds nothing, and a line already there for a date with other words is
// reported, never replaced. Only messages signed by the Governor's key (the
// inbox's verified and signer_checked, and the key itself) are read.

// Sermon is one sermon read from a message.
type Sermon struct {
	Series, Date, Title, Speaker, Passage, Tx string
}

// Unread is an entry or message the reader could not place, and why.
type Unread struct{ Tx, Title, Why string }

var (
	dateRE    = regexp.MustCompile(`(?i)^(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*\.?\s+(\d{1,2}),?\s+(\d{4})\.?(?:\s+(.*))?$`)
	noNotesRE = regexp.MustCompile(`(?i)^\(?\s*no notes`)
	monthNum  = map[string]time.Month{"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6, "jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12}
)

// tailPassageRE finds the longest ending of a text that is a passage: a book
// of the table (or Psalm, Song of Songs), a chapter, and a verse or range.
var tailPassageRE = func() *regexp.Regexp {
	names := map[string]bool{"Psalm": true, "Song of Songs": true}
	for _, b := range passage.Books() {
		names[b.Name] = true
	}
	list := make([]string, 0, len(names))
	for n := range names {
		list = append(list, n)
	}
	sort.Slice(list, func(i, j int) bool { return len(list[i]) > len(list[j]) })
	for i, n := range list {
		list[i] = strings.ReplaceAll(regexp.QuoteMeta(n), `\ `, `\s+`)
	}
	np := passage.NumberPattern
	return regexp.MustCompile(`(?i)(?:^|\s)(((?:` + strings.Join(list, "|") + `))\.?\s+(\d{1,3})(?::(` + np + `))?(?:\s*[-–—]\s*(` + np + `)(?::(` + np + `))?)?)\s*$`)
}()

var bookOSIS = func() map[string]string {
	m := map[string]string{"psalm": "Ps", "song of songs": "Song"}
	for _, b := range passage.Books() {
		m[strings.ToLower(b.Name)] = b.OSIS
	}
	return m
}()

// splitPassage splits text into what comes before its passage (the preacher)
// and the passage, which is the longest ending of the text that the book table
// reads. A text that has none gives ("", "", ""); one whose ending looks like
// a passage but is refused gives the reason in why.
func splitPassage(text string) (before, pass, why string) {
	m := tailPassageRE.FindStringSubmatchIndex(text)
	if m == nil {
		return "", "", ""
	}
	sub := tailPassageRE.FindStringSubmatch(text)
	pass = strings.TrimSpace(sub[1])
	osis, ok := bookOSIS[strings.ToLower(strings.Join(strings.Fields(sub[2]), " "))]
	if !ok {
		return "", "", fmt.Sprintf("the passage %q is refused by the book table", pass)
	}
	ref, err := passage.RangeOSIS(osis, sub[3], sub[4], sub[5], sub[6])
	if err == nil {
		var r passage.Range
		if r, err = passage.Parse(ref); err == nil && r.Chapters() > 3 {
			return "", "", fmt.Sprintf("the passage %q spans %d chapters, which is a series and not a text", pass, r.Chapters())
		}
	}
	if err != nil {
		return "", "", fmt.Sprintf("the passage %q is refused by the book table", pass)
	}
	return strings.TrimSpace(text[:m[2]]), pass, ""
}

func cleanLine(s string) string {
	s = strings.ReplaceAll(s, " ", " ")
	return strings.Join(strings.Fields(s), " ")
}

// cleanSpeaker drops the bullet a message may print after a name.
func cleanSpeaker(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "•", " ")), " ")
}

// ParseNoNotes reads the sermons of one message. candidate is false when the
// message has no line beginning "No notes" (it is not one of these messages and
// nothing is said of it).
func ParseNoNotes(tx, text string) (sermons []Sermon, unread []Unread, candidate bool) {
	type block struct {
		lines  []string
		closed bool // ends at a "No notes" line
	}
	var blocks []block
	var cur []string
	for _, raw := range strings.Split(text, "\n") {
		l := cleanLine(raw)
		if l == "" {
			continue
		}
		if noNotesRE.MatchString(l) {
			candidate = true
			blocks = append(blocks, block{cur, true})
			cur = nil
			continue
		}
		cur = append(cur, l)
	}
	if !candidate {
		return nil, nil, false
	}
	if len(cur) > 0 {
		blocks = append(blocks, block{cur, false})
	}
	for _, b := range blocks {
		s, u := parseBlock(tx, b.lines, b.closed)
		sermons = append(sermons, s...)
		unread = append(unread, u...)
	}
	return sermons, unread, true
}

// parseBlock reads the sermons between two "No notes" lines (or the start or
// end of the message). The lines after the last "No notes" that hold no date
// are not a sermon ("Stopping for now.") and are skipped.
func parseBlock(tx string, lines []string, closed bool) ([]Sermon, []Unread) {
	var dates []int
	for i, l := range lines {
		if dateRE.MatchString(l) {
			dates = append(dates, i)
		}
	}
	if len(dates) == 0 {
		if closed && len(lines) > 0 {
			return nil, []Unread{{tx, lines[0], "no date line (like Nov 16, 2025) before the No notes line"}}
		}
		return nil, nil
	}
	var out []Sermon
	var bad []Unread
	titleLines := lines[:dates[0]]
	titleOK := true
	for k, di := range dates {
		last := k == len(dates)-1
		end := len(lines)
		if !last {
			end = dates[k+1]
		}
		m := dateRE.FindStringSubmatch(lines[di])
		day, _ := strconv.Atoi(m[2])
		year, _ := strconv.Atoi(m[3])
		d := time.Date(year, monthNum[strings.ToLower(m[1])], day, 0, 0, 0, 0, time.UTC)
		date := d.Format("2006-01-02")
		if d.Day() != day {
			bad = append(bad, Unread{tx, firstOr(titleLines, lines[di]), fmt.Sprintf("%q is not a date", lines[di])})
			titleLines, titleOK = nil, false
			continue
		}
		rest := m[4]
		S := lines[di+1 : end]

		speaker, pass, consumed, why := readBody(rest, S, last)
		fail := func(w string) {
			bad = append(bad, Unread{tx, firstOr(titleLines, date), w})
		}
		var left []string
		switch {
		case why != "":
			fail(why)
			// Where the next title starts is not known either.
			titleLines, titleOK = nil, false
			continue
		default:
			left = S[consumed:]
		}
		ok := titleOK
		reason := "its title follows an entry that could not be read, so which line is the title is not known"
		var title, series string
		if ok {
			switch len(titleLines) {
			case 1:
				title = titleLines[0]
			case 2:
				title, series = titleLines[0], titleLines[1]
			case 0:
				ok, reason = false, "no title line above the date"
			default:
				ok, reason = false, fmt.Sprintf("%d lines above the date, want a title and at most a series", len(titleLines))
			}
		}
		if last {
			titleLines, titleOK = nil, true
			if closed && len(left) > 0 {
				ok, reason = false, fmt.Sprintf("%d line(s) after the passage that are not a sermon: %q", len(left), strings.Join(left, " / "))
			}
		} else {
			titleLines, titleOK = left, true
		}
		if !ok {
			bad = append(bad, Unread{tx, firstOr(titleLines, date), reason})
			if title != "" {
				bad[len(bad)-1].Title = title
			}
			continue
		}
		out = append(out, Sermon{Series: series, Date: date, Title: title, Speaker: speaker, Passage: pass, Tx: tx})
	}
	return out, bad
}

func firstOr(l []string, def string) string {
	if len(l) > 0 {
		return l[0]
	}
	return def
}

// readBody reads the preacher and the passage that follow a date: rest is what
// is on the date's line after the date, s the lines after it up to the next
// title. It takes lines one at a time until what it has ends in a passage (at
// most three more lines). With no passage, the body is the preacher alone;
// where the lines end at the next entry's title, that is only certain when the
// number of lines says which are the preacher and which the title.
func readBody(rest string, s []string, last bool) (speaker, pass string, consumed int, why string) {
	acc := rest
	for k := 0; ; k++ {
		before, p, w := splitPassage(acc)
		if w != "" {
			return "", "", 0, w
		}
		if p != "" {
			return cleanSpeaker(before), p, k, ""
		}
		if k >= len(s) || k >= 3 {
			break
		}
		acc = strings.TrimSpace(acc + " " + s[k])
	}
	// No passage.
	if last {
		if len(s) > 2 {
			return "", "", 0, fmt.Sprintf("%d lines after the date and no passage in them", len(s))
		}
		return cleanSpeaker(strings.TrimSpace(rest + " " + strings.Join(s, " "))), "", len(s), ""
	}
	switch len(s) {
	case 1:
		return cleanSpeaker(rest), "", 0, ""
	case 3:
		return cleanSpeaker(strings.TrimSpace(rest + " " + s[0])), "", 1, ""
	}
	return "", "", 0, "no passage, and it cannot be told where the preacher ends and the next title begins"
}

// --- the inbox ---

// Message is one message of the Postern inbox that the reader may use.
type Message struct {
	Tx, Ts, Text string
}

// ReadInbox reads the messages of the inbox directory that are signed by key:
// verified, signer_checked and from the key. Any other message is not read.
func ReadInbox(dir, key string) ([]Message, error) {
	names, err := filepath.Glob(filepath.Join(dir, "*.message.json"))
	if err != nil {
		return nil, err
	}
	var out []Message
	for _, n := range names {
		b, err := os.ReadFile(n)
		if err != nil {
			return nil, err
		}
		var m struct {
			Txid          string `json:"txid"`
			From          string `json:"from"`
			Verified      bool   `json:"verified"`
			SignerChecked bool   `json:"signer_checked"`
			Ts            string `json:"ts"`
			Text          string `json:"text"`
		}
		if json.Unmarshal(b, &m) != nil || m.Text == "" {
			continue
		}
		if !m.Verified || !m.SignerChecked || key == "" || !strings.EqualFold(m.From, key) {
			continue
		}
		out = append(out, Message{Tx: m.Txid, Ts: m.Ts, Text: m.Text})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Ts != out[j].Ts {
			return out[i].Ts < out[j].Ts
		}
		return out[i].Tx < out[j].Tx
	})
	return out, nil
}

// --- the two lists ---

const noNotesHeader = "date\trow\ttxid"

// ReadNoNotes is the set of row ids that no-notes.tsv names. A file that is
// not there names none.
func ReadNoNotes(path string) (map[string]bool, error) {
	ents, err := readNoNotesFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, e := range ents {
		out[e.Row] = true
	}
	return out, nil
}

type noNoteLine struct{ Date, Row, Tx string }

func readNoNotesFile(path string) ([]noNoteLine, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []noNoteLine
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	header := false
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !header {
			if line != noNotesHeader {
				return nil, fmt.Errorf("%s:%d: the header is %q, want %q", path, n, line, noNotesHeader)
			}
			header = true
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 3 {
			return nil, fmt.Errorf("%s:%d: %d columns, want date, row, txid", path, n, len(f))
		}
		out = append(out, noNoteLine{f[0], f[1], f[2]})
	}
	return out, nil
}

type listLine struct{ Series, Title, Speaker, Passage, Row string }

// readPassages reads the date -> line map of passages.tsv and requires the row
// column.
func readPassages(path string) (map[string]listLine, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]listLine{}
	header := false
	for n, line := range strings.Split(string(b), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if !header {
			if len(f) < 6 || strings.TrimSpace(f[5]) != "row" {
				return nil, fmt.Errorf("%s:%d: the header has no row column (series, date, title, speaker, passage, row)", path, n+1)
			}
			header = true
			continue
		}
		if len(f) < 5 {
			continue
		}
		l := listLine{Series: f[0], Title: f[2], Speaker: f[3], Passage: f[4]}
		if len(f) > 5 {
			l.Row = f[5]
		}
		out[strings.TrimSpace(f[1])] = l
	}
	if !header {
		return nil, fmt.Errorf("%s: no header line", path)
	}
	return out, nil
}

// NoNotesResult is what RunNoNotes did.
type NoNotesResult struct {
	Read     int      // sermons read from messages
	Added    int      // sermons newly written to passages.tsv
	Already  int      // sermons already in both lists
	Problems []string // one line each: the txid, the title and why
}

func tsvField(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "\t", " ")), " ")
}

// RunNoNotes reads the Governor's messages in inbox and writes what it read to
// passages.tsv and no-notes.tsv; rowsPath is the rows.jsonl the sermons are
// matched to (the one row dated within a day of the sermon's date).
func RunNoNotes(inbox, key, passagesPath, noNotesPath, rowsPath string) (*NoNotesResult, error) {
	msgs, err := ReadInbox(inbox, key)
	if err != nil {
		return nil, err
	}
	res := &NoNotesResult{}
	var sermons []Sermon
	for _, m := range msgs {
		got, bad, _ := ParseNoNotes(m.Tx, m.Text)
		sermons = append(sermons, got...)
		for _, u := range bad {
			res.Problems = append(res.Problems, fmt.Sprintf("%s: %q: %s", u.Tx, u.Title, u.Why))
		}
	}
	res.Read = len(sermons)
	if len(sermons) == 0 && len(res.Problems) == 0 {
		return res, nil
	}
	pass, err := readPassages(passagesPath)
	if err != nil {
		return nil, err
	}
	nnLines, err := readNoNotesFile(noNotesPath)
	if err != nil {
		return nil, err
	}
	nnByDate := map[string]noNoteLine{}
	nnByRow := map[string]noNoteLine{}
	for _, l := range nnLines {
		nnByDate[l.Date] = l
		nnByRow[l.Row] = l
	}
	rows, err := readRows(rowsPath)
	if err != nil {
		return nil, err
	}

	// Pass 1: which sermons are to be written, and for which row.
	type plan struct {
		s        Sermon
		row      string
		inPass   bool // passages.tsv already holds these words
		inNoNote bool
	}
	var plans []plan
	taken := map[string]int{} // date -> its plan, for the dates settled within this run
	claims := map[string][]int{}
	problem := func(s Sermon, why string) {
		res.Problems = append(res.Problems, fmt.Sprintf("%s: %q: %s", s.Tx, s.Title, why))
	}
	for _, s := range sermons {
		if i, ok := taken[s.Date]; ok {
			if p := plans[i].s; p.Series == s.Series && p.Title == s.Title && p.Speaker == s.Speaker && p.Passage == s.Passage {
				continue // the same words sent twice
			}
			problem(s, fmt.Sprintf("another message gave other words for %s; the first is kept", s.Date))
			continue
		}
		e, inPass := pass[s.Date]
		if inPass && (e.Series != s.Series || e.Title != s.Title || e.Speaker != s.Speaker || e.Passage != s.Passage) {
			problem(s, fmt.Sprintf("passages.tsv already has a line for %s with other words (%q); it is not replaced", s.Date, e.Title))
			continue
		}
		nn, inNN := nnByDate[s.Date]
		if inNN && !inPass {
			problem(s, fmt.Sprintf("no-notes.tsv has a line for %s but passages.tsv does not; left for a person to look at", s.Date))
			continue
		}
		if inNN {
			res.Already++
			taken[s.Date] = len(plans)
			plans = append(plans, plan{s: s, row: nn.Row, inPass: true, inNoNote: true})
			continue
		}
		row, why := rowWithin(rows, s.Date)
		if why != "" {
			problem(s, why)
			continue
		}
		if l, ok := nnByRow[row]; ok {
			problem(s, fmt.Sprintf("row %s already belongs to the sermon of %s in no-notes.tsv", row, l.Date))
			continue
		}
		taken[s.Date] = len(plans)
		claims[row] = append(claims[row], len(plans))
		plans = append(plans, plan{s: s, row: row, inPass: inPass})
	}
	// A row two sermons both fall on belongs to neither.
	drop := map[int]bool{}
	for row, idx := range claims {
		if len(idx) > 1 {
			for _, i := range idx {
				drop[i] = true
				problem(plans[i].s, fmt.Sprintf("row %s is also within a day of the sermon of another date in the messages, so which sermon it holds is not known", row))
			}
		}
	}

	// Pass 2: write.
	var passAdd, nnAdd strings.Builder
	for i, p := range plans {
		if drop[i] || p.inNoNote {
			continue
		}
		if !p.inPass {
			fmt.Fprintf(&passAdd, "%s\t%s\t%s\t%s\t%s\t%s\n", tsvField(p.s.Series), p.s.Date, tsvField(p.s.Title), tsvField(p.s.Speaker), tsvField(p.s.Passage), p.row)
			res.Added++
		} else {
			res.Already++
		}
		fmt.Fprintf(&nnAdd, "%s\t%s\t%s\n", p.s.Date, p.row, p.s.Tx)
	}
	if passAdd.Len() > 0 {
		if err := appendLines(passagesPath, "", passAdd.String()); err != nil {
			return nil, err
		}
	}
	if nnAdd.Len() > 0 {
		if err := appendLines(noNotesPath, noNotesHeader+"\n", nnAdd.String()); err != nil {
			return nil, err
		}
	}
	return res, nil
}

// rowWithin is the id of the one row dated within a day of date.
func rowWithin(rows []sources.Row, date string) (string, string) {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", fmt.Sprintf("%q is not a date", date)
	}
	var found []string
	for _, r := range rows {
		if r.Date == nil {
			continue
		}
		rd, err := time.Parse("2006-01-02", *r.Date)
		if err != nil {
			continue
		}
		if diff := rd.Sub(d); diff >= -24*time.Hour && diff <= 24*time.Hour {
			found = append(found, r.ID)
		}
	}
	switch len(found) {
	case 1:
		return found[0], ""
	case 0:
		return "", fmt.Sprintf("no row of the rows file is dated within a day of %s", date)
	}
	return "", fmt.Sprintf("%d rows (%s) are dated within a day of %s, so none is chosen", len(found), strings.Join(found, ", "), date)
}

// appendLines adds text to the file, first writing header if the file is new,
// and a newline if the file does not end in one.
func appendLines(path, header, text string) error {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	var out []byte
	if len(b) == 0 {
		out = []byte(header + text)
	} else {
		out = b
		if out[len(out)-1] != '\n' {
			out = append(out, '\n')
		}
		out = append(out, text...)
	}
	return os.WriteFile(path, out, 0o644)
}

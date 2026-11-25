// Command clarkson-works writes the sermon source sources/clarkson-works from
// the Internet Archive's OCR of the Nichol edition of David Clarkson's
// Practical Works (3 volumes, Edinburgh 1864-65).
//
//	go run ./tools/clarkson-works [--cache <dir>] [--out <source dir>] [--report <file>] [--evidence <file>]
//
// It fetches the _djvu.xml of the three scans into the cache directory
// (.archive-cache/clarkson-works, git-ignored; a file already there is not
// fetched again; one request at a time, two seconds apart). A sermon is an
// entry of a volume's contents page (catalogue below: its title, its printed
// page and its text, typed from the page images, as the OCR garbles them).
// For each, the tool finds the leaf of the scan where the sermon begins, reads
// the text under its heading and in its running heads, and keeps the row only
// when one of the two reads the text the contents page prints. Nothing of a
// sermon's words is kept: not in the rows, not in the reports, and the cache is never
// committed. Two runs over the same cache write the same bytes.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

const (
	userAgent = "sermon-index-clarkson-works (+https://github.com/Jonathan-A-White/sermon-index)"
	preacher  = "David Clarkson"
)

// Volume is a volume of the edition and the Archive item that scans it.
type Volume struct {
	N    int
	Item string
}

var volumes = []Volume{
	{1, "practicalworksof01clar"},
	{2, "practicalworksof02clar"},
	{3, "practicalworksof03clar"},
}

// OCRURL is the address of an item's OCR, leaf by leaf.
func OCRURL(item string) string {
	return "https://archive.org/download/" + item + "/" + item + "_djvu.xml"
}

// LeafURL is the Archive's reader at a leaf: n counts the leaves of the scan
// from 0, as the OBJECT elements of its _djvu.xml do.
func LeafURL(item string, n int) string {
	return fmt.Sprintf("https://archive.org/details/%s/page/n%d", item, n)
}

// Entry is a sermon of a volume's contents page.
type Entry struct {
	Vol   int
	Page  int    // the page the contents page prints
	Title string // as the contents page prints it, without the final point
	Text  string // the text the contents page prints, as OSIS
}

// ref is the entry's text in the canonical OSIS form, "" when it does not
// resolve.
func (e Entry) ref() string {
	rg, err := passage.Parse(e.Text)
	if err != nil {
		return ""
	}
	return rg.OSIS
}

func (e Entry) id() string { return fmt.Sprintf("v%d-p%d", e.Vol, e.Page) }

// catalogue is the sermons of the contents pages: leaf n10 of volumes 1 and 2
// and leaf n8 of volume 3, read from the page images.
var catalogue = []Entry{
	{1, 3, "Of Original Sin", "Ps.51.5"},
	{1, 16, "Of Repentance", "Luke.13.3"},
	{1, 63, "Of Faith", "Mark.16.16"},
	{1, 176, "Of Living by Faith", "Heb.10.38"},
	{1, 197, "Faith in Prayer", "Jas.1.6"},
	{1, 238, "Of Dying in Faith", "Heb.11.13"},
	{1, 243, "Of Living as Strangers", "Heb.11.13"},
	{1, 247, "The Excellent Knowledge of Christ", "Phil.3.8"},
	{1, 273, "Justification by the Righteousness of Christ", "Phil.3.9"},
	{1, 331, "Men by Nature Unwilling to come to Christ", "John.5.40"},
	{1, 365, "The Lord the Owner of all things; an inducement from Earthly-mindedness", "1Chr.29.11"},
	{1, 428, "Hearing the Word", "Luke.8.18"},
	{1, 447, "Of Taking up the Cross", "Luke.14.27"},
	{2, 3, "The New Creature", "Gal.6.15"},
	{2, 34, "Christ's Gracious Invitation to Sinners", "Rev.3.20"},
	{2, 101, "Man's Insufficiency to do Anything of Himself", "John.15.5"},
	{2, 137, "Against Anxious Carefulness", "Phil.4.6"},
	{2, 172, "Pray for Everything", "Phil.4.6"},
	{2, 185, "God's End in Sending Calamities and Afflictions on his People", "Isa.27.9"},
	{2, 241, "The Conviction of Hypocrites", "Matt.7.22-Matt.7.23"},
	{2, 299, "Soul Idolatry Excludes Men out of Heaven", "Eph.5.5"},
	{2, 334, "The Children of God should not be Partakers with Others in their Sins", "Eph.5.7"},
	{2, 355, "Unconverted Sinners are Darkness", "Eph.5.8"},
	{2, 385, "Of Christ Seeking Fruit, and Finding None", "Luke.13.6"},
	{2, 454, "The Lord Rules over all", "Ps.103.19"},
	{2, 517, "Sinners under the Curse", "Gal.3.10"},
	{3, 3, "The Love of Christ", "Eph.5.2"},
	{3, 47, "Christ's Sacrifice", "Eph.5.2"},
	{3, 63, "Christ's Dying for Sinners", "Rom.5.8"},
	{3, 81, "Christ Touched with the Feeling of our Infirmities", "Heb.4.15"},
	{3, 110, "Of Coming Boldly unto the Throne of Grace", "Heb.4.16"},
	{3, 143, "Of Christ's Making Intercession", "Heb.7.25"},
	{3, 165, "Believers' Communion with the Father and Son", "1John.1.3"},
	{3, 187, "Public Worship to be Preferred before Private", "Ps.87.2"},
}

// notSermons are the other parts the contents pages list, which are not rows.
var notSermons = []string{
	"Volume 1, \"Prefatory Note\": the editor's account of Clarkson, not his sermon.",
	"Volume 3, \"The Practical Divinity of the Papists Discovered to be Destructive of Christianity and Men's Souls\", with its contents (its own pages 1 on): a treatise, not a sermon on a text.",
	"Volume 3, \"General Index\" and \"Index of Scripture Texts\": indexes.",
}

func main() {
	var o options
	flag.StringVar(&o.cache, "cache", ".archive-cache/clarkson-works", "directory for the fetched OCR (not committed)")
	flag.StringVar(&o.out, "out", "sources/clarkson-works", "the source directory to write rows.jsonl in")
	flag.StringVar(&o.report, "report", "sources/clarkson-works/REPORT.md", "write the counts and what was left out to this file")
	flag.StringVar(&o.evidence, "evidence", "tools/clarkson-works/REPORT.md", "write what the tool read in the scan for each row to this file")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: clarkson-works [--cache <dir>] [--out <source dir>] [--report <file>] [--evidence <file>]")
		os.Exit(2)
	}
	if err := run(context.Background(), o, catalogue, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "clarkson-works:", err)
		os.Exit(1)
	}
}

type options struct {
	cache, out, report, evidence string
}

func run(ctx context.Context, o options, cat []Entry, stdout io.Writer) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	if err := fetchVolumes(ctx, client, OCRURL, o.cache, 2*time.Second, stdout); err != nil {
		return err
	}
	vols := map[int][]Leaf{}
	for _, v := range volumes {
		f, err := os.Open(volumePath(o.cache, v.Item))
		if err != nil {
			return err
		}
		leaves, err := ParseDjvu(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", v.Item, err)
		}
		vols[v.N] = leaves
	}
	rows, rep := BuildRows(vols, cat)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, r := range rows {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	for _, f := range []struct {
		path string
		data []byte
	}{
		{filepath.Join(o.out, "rows.jsonl"), buf.Bytes()},
		{o.report, []byte(rep.Markdown())},
		{o.evidence, []byte(rep.Evidence())},
	} {
		if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(f.path, f.data, 0o644); err != nil {
			return err
		}
	}
	fmt.Fprint(stdout, rep.Summary())
	fmt.Fprintf(stdout, "rows.jsonl: %d rows, %d bytes\n", len(rows), buf.Len())
	return nil
}

func volumePath(cache, item string) string {
	return filepath.Join(cache, item+"_djvu.xml")
}

// fetchVolumes fetches the volumes' OCR not yet in the cache, one request at
// a time with delay between them.
func fetchVolumes(ctx context.Context, client *http.Client, urlOf func(item string) string, cache string, delay time.Duration, log io.Writer) error {
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return err
	}
	fetched := 0
	for _, v := range volumes {
		path := volumePath(cache, v.Item)
		if st, err := os.Stat(path); err == nil && st.Size() > 0 {
			continue
		}
		if fetched > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		fetched++
		u := urlOf(v.Item)
		fmt.Fprintf(log, "fetching %s\n", u)
		if err := fetchOne(ctx, client, u, path); err != nil {
			return err
		}
	}
	return nil
}

func fetchOne(ctx context.Context, client *http.Client, u, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", u, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s: %w", u, err)
	}
	if !bytes.Contains(body[:min(len(body), 2000)], []byte("DjVuXML")) {
		return fmt.Errorf("%s: not DjVu XML", u)
	}
	tmp := path + ".part"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Line is a line of a leaf: where it starts, its top, the height of its words
// (their median) and its text.
type Line struct {
	X0, Top, Height int
	Text            string
}

// Leaf is a leaf of a scan, in the OCR's order of lines.
type Leaf struct{ Lines []Line }

// ParseDjvu reads an Archive _djvu.xml: a Leaf for each OBJECT.
func ParseDjvu(r io.Reader) ([]Leaf, error) {
	dec := xml.NewDecoder(r)
	dec.Strict = false
	var leaves []Leaf
	var words []string
	var heights []int
	var word strings.Builder
	x0, top, inWord := 0, 0, false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return leaves, nil
		}
		if err != nil {
			return nil, err
		}
		switch e := tok.(type) {
		case xml.StartElement:
			switch e.Name.Local {
			case "OBJECT":
				leaves = append(leaves, Leaf{})
			case "LINE":
				words, heights = nil, nil
			case "WORD":
				inWord = true
				word.Reset()
				var c []int
				for _, a := range e.Attr {
					if a.Name.Local != "coords" {
						continue
					}
					for _, s := range strings.Split(a.Value, ",") {
						n, err := strconv.Atoi(s)
						if err != nil {
							return nil, fmt.Errorf("leaf n%d: coords %q", len(leaves)-1, a.Value)
						}
						c = append(c, n)
					}
				}
				if len(c) < 4 { // coords are x0,bottom,x1,top
					return nil, fmt.Errorf("leaf n%d: a word with no coords", len(leaves)-1)
				}
				if len(words) == 0 || c[0] < x0 {
					x0 = c[0]
				}
				if len(words) == 0 || c[3] < top {
					top = c[3]
				}
				heights = append(heights, c[1]-c[3])
			}
		case xml.EndElement:
			switch e.Name.Local {
			case "WORD":
				inWord = false
				words = append(words, word.String())
			case "LINE":
				if len(words) > 0 && len(leaves) > 0 {
					l := &leaves[len(leaves)-1]
					l.Lines = append(l.Lines, Line{X0: x0, Top: top, Height: median(heights), Text: strings.Join(words, " ")})
				}
				words, heights = nil, nil
			}
		case xml.CharData:
			if inWord {
				word.Write(e)
			}
		}
	}
}

// median is the lower middle of the numbers, 0 for none.
func median(ns []int) int {
	if len(ns) == 0 {
		return 0
	}
	s := slices.Clone(ns)
	slices.Sort(s)
	return s[(len(s)-1)/2]
}

// headBand is how far below a leaf's highest line a line is still part of its
// running head.
const headBand = 60

// runningHead is the text of a leaf's top line, left to right, and the page
// number printed at one of its ends.
func (lf Leaf) runningHead() (text string, page int, ok bool) {
	if len(lf.Lines) == 0 {
		return "", 0, false
	}
	top := lf.Lines[0].Top
	for _, l := range lf.Lines {
		top = min(top, l.Top)
	}
	var band []Line
	for _, l := range lf.Lines {
		if l.Top <= top+headBand {
			band = append(band, l)
		}
	}
	slices.SortStableFunc(band, func(a, b Line) int { return a.X0 - b.X0 })
	var parts []string
	for _, l := range band {
		parts = append(parts, l.Text)
	}
	text = strings.Join(parts, " ")
	f := strings.Fields(text)
	for _, s := range []string{f[0], f[len(f)-1]} {
		if n, err := strconv.Atoi(s); err == nil {
			return text, n, true
		}
	}
	return text, 0, false
}

// largeLines are the lines of a leaf set as a heading: a quarter taller than
// the leaf's usual line, and mostly capitals.
func (lf Leaf) largeLines() []int {
	var hs []int
	for _, l := range lf.Lines {
		hs = append(hs, l.Height)
	}
	usual := median(hs)
	var out []int
	for i, l := range lf.Lines {
		letters, capitals := 0, 0
		for _, r := range l.Text {
			if unicode.IsLetter(r) {
				letters++
				if unicode.IsUpper(r) {
					capitals++
				}
			}
		}
		if l.Height*4 >= usual*5 && letters >= 3 && capitals*10 >= letters*7 {
			out = append(out, i)
		}
	}
	return out
}

// text is a leaf's lines as one string, a word broken at a line's end joined.
func joinLines(lines []Line) string {
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l.Text)
		b.WriteByte(' ')
	}
	return hyphenBreak.ReplaceAllString(b.String(), "$1$2")
}

var hyphenBreak = regexp.MustCompile(`([a-z])- ([a-z])`)

// keyWords are the words of four letters or more, in lower case, without
// apostrophes: what a title and an OCR of it are compared by.
func keyWords(s string) []string {
	s = strings.NewReplacer("'", "", "’", "").Replace(strings.ToLower(s))
	var out []string
	for _, w := range strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) }) {
		if len(w) >= 4 {
			out = append(out, w)
		}
	}
	return out
}

// matches counts the title's key words that text holds, allowing the OCR a
// wrong letter in a short word and two in a long one.
func matches(title, text string) int {
	found := keyWords(text)
	n := 0
	for _, w := range keyWords(title) {
		allow := 1
		if len(w) >= 7 {
			allow = 2
		}
		if slices.ContainsFunc(found, func(f string) bool { return distance(w, f) <= allow }) {
			n++
		}
	}
	return n
}

// distance is the Levenshtein distance of two words.
func distance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

// refPattern is a reference as the edition prints it: "Ps. LI. 5",
// "1 Chron. XXIX. 11", "Mat. VII. 22, 23". A lower-case l in the numeral is
// the OCR's I.
const refPattern = `(?:\b([123])\s*|\b)([A-Za-z]{2,})\.?\s*([IVXLCivxlc]+)\s*[.,]\s*(\d{1,3})\b(?:\s*([,-])\s*(\d{1,3})\b)?`

var (
	refRe     = regexp.MustCompile(refPattern)
	dashRefRe = regexp.MustCompile(`[—–―~-]\s*` + refPattern)
	romanRe   = regexp.MustCompile(`^C{0,3}(?:XC|XL|L?X{0,3})(?:IX|IV|V?I{0,3})$`)
)

// ReadRefs is every reference in s that reads cleanly and exists, as OSIS.
// A misread book, numeral or verse reads as nothing.
func ReadRefs(s string) []string {
	var out []string
	for _, m := range refRe.FindAllStringSubmatch(s, -1) {
		if r := osis(m); r != "" {
			out = append(out, r)
		}
	}
	return out
}

// headingRef is the text printed under a heading: the first reference that
// follows a dash.
func headingRef(s string) string {
	for _, m := range dashRefRe.FindAllStringSubmatch(s, -1) {
		if r := osis(m); r != "" {
			return r
		}
	}
	return ""
}

// osis turns a match of refPattern into an OSIS reference, "" when the book,
// the numeral or the verse is not one.
func osis(m []string) string {
	name := m[2]
	if m[1] != "" {
		name = m[1] + " " + name
	}
	book, ok := passage.BookOSIS(name)
	if !ok {
		return ""
	}
	chapter := roman(strings.ToUpper(strings.ReplaceAll(m[3], "l", "I")))
	verse, err := strconv.Atoi(m[4])
	if chapter == 0 || err != nil {
		return ""
	}
	ref := fmt.Sprintf("%s.%d.%d", book, chapter, verse)
	// A second verse makes a range after a dash, or when it is the next one;
	// any other number after a comma is not part of the text.
	if last, err := strconv.Atoi(m[6]); err == nil && last > verse && (m[5] == "-" || last == verse+1) {
		ref += fmt.Sprintf("-%s.%d.%d", book, chapter, last)
	}
	rg, err := passage.Parse(ref)
	if err != nil {
		return ""
	}
	return rg.OSIS
}

// roman is the value of a roman numeral up to 399, 0 when s is not one.
func roman(s string) int {
	if s == "" || !romanRe.MatchString(s) {
		return 0
	}
	value := map[byte]int{'I': 1, 'V': 5, 'X': 10, 'L': 50, 'C': 100}
	n := 0
	for i := 0; i < len(s); i++ {
		v := value[s[i]]
		if i+1 < len(s) && v < value[s[i+1]] {
			v = -v
		}
		n += v
	}
	return n
}

// found is what the tool read in the scan for one entry.
type found struct {
	entry   Entry
	located bool
	offset  int            // leaf = printed page + offset
	leaf    int            // where the sermon begins
	heading bool           // the leaf prints the sermon's heading
	last    int            // the sermon's last leaf
	under   string         // the text read under the heading, "" when none is
	heads   map[string]int // the texts of the running heads, a vote a leaf
	slip    bool           // the running heads mostly print another text
	verdict string         // "" for a row
}

// The windows in which the tool looks: pages around the contents page's for
// the running heads, leaves around the expected one for the heading.
const (
	pagesBefore = 10
	pagesAfter  = 60
	leafWindow  = 4
)

// locate finds where an entry's sermon begins: the page numbers of the running
// heads that carry its title say which leaf the contents page's page is on,
// and the heading is looked for around it. A sermon with no heading of its own
// begins on the expected leaf, if that leaf's running head carries the title.
func locate(leaves []Leaf, e Entry) found {
	f := found{entry: e}
	offsets := map[int]int{}
	for i, lf := range leaves {
		head, page, ok := lf.runningHead()
		if ok && page >= e.Page-pagesBefore && page <= e.Page+pagesAfter && matches(e.Title, head) > 0 {
			offsets[i-page]++
		}
	}
	best := 0
	for o, n := range offsets {
		if n > best || n == best && o < f.offset {
			f.offset, best = o, n
		}
	}
	if best == 0 {
		f.verdict = fmt.Sprintf("no running head near page %d carries the title", e.Page)
		return f
	}
	expected := e.Page + f.offset
	score := 0.0
	for i := max(0, expected-leafWindow); i <= expected+leafWindow && i < len(leaves); i++ {
		var parts []string
		for _, j := range leaves[i].largeLines() {
			parts = append(parts, leaves[i].Lines[j].Text)
		}
		head := strings.Join(parts, " ")
		total := len(keyWords(e.Title)) + len(keyWords(head))
		if total == 0 {
			continue
		}
		s := 2 * float64(matches(e.Title, head)) / float64(total)
		nearer := abs(i-expected) < abs(f.leaf-expected)
		if s >= 0.5 && (s > score || s == score && nearer) {
			f.leaf, f.heading, f.located, score = i, true, true, s
		}
	}
	if !f.located && expected >= 0 && expected < len(leaves) {
		if head, _, _ := leaves[expected].runningHead(); matches(e.Title, head) > 0 {
			f.leaf, f.located = expected, true
		}
	}
	if !f.located {
		f.verdict = fmt.Sprintf("neither a heading nor a running head with the title at leaf n%d, where page %d should be", expected, e.Page)
	}
	return f
}

func abs(n int) int { return max(n, -n) }

// readText reads a sermon's text from its leaves: the reference under its
// heading, and the ones in its running heads, a vote a leaf. The row stands
// when the contents page's text is borne out by the heading, or is the one the
// running heads print most often, at least twice: the contents page is not
// taken alone, and one place the OCR misreads or the printer slips does not
// outvote the other two.
func readText(leaves []Leaf, f *found, end int) {
	f.heads, f.last = map[string]int{}, f.leaf
	if f.heading {
		lf := leaves[f.leaf]
		large := lf.largeLines()
		below := lf.Lines[large[len(large)-1]+1:]
		f.under = headingRef(joinLines(below[:min(len(below), 8)]))
	}
	for i := f.leaf; i < end; i++ {
		if i == f.leaf && f.heading {
			continue
		}
		head, _, _ := leaves[i].runningHead()
		if matches(f.entry.Title, head) == 0 {
			continue
		}
		f.last = i
		refs := ReadRefs(head)
		slices.Sort(refs)
		for _, r := range slices.Compact(refs) {
			f.heads[r]++
		}
	}
	most, tied := "", false
	for r, n := range f.heads {
		switch {
		case n > f.heads[most]:
			most, tied = r, false
		case n == f.heads[most]:
			tied = true
		}
	}
	if tied || f.heads[most] < 2 {
		most = ""
	}
	switch want := f.entry.ref(); {
	case want != "" && (f.under == want || most == want):
		f.slip = most != "" && most != want
	default:
		under := f.under
		if under == "" {
			under = "not read"
		}
		f.verdict = fmt.Sprintf("the scan does not bear out %s, the contents page's text (under the heading: %s; running heads: %s)", f.entry.Text, under, tally(f.heads))
	}
}

// tally lists votes, most first.
func tally(votes map[string]int) string {
	if len(votes) == 0 {
		return "none"
	}
	refs := make([]string, 0, len(votes))
	for r := range votes {
		refs = append(refs, r)
	}
	slices.SortFunc(refs, func(a, b string) int {
		if votes[a] != votes[b] {
			return votes[b] - votes[a]
		}
		return strings.Compare(a, b)
	})
	for i, r := range refs {
		refs[i] = fmt.Sprintf("%s ×%d", r, votes[r])
	}
	return strings.Join(refs, ", ")
}

// cites2Pet14 is the edition's way of citing 2 Peter 1:4: "2 Pet. i. 4".
var cites2Pet14 = regexp.MustCompile(`\b2\s*Pet(?:er)?\.?\s*i\s*[.,]\s*4\b`)

// Report is what a run found: the counts the tool prints and what the two
// REPORT.md files list.
type Report struct {
	Read     int // sermons of the contents pages
	Rows     int // rows written
	WithText int // rows whose preaching text resolves

	LeftOut []string // sermons that are not rows, with why
	Slips   []string // rows whose running heads mostly print another text
	OnPeter []string // rows whose text holds 2 Peter 1:4
	Peter   []string // rows whose text is anywhere in 2 Peter
	Cites   []string // pages that cite 2 Peter 1:4

	perVol map[int]int
	table  []string
	found  []found
}

// BuildRows reads each entry of the catalogue in its volume's leaves and gives
// a row for each whose beginning and text the scan bears out, in the
// catalogue's order.
func BuildRows(vols map[int][]Leaf, cat []Entry) ([]sources.Row, *Report) {
	rep := &Report{Read: len(cat), perVol: map[int]int{}}
	all := make([]found, len(cat))
	for i, e := range cat {
		all[i] = locate(vols[e.Vol], e)
	}
	for i := range all {
		f := &all[i]
		if !f.located {
			continue
		}
		// A sermon ends where the next of its volume begins.
		end := len(vols[f.entry.Vol])
		for _, n := range all {
			if n.located && n.entry.Vol == f.entry.Vol && n.leaf > f.leaf {
				end = min(end, n.leaf)
			}
		}
		readText(vols[f.entry.Vol], f, end)
	}

	items := map[int]string{}
	for _, v := range volumes {
		items[v.N] = v.Item
	}
	peter14, _ := passage.Parse("2Pet.1.4")
	peterFrom, _ := passage.Parse("2Pet.1")
	peterTo, _ := passage.Parse("2Pet.3")
	var rows []sources.Row
	for _, f := range all {
		e := f.entry
		if f.verdict != "" {
			rep.LeftOut = append(rep.LeftOut, fmt.Sprintf("Volume %d, page %d, \"%s\": %s.", e.Vol, e.Page, e.Title, f.verdict))
			continue
		}
		page := f.leaf - f.offset
		notes := fmt.Sprintf("Volume %d, page %d.", e.Vol, page)
		if page != e.Page {
			notes = fmt.Sprintf("Volume %d, page %d (the contents page prints %d).", e.Vol, page, e.Page)
		}
		if !f.heading {
			notes += " It begins with no heading of its own."
		}
		if f.slip {
			rep.Slips = append(rep.Slips, fmt.Sprintf("%s, \"%s\": the text under the heading and the contents page agree on %s; the running heads print %s.", e.id(), e.Title, e.Text, tally(f.heads)))
		}
		ref, who := e.ref(), preacher
		rg, err := passage.Parse(ref)
		if err != nil {
			rep.LeftOut = append(rep.LeftOut, fmt.Sprintf("Volume %d, page %d, \"%s\": %v.", e.Vol, e.Page, e.Title, err))
			continue
		}
		r := sources.Row{ID: e.id(), Title: e.Title, Ref: &ref, URL: LeafURL(items[e.Vol], f.leaf), Preacher: &who, Notes: &notes}
		rows = append(rows, r)
		rep.WithText++
		rep.perVol[e.Vol]++
		if rg.Lo <= peter14.Hi && rg.Hi >= peter14.Lo {
			rep.OnPeter = append(rep.OnPeter, fmt.Sprintf("%s, \"%s\"", r.ID, r.Title))
		}
		if rg.Lo <= peterTo.Hi && rg.Hi >= peterFrom.Lo {
			rep.Peter = append(rep.Peter, fmt.Sprintf("%s, \"%s\" (%s)", r.ID, r.Title, ref))
		}
		rep.table = append(rep.table, fmt.Sprintf("| %s | %d, %d | [n%d](%s) | %s | %s |", r.ID, e.Vol, page, f.leaf, r.URL, ref, r.Title))
	}
	rep.Rows = len(rows)
	rep.found = all

	for _, v := range volumes {
		for i, lf := range vols[v.N] {
			if !cites2Pet14.MatchString(joinLines(lf.Lines)) {
				continue
			}
			where := fmt.Sprintf("Volume %d, leaf n%d, outside the sermons", v.N, i)
			for _, f := range all {
				if f.located && f.entry.Vol == v.N && i >= f.leaf && i <= f.last {
					where = fmt.Sprintf("Volume %d, page %d, in \"%s\" ([leaf n%d](%s))", v.N, i-f.offset, f.entry.Title, i, LeafURL(v.Item, i))
				}
			}
			rep.Cites = append(rep.Cites, where)
		}
	}
	return rows, rep
}

// Summary is the counts, as printed by the tool.
func (r *Report) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "sermons read (the contents pages' entries): %d\n", r.Read)
	fmt.Fprintf(&b, "rows:                                       %d\n", r.Rows)
	fmt.Fprintf(&b, "rows with a resolving text:                 %d\n", r.WithText)
	for _, v := range volumes {
		fmt.Fprintf(&b, "  of volume %d:                              %d\n", v.N, r.perVol[v.N])
	}
	fmt.Fprintf(&b, "sermons left out:                           %d\n", len(r.LeftOut))
	fmt.Fprintf(&b, "rows whose text holds 2 Peter 1:4:          %d\n", len(r.OnPeter))
	return b.String()
}

const written = "Written by `go run ./tools/clarkson-works`; do not edit by hand. Only titles, references, page and leaf numbers are listed.\n\n"

// Markdown is the source's REPORT.md: the counts, what was left out and why,
// and the answer on 2 Peter 1:4.
func (r *Report) Markdown() string {
	var b strings.Builder
	b.WriteString("# David Clarkson's Practical Works: what is in the index\n\n<!-- toc: none -->\n\n" + written)
	b.WriteString("## Counts\n\n```\n" + r.Summary() + "```\n\n")
	list := func(items []string) {
		for _, it := range items {
			fmt.Fprintf(&b, "- %s\n", it)
		}
		if len(items) == 0 {
			b.WriteString("None.\n")
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "## Sermons left out (%d)\n\nA sermon is a row only when the scan shows where it begins and bears out the contents page's text: under the sermon's heading, or in most of its running heads (at least two).\n\n", len(r.LeftOut))
	list(r.LeftOut)
	fmt.Fprintf(&b, "## Running heads that print another text (%d)\n\nThe row keeps the contents page's text, which the heading bears out.\n\n", len(r.Slips))
	list(r.Slips)
	b.WriteString("## Parts of the volumes that are not sermons\n\n")
	list(notSermons)
	b.WriteString("## 2 Peter 1:4\n\n")
	if len(r.OnPeter) == 0 {
		b.WriteString("None: no sermon's text is 2 Peter 1:4.")
	} else {
		b.WriteString("Sermons whose text holds 2 Peter 1:4: " + strings.Join(r.OnPeter, "; ") + ".")
	}
	if len(r.Peter) == 0 {
		b.WriteString(" No row's text is anywhere in 2 Peter.\n\n")
	} else {
		b.WriteString(" Rows with a text in 2 Peter: " + strings.Join(r.Peter, "; ") + ".\n\n")
	}
	fmt.Fprintf(&b, "Pages that cite the verse, found by \"2 Pet. i. 4\" in the OCR (a citation the OCR garbled, or one written \"ver. 4\", is missed): %d.\n\n", len(r.Cites))
	list(r.Cites)
	fmt.Fprintf(&b, "## Rows (%d)\n\n| row | volume, page | leaf | text | title |\n| --- | --- | --- | --- | --- |\n", len(r.table))
	for _, l := range r.table {
		b.WriteString(l + "\n")
	}
	return b.String()
}

// Evidence is the tool's REPORT.md: for each sermon, what the scan showed.
func (r *Report) Evidence() string {
	var b strings.Builder
	b.WriteString("# clarkson-works: what the scan showed\n\n<!-- toc: none -->\n\n" + written)
	b.WriteString("For each entry of the contents pages: the leaf where the sermon begins (n counts from 0), whether that leaf prints its heading, the text read under the heading, and every text read in the running heads with the number of leaves that print it (a misread book or number reads as nothing).\n\n")
	b.WriteString("| row | leaf | begins with | under the heading | in the running heads |\n| --- | --- | --- | --- | --- |\n")
	for _, f := range r.found {
		if !f.located {
			fmt.Fprintf(&b, "| %s | not found | | | |\n", f.entry.id())
			continue
		}
		how := "a running head only"
		if f.heading {
			how = "its heading"
		}
		under := f.under
		if under == "" {
			under = "not read"
		}
		fmt.Fprintf(&b, "| %s | n%d | %s | %s | %s |\n", f.entry.id(), f.leaf, how, under, tally(f.heads))
	}
	return b.String()
}

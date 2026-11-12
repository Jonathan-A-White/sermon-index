// Package sources reads, checks and loads a source directory: one
// directory under sources/ holding a source.toml and its rows
// (docs/sources.md).
//
// A sermon source has rows.jsonl, one sermon a line, and may have
// sermon-topics.jsonl, its sermons linked directly to topics of theme
// sources. A theme source has topics.jsonl and links.jsonl. The files are the
// truth; `sermons load` makes the database match them.
package sources

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
	"github.com/Jonathan-A-White/sermon-index/internal/themes"
)

// The files of a source directory.
const (
	SourceFile = "source.toml"
	RowsFile   = "rows.jsonl"
	TopicsFile = "topics.jsonl"
	LinksFile  = "links.jsonl"
	// SermonTopicsFile is a sermon source's direct links from its sermons
	// to topics (optional).
	SermonTopicsFile = "sermon-topics.jsonl"
)

// Kinds of source.
const (
	KindSermons = "sermons"
	KindThemes  = "themes"
)

// Source is source.toml.
type Source struct {
	ID          string    `toml:"id"`
	Name        string    `toml:"name"`
	Kind        string    `toml:"kind"`
	Licence     string    `toml:"licence"`
	LicenceNote string    `toml:"licence_note"`
	Credit      string    `toml:"credit"`
	Homepage    string    `toml:"homepage"`
	TermsURL    string    `toml:"terms_url"`
	Added       time.Time `toml:"added"`
	OnByDefault bool      `toml:"on_by_default"`
	// Language is the language the source's titles are in, a two-letter ISO
	// 639-1 code from Languages (optional; English when it is left out).
	Language string `toml:"language"`
	// By is the name the source's sermons are by, shown for a row that names
	// no preacher (optional): only a source of one church's or one preacher's
	// sermons sets it. It is never written into a row's preacher.
	By string `toml:"by"`

	// HostRules are what `sermons check --urls` does for a host, by host
	// name (optional): see HostRule and DefaultHostRules.
	HostRules map[string]HostRule `toml:"host_rules"`
}

// DefaultLanguage is the language of a source that does not say.
const DefaultLanguage = "en"

// Languages are the codes a source may give as its language, in the order
// the checker names them. Add a code here when a source needs one.
var Languages = []string{"en", "pt", "la", "de", "nl", "fr", "el", "he"}

// LanguageOrDefault is the source's language, DefaultLanguage when it is not given.
func (s Source) LanguageOrDefault() string {
	if s.Language == "" {
		return DefaultLanguage
	}
	return s.Language
}

// HostRule is how `sermons check --urls` treats the urls of one host.
type HostRule struct {
	// TitleInPage: the host answers 200 to an address it does not have
	// (CCEL does), so a 200 proves nothing: GET the page and require the
	// row's title (or its expect_title) in it.
	TitleInPage *bool `toml:"title_in_page"`
	// DelayMS is the wait, in milliseconds, before each request to the
	// host after the first (a host that asks for a crawl delay).
	DelayMS *int `toml:"delay_ms"`
}

// Row is one line of rows.jsonl: a sermon. An optional field is a pointer
// so that a field given empty can be told from one left out.
type Row struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Ref       *string  `json:"ref,omitempty"`
	URL       string   `json:"url"`
	AltURL    *string  `json:"alt_url,omitempty"`
	Kind      *string  `json:"kind,omitempty"`
	Preacher  *string  `json:"preacher,omitempty"`
	Date      *string  `json:"date,omitempty"`
	DurationS *int     `json:"duration_s,omitempty"`
	Cited     []string `json:"cited,omitempty"`
	Notes     *string  `json:"notes,omitempty"`
	// Series: the row is a series or a whole book, wider than one text
	// (three chapters is the most a text has). It is kept and found by its
	// title, but it has no passage and no passage matches it.
	Series *bool `json:"series,omitempty"`
	// ExpectTitle: what `--urls` requires in the page at url, where it
	// differs from the title (a catalogue's short name for a sermon whose
	// page prints a longer one); given, the page is fetched with GET.
	ExpectTitle *string `json:"expect_title,omitempty"`

	Line int `json:"-"` // the line of rows.jsonl it came from

	// Set by the check: the parsed preaching text (nil when the row has
	// none) and cited references.
	text  *passage.Range
	cited []passage.Range
}

// Kind values of a sermon row.
const (
	RowLink       = "link"       // the sermon is elsewhere; the row links to it (the default)
	RowHosted     = "hosted"     // the sermon is a file this service hosts
	RowExposition = "exposition" // an exposition of a passage, not a sermon on a text
)

// IsSeries is whether the row is marked `series`.
func (r Row) IsSeries() bool { return r.Series != nil && *r.Series }

// KindOrDefault is the row's kind, RowLink when it is not given.
func (r Row) KindOrDefault() string {
	if r.Kind == nil {
		return RowLink
	}
	return *r.Kind
}

// SermonTopic is one line of sermon-topics.jsonl: a sermon of this source
// (Row, its id in rows.jsonl) linked directly to a topic (Topic, its id in
// the topics.jsonl of the theme source Theme, a directory beside this one).
type SermonTopic struct {
	Row   string `json:"row"`
	Theme string `json:"theme"`
	Topic string `json:"topic"`

	Line int `json:"-"` // the line of sermon-topics.jsonl it came from
}

// Dir is a source directory read from disk.
type Dir struct {
	Path   string
	Source Source
	Rows   []Row
	Topics []themes.Topic
	Links  []themes.Link
	// A sermon source's direct links to topics; HasSermonTopics is whether
	// it has the file at all (a source without it has none).
	SermonTopics    []SermonTopic
	HasSermonTopics bool
}

// Fault is one thing wrong with a source directory, at a file and line
// (Line 0 for the file as a whole).
type Fault struct {
	File string
	Line int
	Msg  string
}

func (f Fault) String() string {
	if f.Line > 0 {
		return fmt.Sprintf("%s:%d: %s", f.File, f.Line, f.Msg)
	}
	return fmt.Sprintf("%s: %s", f.File, f.Msg)
}

// Faults is a list of faults as an error, one a line.
type Faults []Fault

func (fs Faults) Error() string {
	lines := make([]string, len(fs))
	for i, f := range fs {
		lines[i] = f.String()
	}
	return strings.Join(lines, "\n")
}

// Read reads a source directory: source.toml and the row files its kind
// has. The faults are those of reading (a missing file, a line that is not
// JSON, an unknown key); Check adds the rest.
func Read(dir string) (*Dir, []Fault) {
	d := &Dir{Path: dir}
	var faults []Fault
	add := func(file string, line int, format string, args ...any) {
		faults = append(faults, Fault{File: file, Line: line, Msg: fmt.Sprintf(format, args...)})
	}

	md, err := toml.DecodeFile(filepath.Join(dir, SourceFile), &d.Source)
	switch {
	case errors.Is(err, os.ErrNotExist):
		add(SourceFile, 0, "missing")
	case err != nil:
		add(SourceFile, 0, "not TOML: %v", err)
	default:
		for _, k := range md.Undecoded() {
			add(SourceFile, 0, "unknown key '%s'", k.String())
		}
		for _, k := range []string{"id", "name", "kind", "licence", "credit", "homepage", "terms_url", "added", "on_by_default"} {
			if !md.IsDefined(k) {
				add(SourceFile, 0, "%s is missing", k)
			}
		}
		s := d.Source
		for _, f := range []struct{ key, value string }{
			{"id", s.ID}, {"name", s.Name}, {"kind", s.Kind}, {"licence", s.Licence},
			{"credit", s.Credit}, {"homepage", s.Homepage}, {"terms_url", s.TermsURL},
		} {
			if md.IsDefined(f.key) && strings.TrimSpace(f.value) == "" {
				add(SourceFile, 0, "%s is empty", f.key)
			}
		}
		if md.IsDefined("language") && strings.TrimSpace(s.Language) == "" {
			add(SourceFile, 0, "language is empty: leave it out for English")
		}
		if md.IsDefined("by") && strings.TrimSpace(s.By) == "" {
			add(SourceFile, 0, "by is empty: leave it out instead")
		}
		if md.IsDefined("licence_note") && strings.TrimSpace(s.LicenceNote) == "" {
			add(SourceFile, 0, "licence_note is empty: leave it out instead")
		}
	}

	present := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	switch d.Source.Kind {
	case KindSermons:
		faults = append(faults, readLines(dir, RowsFile, func(line int, dec *json.Decoder) error {
			r := Row{Line: line}
			err := dec.Decode(&r)
			if err == nil {
				d.Rows = append(d.Rows, r)
			}
			return err
		})...)
		if present(SermonTopicsFile) {
			d.HasSermonTopics = true
			faults = append(faults, readLines(dir, SermonTopicsFile, func(line int, dec *json.Decoder) error {
				st := SermonTopic{Line: line}
				err := dec.Decode(&st)
				if err == nil {
					d.SermonTopics = append(d.SermonTopics, st)
				}
				return err
			})...)
		}
		for _, f := range []string{TopicsFile, LinksFile} {
			if present(f) {
				add(f, 0, "a sermons source has no %s", f)
			}
		}
	case KindThemes:
		faults = append(faults, readLines(dir, TopicsFile, func(line int, dec *json.Decoder) error {
			t := themes.Topic{Line: line}
			err := dec.Decode(&t)
			if err == nil {
				d.Topics = append(d.Topics, t)
			}
			return err
		})...)
		faults = append(faults, readLines(dir, LinksFile, func(line int, dec *json.Decoder) error {
			l := themes.Link{Line: line}
			err := dec.Decode(&l)
			if err == nil {
				d.Links = append(d.Links, l)
			}
			return err
		})...)
		for _, f := range []string{RowsFile, SermonTopicsFile} {
			if present(f) {
				add(f, 0, "a themes source has no %s", f)
			}
		}
	}
	return d, faults
}

// maxLine is the longest line a JSONL file may have.
const maxLine = 1 << 20

// readLines decodes each non-blank line of a JSONL file with decode, which
// is given a decoder that refuses unknown fields. It returns a fault for a
// missing or empty file and for each line that does not decode.
func readLines(dir, name string, decode func(line int, dec *json.Decoder) error) []Fault {
	f, err := os.Open(filepath.Join(dir, name))
	if errors.Is(err, os.ErrNotExist) {
		return []Fault{{File: name, Msg: "missing"}}
	}
	if err != nil {
		return []Fault{{File: name, Msg: err.Error()}}
	}
	defer f.Close()

	var faults []Fault
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), maxLine)
	n, rows := 0, 0
	for sc.Scan() {
		n++
		text := bytes.TrimSpace(sc.Bytes())
		if len(text) == 0 {
			continue
		}
		rows++
		dec := json.NewDecoder(bytes.NewReader(text))
		dec.DisallowUnknownFields()
		if err := decode(n, dec); err != nil {
			faults = append(faults, Fault{File: name, Line: n, Msg: jsonFault(err)})
			continue
		}
		if _, err := dec.Token(); err != io.EOF {
			faults = append(faults, Fault{File: name, Line: n, Msg: "more than one JSON value on the line"})
		}
	}
	if err := sc.Err(); err != nil {
		faults = append(faults, Fault{File: name, Line: n + 1, Msg: err.Error()})
	}
	if rows == 0 && len(faults) == 0 {
		faults = append(faults, Fault{File: name, Msg: "no rows"})
	}
	return faults
}

// jsonFault words a decoding error for a person.
func jsonFault(err error) string {
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) || errors.Is(err, io.ErrUnexpectedEOF) {
		return "not a JSON object: " + err.Error()
	}
	return strings.TrimPrefix(err.Error(), "json: ")
}

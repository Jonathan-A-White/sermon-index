package main

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"regexp"
	"strings"
	"unicode"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
	"github.com/Jonathan-A-White/sermon-index/internal/themes"
)

// Entry is one entry of a RawLD dictionary: its key (a topic's name, in
// capitals) and its body (ThML).
type Entry struct {
	Key, Body string
}

// ReadRawLD reads the entries of the RawLD module whose files in the zip are
// dataPath+".idx" and dataPath+".dat". The index is one record of six bytes
// an entry, little-endian: the offset into the data (4) and the length (2);
// an entry is its key, "\r\n", then its body, and a "\n" ends it.
func ReadRawLD(zipPath, dataPath string) ([]Entry, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	read := func(name string) ([]byte, error) {
		for _, f := range zr.File {
			if f.Name == name {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(rc)
			}
		}
		return nil, fmt.Errorf("%s: no %s in the zip", zipPath, name)
	}
	idx, err := read(dataPath + ".idx")
	if err != nil {
		return nil, err
	}
	dat, err := read(dataPath + ".dat")
	if err != nil {
		return nil, err
	}
	if len(idx)%6 != 0 {
		return nil, fmt.Errorf("%s: the index is %d bytes, not a whole number of 6-byte records", zipPath, len(idx))
	}
	var out []Entry
	for i := 0; i < len(idx); i += 6 {
		off := int(binary.LittleEndian.Uint32(idx[i:]))
		size := int(binary.LittleEndian.Uint16(idx[i+4:]))
		if off+size > len(dat) {
			return nil, fmt.Errorf("%s: entry %d runs past the data", zipPath, i/6)
		}
		text := string(dat[off : off+size])
		key, body, _ := strings.Cut(text, "\n")
		out = append(out, Entry{
			Key:  strings.TrimSpace(key),
			Body: strings.TrimRight(body, "\r\n"),
		})
	}
	return out, nil
}

// Built is what a module's entries come to: the topics and their links in
// the order of the module, and the counts and lists the report is made of.
type Built struct {
	Module Module
	Topics []themes.Topic
	Links  []themes.Link

	Groups        int          // <scripRef> groups read
	Unparsed      []Unparsed   // items of a group that did not read
	Renumbered    []Renumbered // topics whose part numbers repeated
	SeeTerms      int          // <term> tags read
	SeeKept       int          // see links written
	SeeUnresolved []string     // "KEY: ..." a term that names no topic
	Collisions    []string     // keys whose slug another key has
	Skipped       []string     // entries with no key
	Parts         int
	Unlabelled    []string // parts the module prints with a number and no label
	Duplicates    int      // links read twice for the same topic, part and reference
	Verses        int
}

// Unparsed is an item of a reference group that did not read.
type Unparsed struct {
	Key   string // the topic's key
	Item  string // the item and why it failed
	Group string // the whole group
}

// Renumbered is a topic whose parts repeat a number (its lettered sections
// each begin again at (1)).
type Renumbered struct {
	Key   string
	Moved int // the parts that took a new number
}

// rawPart is a part as read, before its number is settled.
type rawPart struct {
	n       int    // the printed number, 0 when the module has none
	label   string // as printed (all-capitals made title case)
	section string // the lettered section it is in, title case
}

// rawRef is a reference group read, with the part it is under (-1 none).
type rawRef struct {
	part  int
	group string
}

// rawEntry is one entry read, before ids and numbers.
type rawEntry struct {
	key   string
	parts []rawPart
	refs  []rawRef
	terms []string
}

// Build turns the module's entries into topics and links.
func Build(m Module, entries []Entry) *Built {
	b := &Built{Module: m}

	// Ids first: a see link names a topic by its key, and the topic may be
	// later in the module.
	idOf := map[string]string{}              // slug of a key -> the topic's id
	partOf := map[string]map[string]string{} // slug of half a key -> the topics it is half of
	used := map[string]bool{}                // ids taken
	ids := make([]string, len(entries))
	for i, e := range entries {
		s := slug(e.Key)
		if s == "" {
			b.Skipped = append(b.Skipped, fmt.Sprintf("entry %d has no key", i+1))
			continue
		}
		id := s
		for n := 2; used[id]; n++ {
			id = fmt.Sprintf("%s-%d", s, n)
		}
		if id != s {
			b.Collisions = append(b.Collisions, fmt.Sprintf("%s: the id '%s' is taken by another key, this one is '%s'", e.Key, s, id))
		}
		used[id] = true
		ids[i] = id
		if _, taken := idOf[s]; !taken {
			idOf[s] = id
		}
		// Thompson's terms name a topic by one half of a pair
		// ("POVERTY" for POVERTY-WEALTH) or by the word before the comma
		// ("GRACE" for GRACE, DIVINE).
		halves := strings.Split(e.Key, "-")
		halves = append(halves, strings.Split(e.Key, ",")[0])
		for _, h := range halves {
			if hs := slug(h); hs != "" && hs != s {
				if partOf[hs] == nil {
					partOf[hs] = map[string]string{}
				}
				partOf[hs][id] = id
			}
		}
	}

	type linkKey struct {
		topic string
		part  int
		ref   string
	}
	seenLink := map[linkKey]bool{}

	for i, e := range entries {
		if ids[i] == "" {
			continue
		}
		var raw rawEntry
		if m.Kind == kindThompson {
			raw = readThompson(e)
		} else {
			raw = readTorrey(e)
		}
		t := themes.Topic{ID: ids[i], Label: titleCase(e.Key)}

		// Settle the part numbers.
		nums := make([]int, len(raw.parts))
		taken := map[int]bool{}
		top := 0
		for _, p := range raw.parts {
			if p.n > top {
				top = p.n
			}
		}
		next := top + 1
		moved := 0
		for j, p := range raw.parts {
			n, label := p.n, p.label
			if label == "" { // "<b>(1)</b>" and nothing after it
				label = fmt.Sprintf("Part %d", p.n)
				b.Unlabelled = append(b.Unlabelled, fmt.Sprintf("%s: (%d)", e.Key, p.n))
			}
			if n == 0 || taken[n] {
				if p.n != 0 { // a printed number that repeats
					moved++
					for taken[next] {
						next++
					}
					n = next
					if p.section != "" {
						label = p.section + ": " + label
					}
				}
			}
			if n == 0 { // the module prints none: its order
				n = j + 1
			}
			taken[n] = true
			nums[j] = n
			t.Parts = append(t.Parts, themes.Part{N: n, Label: label})
		}
		if moved > 0 {
			b.Renumbered = append(b.Renumbered, Renumbered{Key: e.Key, Moved: moved})
		}
		b.Parts += len(t.Parts)

		// The links.
		for _, r := range raw.refs {
			b.Groups++
			refs, bad := ParseGroup(r.group)
			for _, item := range bad {
				b.Unparsed = append(b.Unparsed, Unparsed{Key: e.Key, Item: item, Group: r.group})
			}
			for _, ref := range refs {
				part := 0
				if r.part >= 0 {
					part = nums[r.part]
				}
				k := linkKey{ids[i], part, ref}
				if seenLink[k] {
					b.Duplicates++
					continue
				}
				seenLink[k] = true
				l := themes.Link{Topic: ids[i], Ref: ref}
				if r.part >= 0 {
					n := part
					l.Part = &n
				}
				b.Links = append(b.Links, l)
				if rg, err := passage.Parse(ref); err == nil {
					b.Verses += verseCount(rg)
				}
			}
		}

		// The see links.
		seen := map[string]bool{}
		for _, term := range raw.terms {
			b.SeeTerms++
			term = strings.TrimSpace(html.UnescapeString(term))
			if term == "" {
				b.SeeUnresolved = append(b.SeeUnresolved, fmt.Sprintf("%s: an empty <term></term>", e.Key))
				continue
			}
			target, ok := idOf[slug(term)]
			if !ok && len(partOf[slug(term)]) == 1 {
				for _, only := range partOf[slug(term)] {
					target, ok = only, true
				}
			}
			if !ok {
				b.SeeUnresolved = append(b.SeeUnresolved, fmt.Sprintf("%s: '%s' is not a topic of the module, or is half of more than one", e.Key, term))
				continue
			}
			if target == ids[i] || seen[target] {
				continue
			}
			seen[target] = true
			t.See = append(t.See, target)
		}
		b.SeeKept += len(t.See)
		b.Topics = append(b.Topics, t)
	}
	return b
}

// verseCount is the number of verses a range covers.
func verseCount(r passage.Range) int {
	n := 0
	for _, bk := range passage.Books() {
		if passage.Ordinal(bk.N, 999, 999) < r.Lo || passage.Ordinal(bk.N, 0, 0) > r.Hi {
			continue
		}
		for ci, last := range bk.Last {
			lo, hi := passage.Ordinal(bk.N, ci+1, 1), passage.Ordinal(bk.N, ci+1, last)
			a, z := max(lo, r.Lo), min(hi, r.Hi)
			if z >= a {
				n += z - a + 1
			}
		}
	}
	return n
}

var (
	tagRe       = regexp.MustCompile(`<[^>]*>`)
	brRe        = regexp.MustCompile(`(?i)<br\s*/?>`)
	termRe      = regexp.MustCompile(`<term>([^<]*)</term>`)
	scripRefRe  = regexp.MustCompile(`(?s)<scripRef>(.*?)</scripRef>`)
	seeTailRe   = regexp.MustCompile(`(?i)[.,;:\s]*\bSee\b[\s.]*$`)
	partTokenRe = regexp.MustCompile(`(?s)<b>\((\d+)\)\s*([^<,][^<]*)?</b>|<b>\(([A-Z]+)\)\s*([^<,][^<]*)?</b>|<scripRef>(.*?)</scripRef>|<term>([^<]*)</term>|(?:^|<br\s*/?>\s*)\((\d+)\)\s*([^<]*)`)
)

// groupText is the text of a <scripRef>: a group wrapped over a line in the
// module ("Le<br />17:11") reads as one line.
func groupText(s string) string {
	return strings.Join(strings.Fields(tagRe.ReplaceAllString(brRe.ReplaceAllString(s, " "), "")), " ")
}

// plain is s without tags, entities read, its spaces collapsed.
func plain(s string) string {
	return strings.Join(strings.Fields(html.UnescapeString(tagRe.ReplaceAllString(s, ""))), " ")
}

// readThompson reads a Thompson entry: <b>(n) label</b> a part (the label
// after the bold when the bold holds only the number), <b>(A) NAME</b> a
// lettered section, each <scripRef> a group of references for the part
// above it, each <term> the key of a topic it sees.
func readThompson(e Entry) rawEntry {
	raw := rawEntry{key: e.Key}
	section := ""
	cur := -1
	body := e.Body
	for _, m := range partTokenRe.FindAllStringSubmatchIndex(body, -1) {
		sub := func(i int) string {
			if m[2*i] < 0 {
				return ""
			}
			return body[m[2*i]:m[2*i+1]]
		}
		switch {
		case m[2] >= 0 || m[14] >= 0: // a numbered part, bold or not
			n, label, end := atoi(sub(1)), sub(2), m[1]
			if m[14] >= 0 {
				n, label = atoi(sub(7)), sub(8)
			}
			label = plain(label)
			if label == "" {
				// "<b>(1)</b> General references to<br />": the label follows.
				rest := body[end:]
				if k := brRe.FindStringIndex(rest); k != nil {
					rest = rest[:k[0]]
				}
				label = plain(termRe.ReplaceAllString(rest, ""))
			}
			label = cleanPartLabel(label)
			raw.parts = append(raw.parts, rawPart{n: n, label: label, section: section})
			cur = len(raw.parts) - 1
		case m[6] >= 0: // a lettered section
			name := plain(sub(4))
			if name == "" { // "<b>(B)</b> PEACE<br />": the name follows
				rest := body[m[1]:]
				if k := brRe.FindStringIndex(rest); k != nil {
					rest = rest[:k[0]]
				}
				name = plain(termRe.ReplaceAllString(rest, ""))
			}
			section = titleCase(name)
			cur = -1
		case m[10] >= 0: // a reference group
			raw.refs = append(raw.refs, rawRef{part: cur, group: groupText(sub(5))})
		case m[12] >= 0: // a see term
			raw.terms = append(raw.terms, sub(6))
		}
	}
	return raw
}

// cleanPartLabel drops a trailing "See" (the term follows it), a trailing
// full stop, and makes an all-capitals label title case.
func cleanPartLabel(l string) string {
	l = strings.TrimSpace(seeTailRe.ReplaceAllString(l, ""))
	l = strings.TrimSpace(strings.TrimSuffix(l, "."))
	return titleCase(l)
}

// readTorrey reads a Torrey entry, line by line (the lines end in <br />):
// "- " begins a part; ". " begins a line under it, a part of its own when
// it has references, its label the part's, then ": ", then its own; ". See
// <term>KEY</term>" is a see link; a line with none of these, before a
// reference and with no blank line between, goes on the label above it; a
// reference group belongs to the part or line above it.
func readTorrey(e Entry) rawEntry {
	raw := rawEntry{key: e.Key}
	body := scripRefRe.ReplaceAllStringFunc(e.Body, func(g string) string {
		return "<scripRef>" + groupText(g) + "</scripRef>"
	})
	parent := ""
	var item struct {
		label string
		sub   bool
		idx   int  // its place in raw.parts, -1 until it has one
		open  bool // no reference yet and no blank line: the label may go on
		set   bool // an item has begun
	}
	ensure := func() {
		if item.idx < 0 {
			raw.parts = append(raw.parts, rawPart{label: item.label})
			item.idx = len(raw.parts) - 1
		}
	}
	for _, line := range brRe.Split(body, -1) {
		line = strings.TrimSpace(line)
		if line == "" {
			item.open = false
			continue
		}
		text := strings.TrimSpace(scripRefRe.ReplaceAllString(line, ""))
		for _, t := range termRe.FindAllStringSubmatch(text, -1) {
			raw.terms = append(raw.terms, t[1])
		}
		switch {
		case strings.HasPrefix(text, "- "):
			parent = cleanTorreyLabel(text[2:])
			item.label, item.sub, item.idx, item.open, item.set = parent, false, -1, true, true
			ensure()
		case strings.HasPrefix(text, ". "):
			rest := strings.TrimSpace(text[2:])
			if isSee(rest) {
				break
			}
			label := cleanTorreyLabel(rest)
			if parent != "" {
				label = parent + ": " + label
			}
			item.label, item.sub, item.idx, item.open, item.set = label, true, -1, true, true
		case text != "" && item.open:
			more := cleanTorreyLabel(text)
			item.label = joinLabel(item.label, more)
			if !item.sub {
				parent = item.label
			}
			if item.idx >= 0 {
				raw.parts[item.idx].label = item.label
			}
		}
		for _, r := range scripRefRe.FindAllStringSubmatch(line, -1) {
			part := -1
			if item.set {
				ensure()
				part = item.idx
			}
			raw.refs = append(raw.refs, rawRef{part: part, group: r[1]})
			item.open = false
		}
	}
	return raw
}

func joinLabel(a, b string) string { return strings.TrimSpace(a + " " + b) }

// isSee is a line that is only "See <term>KEY</term>".
func isSee(s string) bool {
	return strings.HasPrefix(s, "See ") && termRe.MatchString(s)
}

func cleanTorreyLabel(l string) string {
	l = plain(termRe.ReplaceAllString(l, ""))
	return titleCase(l)
}

// titleCase makes a key or label written in capitals title case: each word
// begins with a capital except the short words of a title (the, of, and),
// which are lower case unless they begin the text or follow a comma, a
// semicolon, a hyphen or a bracket. A text that already has lower case
// letters is returned as it is.
func titleCase(s string) string {
	if s != strings.ToUpper(s) {
		return s
	}
	var b strings.Builder
	runes := []rune(s)
	startOfPhrase := true
	for i := 0; i < len(runes); {
		r := runes[i]
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			b.WriteRune(r)
			if strings.ContainsRune(",;:(-.", r) {
				startOfPhrase = true
			}
			i++
			continue
		}
		j := i
		for j < len(runes) && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j]) || (runes[j] == '\'' && j+1 < len(runes) && unicode.IsLetter(runes[j+1]))) {
			j++
		}
		word := strings.ToLower(string(runes[i:j]))
		if !(smallWords[word] && !startOfPhrase) && !(word == "c" && i > 0 && runes[i-1] == '&') { // "&c"
			wr := []rune(word)
			wr[0] = unicode.ToUpper(wr[0])
			word = string(wr)
		}
		b.WriteString(word)
		startOfPhrase = false
		i = j
	}
	return b.String()
}

var smallWords = map[string]bool{
	"a": true, "an": true, "and": true, "as": true, "at": true, "but": true,
	"by": true, "for": true, "in": true, "nor": true, "of": true, "on": true,
	"or": true, "the": true, "to": true,
}

// slug is s as a topic id: lower case letters and digits, the rest a '-'.
func slug(s string) string {
	var b strings.Builder
	dash := true
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

// WriteJSONL writes the topics and the links, one JSON object a line.
func (b *Built) WriteJSONL(topics, links io.Writer) error {
	enc := func(w io.Writer, v any) error {
		var buf bytes.Buffer
		e := json.NewEncoder(&buf)
		e.SetEscapeHTML(false)
		if err := e.Encode(v); err != nil {
			return err
		}
		_, err := w.Write(buf.Bytes())
		return err
	}
	for _, t := range b.Topics {
		if err := enc(topics, t); err != nil {
			return err
		}
	}
	for _, l := range b.Links {
		if err := enc(links, l); err != nil {
			return err
		}
	}
	return nil
}

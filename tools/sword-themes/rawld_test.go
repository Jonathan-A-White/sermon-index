package main

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Jonathan-A-White/sermon-index/internal/themes"
)

// fixture reads testdata/<name>: entries separated by a line "%%", each a
// key line then the body, as the module's .dat holds them (the module's own
// line ends are "\r\n" after the key; the fixture's are "\n").
func fixture(t *testing.T, name string) []Entry {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var out []Entry
	for _, part := range strings.Split(strings.TrimRight(string(b), "\n"), "\n%%\n") {
		key, body, _ := strings.Cut(part, "\n")
		out = append(out, Entry{Key: key, Body: body})
	}
	return out
}

func build(t *testing.T, name string) *Built {
	t.Helper()
	m := modules[name]
	return Build(m, fixture(t, map[string]string{"TCR": "tcr-three.txt", "Torrey": "torrey-three.txt"}[name]))
}

func topicOf(b *Built, id string) *themes.Topic {
	for i := range b.Topics {
		if b.Topics[i].ID == id {
			return &b.Topics[i]
		}
	}
	return nil
}

func partsOf(tp *themes.Topic) map[int]string {
	m := map[int]string{}
	for _, p := range tp.Parts {
		m[p.N] = p.Label
	}
	return m
}

func linksOf(b *Built, topic string) (all []themes.Link) {
	for _, l := range b.Links {
		if l.Topic == topic {
			all = append(all, l)
		}
	}
	return
}

func hasLink(b *Built, topic string, part int, ref string) bool {
	for _, l := range linksOf(b, topic) {
		if l.Ref == ref && ((part == 0 && l.Part == nil) || (l.Part != nil && *l.Part == part)) {
			return true
		}
	}
	return false
}

func TestThompsonThreeTopics(t *testing.T) {
	b := build(t, "TCR")
	if len(b.Topics) != 3 {
		t.Fatalf("%d topics, want 3", len(b.Topics))
	}
	// The key is the label and its slug the id.
	lh := topicOf(b, "love-hatred")
	if lh == nil || lh.Label != "Love-Hatred" {
		t.Fatalf("love-hatred: %+v", lh)
	}
	sf := topicOf(b, "sinner-s-friend-christ-the")
	if sf == nil || sf.Label != "Sinner's Friend, Christ the" {
		t.Fatalf("sinner's friend: %+v", sf)
	}
	ab := topicOf(b, "abarim-mount")
	if ab == nil || ab.Label != "Abarim, Mount" {
		t.Fatalf("abarim: %+v", ab)
	}

	// A part is <b>(n) label</b>; the number the book prints, the first
	// time it is printed in the topic.
	parts := partsOf(lh)
	if parts[4] != "Of God" || parts[5] != "To God Enjoined" || parts[7] != "The Pre-eminence of Love" {
		t.Errorf("parts of love-hatred: %v", parts)
	}
	// "<b>(1)</b> General references to": the label is after the bold.
	if parts[1] != "General references to" {
		t.Errorf("part 1: %q", parts[1])
	}
	if !hasLink(b, "love-hatred", 4, "Rom.5.8") || !hasLink(b, "love-hatred", 4, "Eph.2.4") || !hasLink(b, "love-hatred", 4, "1John.4.16") {
		t.Errorf("links of part 4: %+v", linksOf(b, "love-hatred"))
	}
	// A topic with no parts: its links have no part.
	if !hasLink(b, "sinner-s-friend-christ-the", 0, "Rom.5.8") || !hasLink(b, "sinner-s-friend-christ-the", 0, "Matt.11.19") {
		t.Errorf("links of sinner's friend: %+v", linksOf(b, "sinner-s-friend-christ-the"))
	}
	if !hasLink(b, "abarim-mount", 0, "Num.27.12") || !hasLink(b, "abarim-mount", 0, "Deut.32.49") {
		t.Errorf("links of abarim: %+v", linksOf(b, "abarim-mount"))
	}

	// The lettered sections (A) to (D) each begin again at (1): the repeat
	// of a number takes a number past the topic's printed ones, and its
	// label says which section it is of.
	seen := map[string]int{}
	for n, l := range parts {
		if prev, dup := seen[l]; dup {
			t.Errorf("label %q is parts %d and %d", l, prev, n)
		}
		seen[l] = n
	}
	if len(lh.Parts) <= 7 {
		t.Errorf("love-hatred has %d parts: the sections' repeats were lost", len(lh.Parts))
	}
	var hatredSaints int
	for n, l := range parts {
		if l == "Hatred: Of the Saints" {
			hatredSaints = n
		}
	}
	if hatredSaints <= 7 || !hasLink(b, "love-hatred", hatredSaints, "Isa.66.5") {
		t.Errorf("'Hatred: Of the Saints' is part %d: %+v", hatredSaints, linksOf(b, "love-hatred"))
	}
	if len(b.Renumbered) != 1 || b.Renumbered[0].Key != "LOVE-HATRED" {
		t.Errorf("renumbered: %+v", b.Renumbered)
	}

	// --SEE X, <term>KEY</term> names a topic of the module; the fixture's
	// three topics are none of those the terms name, so they are listed and
	// none is kept (TestSeeResolvesBySlugAndNotToItself has topics that do).
	if len(lh.See) != 0 {
		t.Errorf("see of love-hatred: %v", lh.See)
	}
	listed := false
	for _, u := range b.SeeUnresolved {
		if strings.Contains(u, "LOVE-HATRED: 'SYMPATHY' is not a topic") {
			listed = true
		}
	}
	if !listed {
		t.Errorf("unresolved: %v", b.SeeUnresolved)
	}
	// "Outcasts Received, <term></term>": an empty term is read and listed.
	if len(sf.See) != 0 {
		t.Errorf("see of sinner's friend: %v", sf.See)
	}
	if b.SeeTerms == 0 || b.SeeUnresolved == nil {
		t.Errorf("see terms %d, unresolved %v", b.SeeTerms, b.SeeUnresolved)
	}
	found := false
	for _, u := range b.SeeUnresolved {
		if strings.Contains(u, "SINNER'S FRIEND, CHRIST THE") {
			found = true
		}
	}
	if !found {
		t.Errorf("the empty term is not listed: %v", b.SeeUnresolved)
	}
}

func TestTorreyThreeTopics(t *testing.T) {
	b := build(t, "Torrey")
	if len(b.Topics) != 3 {
		t.Fatalf("%d topics, want 3", len(b.Topics))
	}
	lg := topicOf(b, "love-of-god-the")
	if lg == nil || lg.Label != "Love of God, The" {
		t.Fatalf("love of god: %+v", lg)
	}
	// A line beginning "- " is a part, numbered as it comes.
	parts := partsOf(lg)
	if parts[1] != "Is a part of his character" || parts[2] != "Christ, the especial object of" {
		t.Errorf("parts: %v", parts)
	}
	if !hasLink(b, "love-of-god-the", 1, "1John.4.8") || !hasLink(b, "love-of-god-the", 2, "John.15.9") {
		t.Errorf("links: %+v", linksOf(b, "love-of-god-the"))
	}
	// A line beginning ". " under a part is a part of its own whose label
	// begins with the line it is under, and its links are its own.
	var sov int
	for n, l := range parts {
		if l == "Described as: Sovereign" {
			sov = n
		}
	}
	if sov == 0 || !hasLink(b, "love-of-god-the", sov, "Deut.7.8") {
		t.Errorf("'Described as: Sovereign' is part %d: parts %v", sov, parts)
	}
	// Romans 5:8 is in 'Christ's dying for us while sinners'.
	var dying int
	for n, l := range parts {
		if l == "Exhibited in: Christ's dying for us while sinners" {
			dying = n
		}
	}
	if dying == 0 || !hasLink(b, "love-of-god-the", dying, "Rom.5.8") {
		t.Errorf("part %d: %+v", dying, linksOf(b, "love-of-god-the"))
	}
	// ". See <term>PRAYER</term>": a see link, to a topic the fixture does
	// not have, so it is listed and not kept.
	at := topicOf(b, "access-to-god")
	if at == nil || len(at.See) != 0 {
		t.Errorf("access to god: %+v", at)
	}
	if b.SeeTerms != 2 || len(b.SeeUnresolved) != 2 {
		t.Errorf("see terms %d, unresolved %v", b.SeeTerms, b.SeeUnresolved)
	}
	// A part with sub-lines but no links of its own ("DESCRIBED AS") is
	// there, with no links.
	if hasLink(b, "love-of-god-the", findPart(parts, "Described as"), "Deut.7.8") {
		t.Error("a heading's own part has its sub-lines' links")
	}
}

func findPart(parts map[int]string, label string) int {
	for n, l := range parts {
		if l == label {
			return n
		}
	}
	return -1
}

func TestTorreyContinuedLabelAndSplitGroup(t *testing.T) {
	body := "- The blood of sacrifices put on the horns and poured at the<br />\nfoot of<br />\n<scripRef>Ex 29:12; Le 4:7</scripRef><br /><br />\n" +
		"- Typified<br />\n<scripRef>Ge 4:4; Le<br />17:11; Heb 9:22</scripRef>\n"
	b := Build(modules["Torrey"], []Entry{{Key: "ALTAR, THE", Body: body}})
	tp := topicOf(b, "altar-the")
	if tp == nil {
		t.Fatal("no topic")
	}
	parts := partsOf(tp)
	if parts[1] != "The blood of sacrifices put on the horns and poured at the foot of" || parts[2] != "Typified" {
		t.Errorf("parts: %v", parts)
	}
	if !hasLink(b, "altar-the", 2, "Lev.17.11") || !hasLink(b, "altar-the", 2, "Heb.9.22") || !hasLink(b, "altar-the", 1, "Lev.4.7") {
		t.Errorf("links: %+v", b.Links)
	}
}

func TestLabelCase(t *testing.T) {
	cases := map[string]string{
		"LOVE-HATRED":                   "Love-Hatred",
		"SINNER'S FRIEND, CHRIST THE":   "Sinner's Friend, Christ the",
		"ATONEMENT, THE":                "Atonement, The",
		"RECONCILIATION WITH GOD":       "Reconciliation With God",
		"LAW OF GOD, THE":               "Law of God, The",
		"BIBLE, THE; THE WORD OF GOD":   "Bible, The; The Word of God",
		"JOPPA OR JAPHO (MODERN JAFFA)": "Joppa or Japho (Modern Jaffa)",
		"HERBS, &C":                     "Herbs, &c",
		"Already Mixed":                 "Already Mixed",
	}
	for in, want := range cases {
		if got := titleCase(in); got != want {
			t.Errorf("titleCase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"LOVE-HATRED":                 "love-hatred",
		"SINNER'S FRIEND, CHRIST THE": "sinner-s-friend-christ-the",
		"HERBS, &C":                   "herbs-c",
		"  Spaced  Out ":              "spaced-out",
	}
	for in, want := range cases {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSlugCollisionGetsASuffix(t *testing.T) {
	b := Build(modules["Torrey"], []Entry{
		{Key: "A B", Body: "- x<br /><scripRef>Ro 5:8</scripRef>"},
		{Key: "A-B", Body: "- y<br /><scripRef>Ro 5:9</scripRef>"},
	})
	if len(b.Topics) != 2 || b.Topics[0].ID != "a-b" || b.Topics[1].ID != "a-b-2" {
		t.Errorf("topics: %+v", b.Topics)
	}
}

func TestSeeResolvesBySlugAndNotToItself(t *testing.T) {
	b := Build(modules["TCR"], []Entry{
		{Key: "ENVY", Body: "<scripRef>Ro 5:8</scripRef><br /><b>--SEE</b> Envy, <term>ENVY</term><br />Strife, <term>strife</term><br />& <term>STRIFE</term><br />Lost, <term>NOWHERE</term>"},
		{Key: "STRIFE", Body: "<scripRef>Ro 5:9</scripRef>"},
	})
	envy := topicOf(b, "envy")
	if !reflectEqual(envy.See, []string{"strife"}) {
		t.Errorf("see: %v", envy.See)
	}
	if b.SeeTerms != 4 || len(b.SeeUnresolved) != 1 || !strings.Contains(b.SeeUnresolved[0], "NOWHERE") {
		t.Errorf("terms %d unresolved %v", b.SeeTerms, b.SeeUnresolved)
	}
}

func TestSeeNamesAHalfOfAPairOnlyWhenOneTopicHasIt(t *testing.T) {
	b := Build(modules["TCR"], []Entry{
		{Key: "ABUNDANCE-WANT", Body: "<scripRef>Ro 5:8</scripRef><br /><b>--SEE</b> Poverty, <term>POVERTY</term><br />Grace, <term>GRACE</term><br />Hope, <term>HOPE</term>"},
		{Key: "POVERTY-RICHES", Body: "<scripRef>Ro 5:9</scripRef>"},
		{Key: "GRACE, DIVINE", Body: "<scripRef>Ro 5:10</scripRef>"},
		{Key: "HOPE-DESPAIR", Body: "<scripRef>Ro 5:11</scripRef>"},
		{Key: "HOPE-FEAR", Body: "<scripRef>Ro 5:12</scripRef>"},
	})
	if got := topicOf(b, "abundance-want").See; !reflectEqual(got, []string{"poverty-riches", "grace-divine"}) {
		t.Errorf("see: %v", got)
	}
	if len(b.SeeUnresolved) != 1 || !strings.Contains(b.SeeUnresolved[0], "'HOPE'") {
		t.Errorf("unresolved: %v", b.SeeUnresolved)
	}
}

func TestUnparsedGroupsAreListedAndTheRestKept(t *testing.T) {
	b := Build(modules["TCR"], []Entry{{Key: "ODD", Body: "<scripRef>Ro 5:8; Xx 3:1; Ro 5:99</scripRef>"}})
	if !hasLink(b, "odd", 0, "Rom.5.8") || len(b.Links) != 1 {
		t.Errorf("links: %+v", b.Links)
	}
	if b.Groups != 1 || len(b.Unparsed) != 2 {
		t.Fatalf("groups %d unparsed %+v", b.Groups, b.Unparsed)
	}
	if b.Unparsed[0].Key != "ODD" || !strings.Contains(b.Unparsed[0].Item, "Xx 3:1") {
		t.Errorf("unparsed: %+v", b.Unparsed)
	}
}

func TestAPartWithNoLabelIsCalledPartN(t *testing.T) {
	b := Build(modules["TCR"], []Entry{{Key: "SHECHEM", Body: "<b>(1)</b><br /><br /><scripRef>Ge 12:6</scripRef><br /><b>(2)</b> City of<br /><scripRef>Jos 20:7</scripRef>"}})
	parts := partsOf(topicOf(b, "shechem"))
	if parts[1] != "Part 1" || parts[2] != "City of" || len(b.Unlabelled) != 1 || !hasLink(b, "shechem", 1, "Gen.12.6") {
		t.Errorf("parts %v unlabelled %v links %+v", parts, b.Unlabelled, b.Links)
	}
}

func TestDuplicateLinksAreWrittenOnce(t *testing.T) {
	b := Build(modules["Torrey"], []Entry{{Key: "X", Body: "- p<br /><scripRef>Ro 5:8</scripRef><br /><scripRef>Ro 5:8; Ro 5:9</scripRef>"}})
	if len(b.Links) != 2 {
		t.Errorf("links: %+v", b.Links)
	}
}

func TestReadRawLD(t *testing.T) {
	entries := []string{"AARON\r\nson of Amram<br />\n", "ABEL\r\nson of Adam\n"}
	var idx, dat bytes.Buffer
	for _, e := range entries {
		var rec [6]byte
		binary.LittleEndian.PutUint32(rec[:4], uint32(dat.Len()))
		binary.LittleEndian.PutUint16(rec[4:], uint16(len(e)))
		idx.Write(rec[:])
		dat.WriteString(e)
	}
	var zb bytes.Buffer
	zw := zip.NewWriter(&zb)
	for name, data := range map[string][]byte{"modules/lexdict/rawld/x/x.idx": idx.Bytes(), "modules/lexdict/rawld/x/x.dat": dat.Bytes(), "mods.d/x.conf": []byte("[X]\n")} {
		w, _ := zw.Create(name)
		w.Write(data)
	}
	zw.Close()
	path := filepath.Join(t.TempDir(), "X.zip")
	if err := os.WriteFile(path, zb.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadRawLD(path, "modules/lexdict/rawld/x/x")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Key != "AARON" || got[0].Body != "son of Amram<br />" || got[1].Key != "ABEL" || got[1].Body != "son of Adam" {
		t.Errorf("entries: %+v", got)
	}
	if _, err := ReadRawLD(path, "modules/lexdict/rawld/nope/nope"); err == nil {
		t.Error("a missing module is no error")
	}
}

func TestWriteIsDeterministicAndHasTheFormat(t *testing.T) {
	b := build(t, "TCR")
	var t1, l1, t2, l2 bytes.Buffer
	if err := b.WriteJSONL(&t1, &l1); err != nil {
		t.Fatal(err)
	}
	if err := b.WriteJSONL(&t2, &l2); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(t1.Bytes(), t2.Bytes()) || !bytes.Equal(l1.Bytes(), l2.Bytes()) {
		t.Error("two writes differ")
	}
	lines := strings.Split(strings.TrimRight(t1.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("%d topic lines", len(lines))
	}
	var tp themes.Topic
	dec := json.NewDecoder(strings.NewReader(lines[0]))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&tp); err != nil || tp.ID != "love-hatred" {
		t.Errorf("first line: %v %+v", err, tp)
	}
	if strings.Contains(l1.String(), `"via"`) || strings.Contains(l1.String(), `"from"`) {
		t.Error("a source link carries via or from")
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func reflectEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

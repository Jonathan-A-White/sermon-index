package main

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func fixture(t *testing.T) []Entry {
	t.Helper()
	f, err := os.Open("testdata/sermons02-two.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	es, err := ParseVolume(f, 2)
	if err != nil {
		t.Fatal(err)
	}
	return es
}

func TestParseVolumeFixture(t *testing.T) {
	es := fixture(t)
	if len(es) != 2 {
		t.Fatalf("got %d entries, want 2 (the Indexes division is not a sermon): %+v", len(es), es)
	}

	exp, s := es[0], es[1]
	if !reflect.DeepEqual(exp.Numbers, []int{62}) || exp.Title != "Exposition: 1 John 3:1-10" || exp.DivID != "ix" {
		t.Errorf("exposition: %+v", exp)
	}
	if !exp.Exposition || len(exp.Texts) != 0 {
		t.Errorf("exposition: want an exposition with no preaching text, got %+v", exp)
	}
	if want := []string{"1John.3.1-1John.3.10"}; !reflect.DeepEqual(exp.Cited, want) {
		t.Errorf("exposition cited = %v, want %v", exp.Cited, want)
	}

	if !reflect.DeepEqual(s.Numbers, []int{104}) || s.Title != "Love's Commendation" || s.DivID != "l" || s.Vol != 2 {
		t.Errorf("sermon: %+v", s)
	}
	if want := []string{"Rom.5.8"}; !reflect.DeepEqual(s.Texts, want) {
		t.Errorf("sermon texts = %v, want %v", s.Texts, want)
	}
	// Every tagged reference, the text too; a reference given twice once.
	if want := []string{"Rom.5.8", "John.3.16", "Rom.5.1-Rom.5.11"}; !reflect.DeepEqual(s.Cited, want) {
		t.Errorf("sermon cited = %v, want %v", s.Cited, want)
	}
	if s.Exposition {
		t.Error("sermon 104 is not an exposition")
	}
	if s.Date != "1856-11-23" {
		t.Errorf("date = %q, want 1856-11-23", s.Date)
	}
}

func TestURLs(t *testing.T) {
	if got, want := PageURL(2, "l"), "https://ccel.org/ccel/spurgeon/sermons02/sermons02.l.html"; got != want {
		t.Errorf("PageURL = %s, want %s", got, want)
	}
	if got, want := PageURL(63, "xix"), "https://ccel.org/ccel/spurgeon/sermons63/sermons63.xix.html"; got != want {
		t.Errorf("PageURL = %s, want %s", got, want)
	}
	if got, want := GemsURL("104"), "https://www.spurgeongems.org/sermon/chs104.pdf"; got != want {
		t.Errorf("GemsURL = %s, want %s", got, want)
	}
	if got, want := GemsURL("1451A"), "https://www.spurgeongems.org/sermon/chs1451A.pdf"; got != want {
		t.Errorf("GemsURL = %s, want %s", got, want)
	}
	if got, want := VolumeURL(7), "https://ccel.org/ccel/spurgeon/sermons07.xml"; got != want {
		t.Errorf("VolumeURL = %s, want %s", got, want)
	}
}

func TestRowsFixture(t *testing.T) {
	rows, rep := BuildRows(fixture(t))
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	exp, s := rows[0], rows[1]
	if exp.ID != "62" || s.ID != "104" {
		t.Fatalf("ids %s, %s (rows are in number order)", exp.ID, s.ID)
	}

	if s.Title != "Love's Commendation" || *s.Ref != "Rom.5.8" || s.Kind != nil {
		t.Errorf("104: %+v", s)
	}
	if s.URL != "https://ccel.org/ccel/spurgeon/sermons02/sermons02.l.html" {
		t.Errorf("104 url = %s", s.URL)
	}
	if s.AltURL == nil || *s.AltURL != "https://www.spurgeongems.org/sermon/chs104.pdf" {
		t.Errorf("104 alt_url = %v", s.AltURL)
	}
	if s.Preacher == nil || *s.Preacher != "Charles H. Spurgeon" {
		t.Errorf("104 preacher = %v", s.Preacher)
	}
	if s.Date == nil || *s.Date != "1856-11-23" {
		t.Errorf("104 date = %v", s.Date)
	}
	if want := []string{"John.3.16", "Rom.5.1-Rom.5.11"}; !reflect.DeepEqual(s.Cited, want) {
		t.Errorf("104 cited = %v", s.Cited)
	}

	// An exposition has no preaching text, is of kind exposition and has no
	// Spurgeon Gems pdf of its own.
	if exp.Ref != nil || exp.Kind == nil || *exp.Kind != "exposition" || exp.AltURL != nil {
		t.Errorf("62: %+v", exp)
	}
	if want := []string{"1John.3.1-1John.3.10"}; !reflect.DeepEqual(exp.Cited, want) {
		t.Errorf("62 cited = %v", exp.Cited)
	}

	if rep.Numbers != 2 || rep.WithText != 1 {
		t.Errorf("report: %+v", rep)
	}
}

func TestParseHeading(t *testing.T) {
	for _, c := range []struct {
		in      string
		nums    []int
		suffix  string
		title   string
		ok      bool
		comment string
	}{
		{"Sermon 37. Law and Grace", []int{37}, "", "Law and Grace", true, ""},
		{"Sermon 385-388. Exposition of the Doctrines of Grace", []int{385, 386, 387, 388}, "", "Exposition of the Doctrines of Grace", true, "a combined entry"},
		{"Sermon 297-8. Mr. Evil-Questioning Tried and Executed", []int{297, 298}, "", "Mr. Evil-Questioning Tried and Executed", true, "a short end"},
		{"Sermon 1451A. \"This Year Also\"", []int{1451}, "A", "\"This Year Also\"", true, "a half of a number"},
		{"Sermon 82.htm. The New Park Street Tracts", []int{82}, "", "The New Park Street Tracts", true, "a file name in the number"},
		{"Sermon sermons0221a.htm. A Pastoral Letter", nil, "", "", false, "no number"},
		{"Indexes", nil, "", "", false, ""},
	} {
		nums, suffix, title, ok := ParseHeading(c.in)
		if ok != c.ok || !reflect.DeepEqual(nums, c.nums) || suffix != c.suffix || title != c.title {
			t.Errorf("%q (%s): got %v %q %q %v", c.in, c.comment, nums, suffix, title, ok)
		}
	}
}

func TestHeadDate(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"Delivered on Sabbath Morning, November 23, 1856, by the", "1856-11-23"},
		{"Delivered on Lord's-day Evening, March 18th, 1855", "1855-03-18"},
		{"Published on Thursday, May 4th, 1893. Delivered on Lord's-day Evening, July 5th, 1874", "1874-07-05"},
		{"Intended for Reading on Lord's-day, April 2nd, 1899. Delivered on Thursday Evening, June, 1873", "1873-06"},
		{"Intended for Reading on Lord's-day, April 2nd, 1899", ""},
		{"nothing of a date here", ""},
		{"Delivered in 1587 (a typing slip)", ""},
		{"DELIVERED BY C. H. SPURGEON, AT THE TABERNACLE, ON LORD'S-DAY EVENING, MARCH 2, 1900.", ""},
		{"Delivered at Southwark. \"The text.\" [On January 31, 1892 an editor's note]", ""},
		{"Nos. 385-88. Thursday, April 11th, 1861.", "1861-04-11"},
	} {
		if got := headDate(c.in); got != c.want {
			t.Errorf("headDate(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestOSISFromParsed(t *testing.T) {
	for in, want := range map[string]string{
		"|Rom|5|8|0|0":   "Rom.5.8",
		"|1Cor|2|9|2|10": "1Cor.2.9-1Cor.2.10",
		"|Ps|23|0|0|0":   "Ps.23",
		"|Rom|5|1|6|4":   "Rom.5.1-Rom.6.4",
		"|Jude|0|0|0|0":  "Jude",
		"garbage":        "",
		"":               "",
		"|Rom|x|8|0|0":   "",
	} {
		if got := osisFromParsed(in); got != want {
			t.Errorf("osisFromParsed(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCombinedTo(t *testing.T) {
	for _, c := range []struct {
		in   string
		lo   int
		want int
	}{
		{"Nos. 385-88. Thursday, April 11th, 1861.", 385, 388},
		{"Nos. 385-388.", 385, 388},
		{"No. 104", 104, 0},
		{"Nos. 385-88.", 400, 0},
	} {
		if got := combinedTo(c.in, c.lo); got != c.want {
			t.Errorf("combinedTo(%q, %d) = %d, want %d", c.in, c.lo, got, c.want)
		}
	}
}

func TestCombinedAndSlipRows(t *testing.T) {
	es := []Entry{
		{Vol: 7, DivID: "viii_1", Numbers: []int{385, 386, 387, 388}, Title: "Exposition of the Doctrines of Grace", Exposition: true},
		{Vol: 12, DivID: "xvi", Numbers: []int{683}, Corrected: "6823", Title: "Divine Gentleness Acknowledged", Texts: []string{"Ps.18.35"}, Cited: []string{"Ps.18.35"}},
	}
	rows, rep := BuildRows(es)
	if len(rows) != 5 || rep.Numbers != 5 {
		t.Fatalf("%d rows, %d numbers", len(rows), rep.Numbers)
	}
	for _, r := range rows[1:4] {
		if r.URL != "https://ccel.org/ccel/spurgeon/sermons07/sermons07.viii_1.html" || r.Notes == nil || !strings.Contains(*r.Notes, "385 to 388") {
			t.Errorf("%s: %+v", r.ID, r)
		}
	}
	last := rows[4]
	if last.ID != "683" || last.Notes == nil || !strings.Contains(*last.Notes, "6823") {
		t.Errorf("683: %+v", last)
	}
	if len(last.Cited) != 0 {
		t.Errorf("the text is not also cited: %v", last.Cited)
	}
}

func TestGemsRows(t *testing.T) {
	es := []Entry{
		{Vol: 6, DivID: "xii", Numbers: []int{297, 298}, Title: "A"},
		{Vol: 25, DivID: "a", Numbers: []int{1451}, Suffix: "A", Title: "B"},
		{Vol: 25, DivID: "b", Numbers: []int{1451}, Suffix: "B", Title: "C"},
		{Vol: 24, DivID: "c", Numbers: []int{1331}, Title: "D"},
		{Vol: 24, DivID: "d", Numbers: []int{1331}, Title: "E"},
	}
	rows, _ := BuildRows(es)
	got := map[string]string{}
	for _, r := range rows {
		a := "-"
		if r.AltURL != nil {
			a = strings.TrimPrefix(*r.AltURL, "https://www.spurgeongems.org/sermon/")
		}
		got[r.ID] = a
	}
	want := map[string]string{"297": "chs297-298.pdf", "298": "chs297-298.pdf", "1451A": "chs1451A.pdf", "1451B": "chs1451B.pdf", "1331": "chs1331.pdf", "1331-2": "-"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

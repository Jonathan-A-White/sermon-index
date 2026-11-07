package main

import (
	"reflect"
	"testing"
)

func TestLogosToOSIS(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"bible.1.1", "Gen.1"},                             // Logos gives a chapter as book.chapter
		{"bible.66.9", "Rom.9"},                            // Logos's Romans is 66
		{"bible.65.27", "Acts.27"},                         // and Acts 65
		{"bible.61.1.1", "Matt.1.1"},                       // Matthew is the first of the New Testament
		{"bible.87.22.21", "Rev.22.21"},                    // and Revelation the last
		{"bible.19.51.1-19.51.19", "Ps.51.1-Ps.51.19"},     // a range of verses
		{"bible.1.1-1.3", "Gen.1-Gen.3"},                   // a range of chapters
		{"bible.87.4-87.5.14", "Rev.4-Rev.5.14"},           // a chapter to a verse
		{"bible.9.28.13-9.28.20", "1Sam.28.13-1Sam.28.20"}, // a numbered book
		{"bible.10.12.1-10.12.31", "2Sam.12.1-2Sam.12.31"}, // and its twin
		{"bible.64.1.35-64.1.42", "John.1.35-John.1.42"},   // John
		{"bible.22.1.1-22.7.13", "Song.1.1-Song.7.13"},     // Song of Solomon (22)
		{"bible.67.15.3-67.15.4", "1Cor.15.3-1Cor.15.4"},   // 1 Corinthians (67)
		{"bible.83.1.1-83.1.4", "1John.1.1-1John.1.4"},     // 1 John (83)
	}
	for _, c := range cases {
		got, err := LogosToOSIS(c.in)
		if err != nil || got != c.want {
			t.Errorf("LogosToOSIS(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
}

func TestLogosBookTable(t *testing.T) {
	// The Old Testament keeps the Protestant order at 1 to 39; Logos's New
	// Testament runs 61 (Matthew) to 87 (Revelation); 40 to 60 are not among
	// the sixty-six and are refused.
	want := map[int]string{
		1: "Gen", 19: "Ps", 22: "Song", 23: "Isa", 27: "Dan", 39: "Mal",
		61: "Matt", 62: "Mark", 63: "Luke", 64: "John", 65: "Acts", 66: "Rom",
		67: "1Cor", 68: "2Cor", 69: "Gal", 70: "Eph", 71: "Phil", 72: "Col",
		73: "1Thess", 74: "2Thess", 75: "1Tim", 76: "2Tim", 77: "Titus", 78: "Phlm",
		79: "Heb", 80: "Jas", 81: "1Pet", 82: "2Pet", 83: "1John", 84: "2John",
		85: "3John", 86: "Jude", 87: "Rev",
	}
	for n, osis := range want {
		got, ok := logosBook(n)
		if !ok || got != osis {
			t.Errorf("logosBook(%d) = %q, %v; want %q", n, got, ok, osis)
		}
	}
	for _, n := range []int{0, 40, 45, 60, 88, 100} {
		if got, ok := logosBook(n); ok {
			t.Errorf("logosBook(%d) = %q; want it refused", n, got)
		}
	}
	// Every one of the sixty-six is reached exactly once.
	seen := map[string]int{}
	for n := 1; n <= 100; n++ {
		if o, ok := logosBook(n); ok {
			seen[o]++
		}
	}
	if len(seen) != 66 {
		t.Errorf("%d books reached, want 66", len(seen))
	}
	for o, k := range seen {
		if k != 1 {
			t.Errorf("%s reached %d times", o, k)
		}
	}
}

func TestLogosToOSISRefuses(t *testing.T) {
	for _, in := range []string{"", "bible", "bible.66", "bible.45.3", "bible.66.5.99", "bible.66.x", "bible.66.9.1.1", "john.3.16", "bible.66.9-", "bible.66.9-1.2.3.4"} {
		if got, err := LogosToOSIS(in); err == nil {
			t.Errorf("LogosToOSIS(%q) = %q; want an error", in, got)
		}
	}
}

func TestLabelToOSIS(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"Romans 9", []string{"Rom.9"}},
		{"Jonah", []string{"Jonah"}},
		{"1 Peter", []string{"1Pet"}},
		{"John 3:16", []string{"John.3.16"}},
		{"John 3:1–13", []string{"John.3.1-John.3.13"}},
		{"John 3:1-13", []string{"John.3.1-John.3.13"}},
		{"Genesis 1–3", []string{"Gen.1-Gen.3"}},
		{"Psalm 23", []string{"Ps.23"}},
		{"Song of Solomon 1:1–7:13", []string{"Song.1.1-Song.7.13"}},
		{"Genesis 2; Exodus 31", []string{"Gen.2", "Exod.31"}},
		{"2 Samuel 11; 12:1–31; Psalm 51:1–19", []string{"2Sam.11", "2Sam.12.1-2Sam.12.31", "Ps.51.1-Ps.51.19"}},
		{"1 Corinthians 15:3–4, 15:20–28", []string{"1Cor.15.3-1Cor.15.4", "1Cor.15.20-1Cor.15.28"}},
		{"Acts 1:12–15; 2:1, 7–13", []string{"Acts.1.12-Acts.1.15", "Acts.2.1", "Acts.2.7-Acts.2.13"}},
		{"Acts 2:47; Revelation 4–5:14", []string{"Acts.2.47", "Rev.4-Rev.5.14"}},
		{"Job 1:1–2:13; Revelation 12:1–17", []string{"Job.1.1-Job.2.13", "Rev.12.1-Rev.12.17"}},
		{"Titus 2; Proverbs 31", []string{"Titus.2", "Prov.31"}},
		// A stray comma after the book, and an OCR l for a 1 in a number.
		{"Matthew, 5:20", []string{"Matt.5.20"}},
		{"John, 13:7", []string{"John.13.7"}},
		{"1 Kings, 18:2l", []string{"1Kgs.18.21"}},
		{"Exodus, 20:8", []string{"Exod.20.8"}},
		{"Psalm l9:1", []string{"Ps.19.1"}},
		{"Acts 1:12–15; 2:1, l7", []string{"Acts.1.12-Acts.1.15", "Acts.2.1", "Acts.2.17"}},
		// Old abbreviations, with a stray space or colon before the chapter
		// (Burroughs's "Math . 5:6", "Math : 5:4") and a point after the book.
		{"Math . 5:6", []string{"Matt.5.6"}},
		{"Math : 5:4", []string{"Matt.5.4"}},
		{"Math. 5:3", []string{"Matt.5.3"}},
		{"Matth. 5:3", []string{"Matt.5.3"}},
		{"Colos 3:1", []string{"Col.3.1"}},
		{"Colos. 3:1", []string{"Col.3.1"}},
		{"Hebr. 11:1", []string{"Heb.11.1"}},
		{"Hebr 11:1", []string{"Heb.11.1"}},
		{"Philippians . 4:11", []string{"Phil.4.11"}},
		// The ordinary abbreviations, and a leading St.
		{"Matt. 6:22", []string{"Matt.6.22"}},
		{"ROM. 8:12", []string{"Rom.8.12"}},
		{"ISA. 54:10", []string{"Isa.54.10"}},
		{"1 Cor. 15:35", []string{"1Cor.15.35"}},
		{"2 Chron. 7:14", []string{"2Chr.7.14"}},
		{"Ex. 12:3", []string{"Exod.12.3"}},
		{"Philip. 3:13", []string{"Phil.3.13"}},
		{"Jam 1:5", []string{"Jas.1.5"}},
		{"St. Matthew 5:10", []string{"Matt.5.10"}},
		{"ST. JOHN 13:18", []string{"John.13.18"}},
		{"Matt. 5:3; Luke 6:20", []string{"Matt.5.3", "Luke.6.20"}},
	}
	for _, c := range cases {
		got, err := LabelToOSIS(c.in)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("LabelToOSIS(%q) = %v, %v; want %v", c.in, got, err, c.want)
		}
	}
	for _, in := range []string{"", "Exodus 3 and 4", "Tobit 3", "Romans 99", "Romans 5:99", "Crusade July 4", "Romans x", "Mathematics 5:6", "Exam 3:4"} {
		if got, err := LabelToOSIS(in); err == nil {
			t.Errorf("LabelToOSIS(%q) = %v; want an error", in, got)
		}
	}
}

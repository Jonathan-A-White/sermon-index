package passage

import (
	"errors"
	"strings"
	"testing"
)

func TestTableHas66BooksInLampasOrder(t *testing.T) {
	if len(books) != 66 {
		t.Fatalf("books: %d, want 66", len(books))
	}
	for i, b := range books {
		if b.N != i+1 {
			t.Errorf("books[%d].N = %d, want %d", i, b.N, i+1)
		}
		if len(b.Last) == 0 {
			t.Errorf("%s has no chapters", b.OSIS)
		}
	}
	if books[0].Code != "gen" || books[44].Code != "rom" || books[65].Code != "rev" {
		t.Errorf("order: %s %s %s", books[0].Code, books[44].Code, books[65].Code)
	}
	// Spot checks against Lampas's chapter files (see books.go's header).
	for _, c := range []struct {
		book    string
		chapter int
		last    int
	}{
		{"Ps", 150, 6}, {"Rom", 5, 21}, {"Acts", 8, 40}, {"Mal", 4, 6}, {"Rom", 16, 24}, {"3John", 1, 14},
	} {
		b, ok := Lookup(c.book)
		if !ok {
			t.Fatalf("Lookup(%q) failed", c.book)
		}
		if got := b.Last[c.chapter-1]; got != c.last {
			t.Errorf("%s %d last verse = %d, want %d", c.book, c.chapter, got, c.last)
		}
	}
}

func TestEveryNameIsUnambiguous(t *testing.T) {
	seen := map[string]string{}
	for _, b := range books {
		for _, n := range []string{strings.ToLower(b.OSIS), b.Code} {
			if prev, ok := seen[n]; ok && prev != b.OSIS {
				t.Errorf("name %q is both %s and %s", n, prev, b.OSIS)
			}
			seen[n] = b.OSIS
		}
	}
}

func TestOrdinal(t *testing.T) {
	if got := Ordinal(45, 5, 8); got != 45005008 {
		t.Errorf("Ordinal(Rom 5:8) = %d", got)
	}
	if got := Ordinal(66, 22, 21); got != 66022021 {
		t.Errorf("Ordinal(Rev 22:21) = %d", got)
	}
}

func TestParse(t *testing.T) {
	for _, c := range []struct {
		ref    string
		osis   string
		lo, hi int
	}{
		{"Rom.5.8", "Rom.5.8", 45005008, 45005008},
		{"Rom.5.1-Rom.5.11", "Rom.5.1-Rom.5.11", 45005001, 45005011},
		{"rom.5.8", "Rom.5.8", 45005008, 45005008},                 // Lampas's code
		{"psa.23", "Ps.23", 19023001, 19023006},                    // a whole chapter
		{"Ps.23.1-Ps.24.2", "Ps.23.1-Ps.24.2", 19023001, 19024002}, // cross-chapter spans
		{"Rom.5-Rom.6", "Rom.5-Rom.6", 45005001, 45006023},
		{"Jude", "Jude", 65001001, 65001025}, // a whole book
		{"Rom.5.8-Rom.5.8", "Rom.5.8", 45005008, 45005008},
		{" Rom.5.8 ", "Rom.5.8", 45005008, 45005008},
	} {
		r, err := Parse(c.ref)
		if err != nil {
			t.Errorf("Parse(%q): %v", c.ref, err)
			continue
		}
		if r.OSIS != c.osis || r.Lo != c.lo || r.Hi != c.hi {
			t.Errorf("Parse(%q) = %+v, want %s [%d,%d]", c.ref, r, c.osis, c.lo, c.hi)
		}
	}
}

func TestParseCrossChapterContainsTheVersesBetween(t *testing.T) {
	r, err := Parse("Rom.5.12-Rom.6.4")
	if err != nil {
		t.Fatal(err)
	}
	mid, _ := Parse("Rom.5.21")
	if !(r.Lo <= mid.Lo && mid.Hi <= r.Hi) {
		t.Errorf("%+v does not contain %+v", r, mid)
	}
	after, _ := Parse("Rom.6.5")
	if after.Lo <= r.Hi {
		t.Errorf("%+v reaches %+v", r, after)
	}
}

func TestPG(t *testing.T) {
	r, _ := Parse("Rom.5.1-Rom.5.11")
	if got := r.PG(); got != "[45005001,45005012)" {
		t.Errorf("PG() = %s", got)
	}
}

func TestParseErrors(t *testing.T) {
	for _, c := range []struct{ ref, msg string }{
		{"Rom.5.99", "ref 'Rom.5.99': verse past the chapter"},
		{"Rom.17.1", "ref 'Rom.17.1': chapter past the book"},
		{"Ps.151", "ref 'Ps.151': chapter past the book"},
		{"Tob.1.1", "ref 'Tob.1.1': unknown book 'Tob'"},
		{"Ps.3.0", "ref 'Ps.3.0': verse 0 is not a verse (a superscription is not numbered)"},
		{"Rom.5.11-Rom.5.1", "ref 'Rom.5.11-Rom.5.1': the range ends before it starts"},
		{"Rom.5.1-11", "ref 'Rom.5.1-11': the end of a range is a whole reference (Rom.5.1-Rom.5.11)"},
		{"Rom 5:8", "ref 'Rom 5:8': not an OSIS reference (Book.Chapter.Verse)"},
		{"", "ref '': empty"},
		{"Rom.5.8.1", "ref 'Rom.5.8.1': not an OSIS reference (Book.Chapter.Verse)"},
		{"Rom.x.8", "ref 'Rom.x.8': not an OSIS reference (Book.Chapter.Verse)"},
		{"Rom.5.8-", "ref 'Rom.5.8-': not an OSIS reference (Book.Chapter.Verse)"},
		{"Rom.16.25", "ref 'Rom.16.25': verse past the chapter"},
	} {
		_, err := Parse(c.ref)
		if err == nil {
			t.Errorf("Parse(%q): no error, want %q", c.ref, c.msg)
			continue
		}
		var pe *Error
		if !errors.As(err, &pe) {
			t.Errorf("Parse(%q): error %T is not *Error", c.ref, err)
		}
		if err.Error() != c.msg {
			t.Errorf("Parse(%q) error = %q, want %q", c.ref, err.Error(), c.msg)
		}
	}
}

func TestVerses(t *testing.T) {
	for _, c := range []struct {
		ref  string
		want int
	}{{"Rom.5.8", 1}, {"Rom.5", 21}, {"Rom.5.12-Rom.6.4", 14}, {"Isa.53", 12}} {
		r, err := Parse(c.ref)
		if err != nil {
			t.Fatal(err)
		}
		if got := Verses(r.Lo, r.Hi); got != c.want {
			t.Errorf("Verses(%s) = %d, want %d", c.ref, got, c.want)
		}
	}
}

func TestCountVersesMergesOverlapsAndAdjoins(t *testing.T) {
	r := func(s string) Range {
		x, err := Parse(s)
		if err != nil {
			t.Fatal(err)
		}
		return x
	}
	// Rom.5.1-Rom.5.11 and Rom.5.8-Rom.5.21 overlap (21 verses); Rom.5.21 and Rom.6.1 are two.
	if got := CountVerses([]Range{r("Rom.5.8-Rom.5.21"), r("Rom.5.1-Rom.5.11"), r("Rom.5.8")}); got != 21 {
		t.Errorf("overlapping: %d, want 21", got)
	}
	if got := CountVerses([]Range{r("Rom.5.21"), r("Rom.6.1")}); got != 2 {
		t.Errorf("adjoining: %d, want 2", got)
	}
	if got := CountVerses(nil); got != 0 {
		t.Errorf("none: %d", got)
	}
}

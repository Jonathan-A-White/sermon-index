package passage

import (
	"regexp"
	"testing"
)

func TestBookOSIS(t *testing.T) {
	cases := map[string]string{
		"Genesis": "Gen", "Gen": "Gen", "Gen.": "Gen", "gen": "Gen",
		"Exodus": "Exod", "Ex.": "Exod", "Ex": "Exod", "Exod.": "Exod",
		"Psalm": "Ps", "Psalms": "Ps", "Ps.": "Ps", "Ps": "Ps", "Psa.": "Ps", "Psa": "Ps",
		"Song of Songs": "Song", "Song of Solomon": "Song",
		"1 Cor.": "1Cor", "1 Cor": "1Cor", "1 Corinthians": "1Cor",
		"2 Cor.": "2Cor", "2 Cor": "2Cor", "2  Cor.": "2Cor", "2 Corinthians": "2Cor",
		"Phil.": "Phil", "Phil": "Phil", "Philippians": "Phil", "Philem.": "Phlm", "Philemon": "Phlm",
		"Rev.": "Rev", "Rev": "Rev", "Revelation": "Rev",
		"Matt.": "Matt", "Mt": "Matt", "Rom.": "Rom", "Heb.": "Heb", "Jas.": "Jas", "1 Pet.": "1Pet", "3 Jn": "3John",
		"Isa.": "Isa", "Jer.": "Jer", "Dan.": "Dan", "Deut.": "Deut", "1 Sam.": "1Sam", "2 Kgs.": "2Kgs", "1 Chron.": "1Chr", "1 Thess.": "1Thess",
	}
	for in, want := range cases {
		if got, ok := BookOSIS(in); !ok || got != want {
			t.Errorf("BookOSIS(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "Hezekiah", "Cor.", "3 Cor.", "Morgan"} {
		if got, ok := BookOSIS(bad); ok {
			t.Errorf("BookOSIS(%q) = %q, want none", bad, got)
		}
	}
}

// Every full name in the table, and every OSIS name it maps to, is read, and
// the pattern matches exactly what BookOSIS reads.
func TestBookNamesAreAllReadAndMatched(t *testing.T) {
	re := regexp.MustCompile(`(?i)^` + BookNamePattern() + `$`)
	for _, b := range Books() {
		if o, ok := BookOSIS(b.Name); !ok || o != b.OSIS {
			t.Errorf("%s = %q, %v; want %q", b.Name, o, ok, b.OSIS)
		}
		if !re.MatchString(b.Name) {
			t.Errorf("pattern does not match %q", b.Name)
		}
	}
	for a, o := range bookAbbreviations {
		if _, ok := Lookup(o); !ok {
			t.Errorf("abbreviation %q names %q, which is no book", a, o)
		}
		if !re.MatchString(a) {
			t.Errorf("pattern does not match %q", a)
		}
	}
}

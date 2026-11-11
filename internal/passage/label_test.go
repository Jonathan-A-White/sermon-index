package passage

import "testing"

func TestVerseNumber(t *testing.T) {
	for in, want := range map[string]string{
		"10": "10", "10a": "10", "12b": "12", "7": "7", "10A": "10", "119d": "119",
	} {
		got, err := VerseNumber(in)
		if err != nil || got != want {
			t.Errorf("VerseNumber(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "a", "10ab", "10 a", "2a:4", "1000", "10-12", "+3", "10."} {
		if got, err := VerseNumber(bad); err == nil {
			t.Errorf("VerseNumber(%q) = %q; want it refused", bad, got)
		}
	}
}

func TestChapterNumber(t *testing.T) {
	for in, want := range map[string]string{"2": "2", "40": "40", "119": "119"} {
		got, err := ChapterNumber(in)
		if err != nil || got != want {
			t.Errorf("ChapterNumber(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	// A chapter has no half: its letter is refused, never dropped.
	for _, bad := range []string{"", "2a", "40b", "2a:4", "1000"} {
		if got, err := ChapterNumber(bad); err == nil {
			t.Errorf("ChapterNumber(%q) = %q; want it refused", bad, got)
		}
	}
}

func TestRangeOSIS(t *testing.T) {
	for _, c := range []struct{ c1, v1, x, y, want string }{
		{"4", "1", "5", "14", "Eph.4.1-Eph.5.14"},
		{"4", "", "5", "14", "Eph.4.1-Eph.5.14"}, // 'Ephesians 4-5:14'
		{"4", "", "5", "", "Eph.4-Eph.5"},
		{"4", "", "", "", "Eph.4"},
		{"4", "1", "", "", "Eph.4.1"},
		{"4", "1", "10a", "", "Eph.4.1-Eph.4.10"},
		{"4", "1a", "5", "14b", "Eph.4.1-Eph.5.14"},
	} {
		got, err := RangeOSIS("Eph", c.c1, c.v1, c.x, c.y)
		if err != nil || got != c.want {
			t.Errorf("RangeOSIS(%q, %q, %q, %q) = %q, %v; want %q", c.c1, c.v1, c.x, c.y, got, err, c.want)
		}
	}
	// A letter on a chapter, or a verse after a dash with no second chapter
	// number in the right place, is refused, never read short.
	for _, c := range []struct{ c1, v1, x, y string }{
		{"4", "", "5a", ""},
		{"4", "", "5a", "14"},
		{"4", "1", "5a", "14"},
		{"4", "", "5", "14ab"},
		{"4", "", "5", "x"},
		{"4", "1", "5", "x"},
		{"4", "x", "", ""},
	} {
		if got, err := RangeOSIS("Eph", c.c1, c.v1, c.x, c.y); err == nil {
			t.Errorf("RangeOSIS(%q, %q, %q, %q) = %q; want it refused", c.c1, c.v1, c.x, c.y, got)
		}
	}
}

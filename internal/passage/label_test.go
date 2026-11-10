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

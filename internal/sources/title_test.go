package sources

import "testing"

func TestSameTitleReadsTheRowTitlesPartBeforeTheBar(t *testing.T) {
	const row = "Why the Mission Cannot Wait | Revelation 5:9-10 | Pastor Phil Morgan"
	for _, tc := range []struct {
		row, mail string
		want      bool
	}{
		{row, "Why the Mission Cannot Wait", true},
		{row, "why the mission cannot wait", true},
		{row, "Why The Mission, Cannot Wait!", true},
		{row, "How to Spot a Fable", false},
		{row, "Revelation 5:9-10", false},
		{"How to spot a Fable", "How to Spot a Fable", true},
		{"Title | Passage", "Title | Passage", true},
		{row, "", false},
		{"", "", false},
	} {
		if got := SameTitle(tc.row, tc.mail); got != tc.want {
			t.Errorf("SameTitle(%q, %q) = %v, want %v", tc.row, tc.mail, got, tc.want)
		}
	}
}

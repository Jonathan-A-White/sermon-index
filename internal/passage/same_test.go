package passage

import "testing"

func TestSamePlace(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"Rev.5", "Rev.5.9-Rev.5.10", true},
		{"Rev.5.9-Rev.5.10", "Rev.5", true},
		{"Rev.5.9-Rev.5.10", "Rev.5.9-Rev.5.10", true},
		{"Rev.5", "Rev.6", false},
		{"Rev.5.9-Rev.5.10", "Rev.6", false},
		{"Rev.5", "Matt.5", false},
		{"Rev.5", "nonsense", false},
		{"", "Rev.5", false},
	} {
		if got := SamePlace(c.a, c.b); got != c.want {
			t.Errorf("SamePlace(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

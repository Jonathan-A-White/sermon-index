package passage

import (
	"math"
	"sort"
)

// Verses counts the verses from the ordinal lo to the ordinal hi, both
// inclusive, by the book table (so Rom.5.12-Rom.6.4 is 14: Romans 5 ends
// at verse 21). An ordinal outside the table counts as its difference.
func Verses(lo, hi int) int {
	b, c, v := lo/1000000, lo/1000%1000, lo%1000
	eb, ec, ev := hi/1000000, hi/1000%1000, hi%1000
	valid := func(b, c int) bool { return b >= 1 && b <= len(books) && c >= 1 && c <= len(books[b-1].Last) }
	if hi < lo || !valid(b, c) || !valid(eb, ec) {
		return int(math.Abs(float64(hi-lo))) + 1
	}
	n := 0
	for b != eb || c != ec {
		n += books[b-1].Last[c-1] - v + 1
		c, v = c+1, 1
		if c > len(books[b-1].Last) {
			b, c = b+1, 1
		}
	}
	return n + ev - v + 1
}

// CountVerses counts the verses the ranges cover together: a verse in two
// ranges is counted once. No ranges cover 0.
func CountVerses(rs []Range) int {
	if len(rs) == 0 {
		return 0
	}
	sorted := append([]Range(nil), rs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Lo < sorted[j].Lo })
	n := 0
	lo, hi := sorted[0].Lo, sorted[0].Hi
	for _, r := range sorted[1:] {
		if r.Lo <= hi {
			if r.Hi > hi {
				hi = r.Hi
			}
			continue
		}
		n += Verses(lo, hi)
		lo, hi = r.Lo, r.Hi
	}
	return n + Verses(lo, hi)
}

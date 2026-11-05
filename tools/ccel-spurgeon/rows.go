package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Jonathan-A-White/sermon-index/internal/passage"
	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

const preacher = "Charles H. Spurgeon"

// Report is what a run found: the counts the tool prints and the numbers
// REPORT.md lists.
type Report struct {
	Entries  int // sermon entries read
	Numbers  int // distinct sermon numbers
	Rows     int // rows written
	WithText int // rows whose preaching text resolves
	Romans5  int // rows whose preaching text lies in Romans 5
	Dated    int // rows with a date
	Exposits int // rows of kind exposition

	TwoTexts   []string // numbers the ThML gives two or more preaching texts for
	NoText     []string // numbers with no preaching text, not an exposition: "385 Title"
	Unresolved []string // numbers whose scripCom gave something that does not resolve: "416 ''"
	Combined   []string // combined entries: "385-388"
	Repeated   []string // numbers CCEL gives more than one entry: "1331: ids 1331, 1331-2"
	Refused    []string // tagged references Lampas's book table refuses, left out: "1448: ref 'Eph.4.82': verse past the chapter"
}

// BuildRows turns entries into rows, one a sermon number, in number order.
func BuildRows(entries []Entry) ([]sources.Row, *Report) {
	rep := &Report{Entries: len(entries)}
	rom5, _ := passage.Parse("Rom.5")

	type item struct {
		n    int
		ord  int // the entry's position, to keep a repeat in file order
		suf  string
		row  sources.Row
		text *passage.Range
	}
	var items []item
	count := map[int]int{} // entries a number has
	for _, e := range entries {
		for _, n := range e.Numbers {
			count[n]++
		}
	}
	nth := map[int]int{}
	for ord, e := range entries {
		// The references the checker takes: the preaching text, then the
		// others (a second preaching text, every other tagged reference).
		var texts []string
		var text *passage.Range
		for _, t := range e.Texts {
			if rg, err := passage.Parse(t); err == nil {
				texts = append(texts, t)
				if text == nil {
					rg := rg
					text = &rg
				}
			} else {
				rep.Refused = append(rep.Refused, fmt.Sprintf("%s: %v", numberLabel(e.Numbers), err))
			}
		}
		var cited []string
		for _, c := range append(append([]string{}, texts[min(1, len(texts)):]...), e.Cited...) {
			if len(texts) > 0 && c == texts[0] {
				continue
			}
			if _, err := passage.Parse(c); err != nil {
				if !contains(e.Texts, c) {
					rep.Refused = append(rep.Refused, fmt.Sprintf("%s: %v", numberLabel(e.Numbers), err))
				}
				continue
			}
			cited = appendOnce(cited, c)
		}
		for _, n := range e.Numbers {
			nth[n]++
			id := strconv.Itoa(n)
			if count[n] > 1 {
				switch {
				case e.Suffix != "":
					id += e.Suffix // 1451A
				case nth[n] > 1:
					id += "-" + strconv.Itoa(nth[n]) // 1331-2
				}
			}
			r := sources.Row{
				ID:       id,
				Title:    e.Title,
				URL:      PageURL(e.Vol, e.DivID),
				Preacher: ptr(preacher),
			}
			var notes []string
			if len(e.Numbers) > 1 {
				notes = append(notes, fmt.Sprintf("A combined entry: CCEL has one page for the numbers %d to %d, so each has a row with the same url.", e.Numbers[0], e.Numbers[len(e.Numbers)-1]))
			}
			if e.Corrected != "" {
				notes = append(notes, fmt.Sprintf("CCEL's heading reads %s, a slip: the entry stands between %d and %d in volume %d.", e.Corrected, n-1, n+1, e.Vol))
			}
			if count[n] > 1 {
				notes = append(notes, fmt.Sprintf("CCEL has %d entries numbered %d; this is the %s (volume %d).", count[n], n, ordinal(nth[n]), e.Vol))
			}
			if text != nil {
				r.Ref = ptr(texts[0])
			}
			switch {
			case e.Exposition:
				r.Kind = ptr(sources.RowExposition)
			case count[n] > 1 && e.Suffix == "":
				// Two entries of one number, neither lettered: Gems' pdf is
				// the first's, and the second has none.
				if nth[n] == 1 {
					r.AltURL = ptr(GemsURL(strconv.Itoa(n)))
				}
			case gemsRanges[n] != "":
				r.AltURL = ptr(GemsURL(gemsRanges[n]))
			default:
				r.AltURL = ptr(GemsURL(strconv.Itoa(n) + e.Suffix))
			}
			if len(texts) > 1 {
				notes = append(notes, "CCEL tags more than one preaching text; the first is the ref and the others are cited.")
			}
			if text == nil && !e.Exposition {
				if len(e.TextAttrs) > 0 {
					notes = append(notes, fmt.Sprintf("CCEL's preaching text %q has no usable reference, so there is no ref.", e.TextAttrs[0]))
				} else {
					notes = append(notes, "CCEL tags no preaching text (a meeting, an address or a dedication), so there is no ref.")
				}
			}
			r.Cited = cited
			if e.Date != "" {
				r.Date = ptr(e.Date)
			}
			if len(notes) > 0 {
				r.Notes = ptr(strings.Join(notes, " "))
			}
			items = append(items, item{n: n, ord: ord, suf: e.Suffix, row: r, text: text})
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].n != items[j].n {
			return items[i].n < items[j].n
		}
		return items[i].ord < items[j].ord
	})

	rows := make([]sources.Row, len(items))
	numbers := map[int]bool{}
	for i, it := range items {
		rows[i] = it.row
		numbers[it.n] = true
		if it.text != nil {
			rep.WithText++
			if it.text.Lo <= rom5.Hi && rom5.Lo <= it.text.Hi {
				rep.Romans5++
			}
		}
		if it.row.Date != nil {
			rep.Dated++
		}
		if it.row.Kind != nil {
			rep.Exposits++
		}
	}
	rep.Rows = len(rows)
	rep.Numbers = len(numbers)

	done := map[int]bool{}
	for _, e := range entries {
		first := e.Numbers[0]
		num := numberLabel(e.Numbers)
		if len(e.Texts) > 1 {
			rep.TwoTexts = append(rep.TwoTexts, num)
		}
		if len(e.Numbers) > 1 {
			rep.Combined = append(rep.Combined, num)
		}
		hasText := false
		for _, t := range e.Texts {
			if _, err := passage.Parse(t); err == nil {
				hasText = true
			}
		}
		if !hasText && !e.Exposition {
			rep.NoText = append(rep.NoText, fmt.Sprintf("%s %s", num, e.Title))
		}
		if !hasText && len(e.TextAttrs) > 0 && !e.Exposition {
			rep.Unresolved = append(rep.Unresolved, fmt.Sprintf("%s %q", num, strings.Join(e.TextAttrs, "; ")))
		}
		if count[first] > 1 && !done[first] {
			done[first] = true
			var ids []string
			for _, it := range items {
				if it.n == first {
					ids = append(ids, it.row.ID)
				}
			}
			rep.Repeated = append(rep.Repeated, fmt.Sprintf("%d: ids %s", first, strings.Join(ids, ", ")))
		}
	}
	return rows, rep
}

func numberLabel(ns []int) string {
	if len(ns) == 1 {
		return strconv.Itoa(ns[0])
	}
	return fmt.Sprintf("%d-%d", ns[0], ns[len(ns)-1])
}

func ordinal(n int) string {
	switch n {
	case 1:
		return "first"
	case 2:
		return "second"
	case 3:
		return "third"
	}
	return strconv.Itoa(n) + "th"
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func ptr[T any](v T) *T { return &v }

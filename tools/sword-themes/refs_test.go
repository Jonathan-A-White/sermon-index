package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseGroup(t *testing.T) {
	cases := []struct {
		group string
		want  []string
	}{
		{"Ro 5:8", []string{"Rom.5.8"}},
		// A later item without a book takes the earlier item's; a comma
		// list is one reference a verse.
		{"Ex 4:14; 5:20; 6:20; 7:1,7,12", []string{"Exod.4.14", "Exod.5.20", "Exod.6.20", "Exod.7.1", "Exod.7.7", "Exod.7.12"}},
		{"Le 10:6; Nu 12:1; 1Co 13:1", []string{"Lev.10.6", "Num.12.1", "1Cor.13.1"}},
		{"1 Co 13:1", []string{"1Cor.13.1"}},
		{"Ro 9:24-26", []string{"Rom.9.24-Rom.9.26"}},
		{"Ro 5:8-8", []string{"Rom.5.8"}},
		{"Isa 52:13-53:12", []string{"Isa.52.13-Isa.53.12"}},
		{"Ps 23; Isa 40-42", []string{"Ps.23", "Isa.40-Isa.42"}},
		{"Ps 19:8-10,14; 119:20", []string{"Ps.19.8-Ps.19.10", "Ps.19.14", "Ps.119.20"}},
		// A book of one chapter: a number alone is a verse.
		{"Jude 21; Phm 5-7; 3Jo 1:4", []string{"Jude.1.21", "Phlm.1.5-Phlm.1.7", "3John.1.4"}},
		// Spellings the modules use: Jud is Judges, Joe Joel, He Hebrews.
		{"Jud 4:2; Joe 2:28; He 11:1; Mr 12:30; Re 1:1; Php 1:9; Mi 5:2", []string{"Judg.4.2", "Joel.2.28", "Heb.11.1", "Mark.12.30", "Rev.1.1", "Phil.1.9", "Mic.5.2"}},
		{"2KI 2:1; EX 3:1; MT 1:1", []string{"2Kgs.2.1", "Exod.3.1", "Matt.1.1"}},
		// A group wrapped over lines in the module has a newline or spaces.
		{"Le 16:30,34;\n  Heb 9:7,12,28; Le 17:11", []string{"Lev.16.30", "Lev.16.34", "Heb.9.7", "Heb.9.12", "Heb.9.28", "Lev.17.11"}},
		// Some groups separate items with a comma before the next book.
		{"Re 21:1-27, 22:1-21, Joh 14:1-31", []string{"Rev.21.1-Rev.21.27", "Rev.22.1-Rev.22.21", "John.14.1-John.14.31"}},
		{"2Ki 9:25, Isa 13:1", []string{"2Kgs.9.25", "Isa.13.1"}},
		// Slips of the keying: a trailing comma, a comma after the colon.
		{"2Sa 21:2,; Jos 9:3-16", []string{"2Sam.21.2", "Josh.9.3-Josh.9.16"}},
		{"De 28:,34; 1Sa 2:,12,17", []string{"Deut.28.34", "1Sam.2.12", "1Sam.2.17"}},
		// A range from one book into the next.
		{"2Sa 2:4-1Ki 2:11; 1Ch 11:1-29:30", []string{"2Sam.2.4-1Kgs.2.11", "1Chr.11.1-1Chr.29.30"}},
		// "Co" in Thompson's worldliness topic is Colossians 3:2.
		{"Co 3:2", []string{"Col.3.2"}},
		{"", nil},
	}
	for _, c := range cases {
		got, bad := ParseGroup(c.group)
		if len(bad) != 0 {
			t.Errorf("%q: unexpected bad items %v", c.group, bad)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q:\n got %v\nwant %v", c.group, got, c.want)
		}
	}
}

func TestParseGroupBad(t *testing.T) {
	cases := []struct {
		group    string
		wantRefs []string
		wantBad  string // a substring of the one bad item's text and reason
	}{
		{"Xx 1:1", nil, "unknown book 'Xx'"},
		{"Ro 5:99", nil, "verse past the chapter"},
		{"Ro 5:8; Ro 99:1", []string{"Rom.5.8"}, "chapter past the book"},
		{"(marg)", nil, "(marg)"},
		{"Ro 5:9-8", nil, "the range ends before it starts"},
		// An item that has no book and no earlier item to take it from.
		{"5:8", nil, "no book"},
	}
	for _, c := range cases {
		got, bad := ParseGroup(c.group)
		if !reflect.DeepEqual(got, c.wantRefs) {
			t.Errorf("%q: refs %v, want %v", c.group, got, c.wantRefs)
		}
		if len(bad) != 1 || !strings.Contains(bad[0], c.wantBad) {
			t.Errorf("%q: bad %v, want one containing %q", c.group, bad, c.wantBad)
		}
	}
}

func TestEveryAbbreviationNamesABook(t *testing.T) {
	for abbr, osis := range bookAbbr {
		if _, err := ParseGroup(abbr + " 1:1"); len(err) != 0 && osis != "Jude" {
			// Jude, Obad, Phlm, 2John, 3John have one chapter: 1:1 is fine too.
			t.Errorf("%s (%s): %v", abbr, osis, err)
		}
	}
}

package passage

import (
	"regexp"
	"sort"
	"strings"
)

// bookAbbreviations are the short forms of a book's name that a church's mail
// or a message is written with, each with the OSIS name it stands for. A key is
// lower case and has no final point, so "2 Cor." and "2 Cor" are one.
var bookAbbreviations = map[string]string{
	"gen": "Gen",
	"ex":  "Exod", "exod": "Exod", "exo": "Exod",
	"lev": "Lev",
	"num": "Num", "numb": "Num",
	"deut": "Deut", "deu": "Deut",
	"josh": "Josh", "jos": "Josh",
	"judg": "Judg", "jdg": "Judg",
	"1 sam": "1Sam", "1 sa": "1Sam", "2 sam": "2Sam", "2 sa": "2Sam",
	"1 kgs": "1Kgs", "1 ki": "1Kgs", "1 kin": "1Kgs", "2 kgs": "2Kgs", "2 ki": "2Kgs", "2 kin": "2Kgs",
	"1 chron": "1Chr", "1 chr": "1Chr", "1 ch": "1Chr", "2 chron": "2Chr", "2 chr": "2Chr", "2 ch": "2Chr",
	"neh":  "Neh",
	"esth": "Esth", "est": "Esth",
	"ps": "Ps", "psa": "Ps", "pss": "Ps", "psalm": "Ps", "psalms": "Ps",
	"prov": "Prov", "pro": "Prov", "prv": "Prov",
	"eccl": "Eccl", "eccles": "Eccl", "ecc": "Eccl",
	"song": "Song", "song of songs": "Song", "song of sol": "Song", "sos": "Song",
	"isa":  "Isa",
	"jer":  "Jer",
	"lam":  "Lam",
	"ezek": "Ezek", "eze": "Ezek",
	"dan":  "Dan",
	"hos":  "Hos",
	"obad": "Obad",
	"mic":  "Mic",
	"nah":  "Nah",
	"hab":  "Hab",
	"zeph": "Zeph", "zep": "Zeph",
	"hag":  "Hag",
	"zech": "Zech", "zec": "Zech",
	"mal":  "Mal",
	"matt": "Matt", "mat": "Matt", "mt": "Matt",
	"mk": "Mark", "mrk": "Mark",
	"lk": "Luke", "luk": "Luke",
	"jn": "John", "joh": "John",
	"rom":   "Rom",
	"1 cor": "1Cor", "1 co": "1Cor", "2 cor": "2Cor", "2 co": "2Cor",
	"gal": "Gal",
	"eph": "Eph", "ephes": "Eph",
	"phil": "Phil", "php": "Phil",
	"col":     "Col",
	"1 thess": "1Thess", "1 thes": "1Thess", "1 th": "1Thess", "2 thess": "2Thess", "2 thes": "2Thess", "2 th": "2Thess",
	"1 tim": "1Tim", "1 ti": "1Tim", "2 tim": "2Tim", "2 ti": "2Tim",
	"tit":    "Titus",
	"philem": "Phlm", "phlm": "Phlm", "phm": "Phlm",
	"heb": "Heb",
	"jas": "Jas", "jam": "Jas",
	"1 pet": "1Pet", "1 pe": "1Pet", "1 pt": "1Pet", "2 pet": "2Pet", "2 pe": "2Pet", "2 pt": "2Pet",
	"1 jn": "1John", "1 jo": "1John", "2 jn": "2John", "2 jo": "2John", "3 jn": "3John", "3 jo": "3John",
	"rev": "Rev", "revelations": "Rev",
}

// bookNames maps every spelling BookOSIS reads, in lower case with single
// spaces and no final point, to its OSIS name.
var bookNames = func() map[string]string {
	m := map[string]string{"psalm": "Ps", "song of songs": "Song"}
	for _, b := range books {
		m[strings.ToLower(b.Name)] = b.OSIS
	}
	for a, o := range bookAbbreviations {
		m[a] = o
	}
	return m
}()

// BookOSIS reads a book's name as a person writes it: the full name from the
// book table ("2 Corinthians"), "Psalm", "Song of Songs", or a usual
// abbreviation ("2 Cor", "Gen", "Ps", "Phil"), with or without a final point,
// in any case, and returns its OSIS name ("2Cor").
func BookOSIS(name string) (string, bool) {
	s := strings.ToLower(strings.Join(strings.Fields(name), " "))
	s = strings.TrimSpace(strings.TrimSuffix(s, "."))
	o, ok := bookNames[s]
	return o, ok
}

// BookNamePattern is a regular expression (without a capture group) that
// matches any spelling BookOSIS reads, longest first, with a space in a name
// matching any run of white space. The final point of an abbreviation is not
// part of it.
func BookNamePattern() string {
	list := make([]string, 0, len(bookNames))
	for n := range bookNames {
		list = append(list, n)
	}
	sort.Slice(list, func(i, j int) bool {
		if len(list[i]) != len(list[j]) {
			return len(list[i]) > len(list[j])
		}
		return list[i] < list[j]
	})
	for i, n := range list {
		list[i] = strings.ReplaceAll(regexp.QuoteMeta(n), `\ `, `\s+`)
	}
	return "(?:" + strings.Join(list, "|") + ")"
}

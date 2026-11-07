package main

import (
	"regexp"
	"strings"
)

// Calvin's Sermões sobre Eleição e Reprovação print their texts in Portuguese:
// "I. Primeiro Sermão: Gênesis 25.12–22", chapter and verse parted by a full
// stop and verses by a comma. PtLabel writes such a passage as the book table
// reads it ("Genesis 25:12–22"); the tool's other readers need no change.

// ptBooks are the Portuguese names of the sixty-six books (Almeida's, and the
// shorter forms a title may use), lower case with the accents taken off, as
// the English name the book table has. A numbered book is "1 samuel" (the
// number first).
var ptBooks = map[string]string{
	"genesis": "Genesis", "exodo": "Exodus", "levitico": "Leviticus", "numeros": "Numbers",
	"deuteronomio": "Deuteronomy", "josue": "Joshua", "juizes": "Judges", "rute": "Ruth",
	"1 samuel": "1 Samuel", "2 samuel": "2 Samuel", "1 reis": "1 Kings", "2 reis": "2 Kings",
	"1 cronicas": "1 Chronicles", "2 cronicas": "2 Chronicles", "esdras": "Ezra", "neemias": "Nehemiah",
	"ester": "Esther", "jo": "Job", "salmos": "Psalms", "salmo": "Psalms", "proverbios": "Proverbs",
	"eclesiastes": "Ecclesiastes", "cantares": "Song of Solomon", "canticos": "Song of Solomon",
	"canticos dos canticos": "Song of Solomon", "cantares de salomao": "Song of Solomon",
	"isaias": "Isaiah", "jeremias": "Jeremiah", "lamentacoes": "Lamentations", "ezequiel": "Ezekiel",
	"daniel": "Daniel", "oseias": "Hosea", "joel": "Joel", "amos": "Amos", "obadias": "Obadiah",
	"jonas": "Jonah", "miqueias": "Micah", "naum": "Nahum", "habacuque": "Habakkuk",
	"sofonias": "Zephaniah", "ageu": "Haggai", "zacarias": "Zechariah", "malaquias": "Malachi",
	"mateus": "Matthew", "marcos": "Mark", "lucas": "Luke", "joao": "John", "atos": "Acts",
	"atos dos apostolos": "Acts", "romanos": "Romans", "1 corintios": "1 Corinthians",
	"2 corintios": "2 Corinthians", "galatas": "Galatians", "efesios": "Ephesians",
	"filipenses": "Philippians", "colossenses": "Colossians", "1 tessalonicenses": "1 Thessalonians",
	"2 tessalonicenses": "2 Thessalonians", "1 timoteo": "1 Timothy", "2 timoteo": "2 Timothy",
	"tito": "Titus", "filemom": "Philemon", "filemon": "Philemon", "hebreus": "Hebrews",
	"tiago": "James", "1 pedro": "1 Peter", "2 pedro": "2 Peter", "1 joao": "1 John",
	"2 joao": "2 John", "3 joao": "3 John", "judas": "Jude", "apocalipse": "Revelation",
}

var accentFolder = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

// foldAccents is s in lower case with the Portuguese accents off.
func foldAccents(s string) string { return accentFolder.Replace(strings.ToLower(s)) }

// ptBookRE: the book at the head of a piece, then the numbers after it.
var ptBookRE = regexp.MustCompile(`^(\s*(?:[123]\s?)?[^\d;,]*?)\s*(\d.*)?$`)

var ptChapterVerseRE = regexp.MustCompile(`(\d+)\.(\d+)`)

// PtLabel writes a passage printed in Portuguese as the book table reads it:
// the book's name in English and a colon between chapter and verse
// ("Gênesis 25.12–22" is "Genesis 25:12–22", "Salmos 119.1–8; Romanos 8.28"
// is "Psalms 119:1–8; Romans 8:28"). A piece whose book is not one of the
// sixty-six is left as it is, which the book table then refuses.
func PtLabel(label string) string {
	pieces := strings.Split(label, ";")
	for i, piece := range pieces {
		lead := piece[:len(piece)-len(strings.TrimLeft(piece, " "))]
		m := ptBookRE.FindStringSubmatch(strings.TrimSpace(piece))
		if m == nil {
			continue
		}
		name, nums := strings.TrimSpace(m[1]), m[2]
		if en, ok := ptBooks[strings.Join(strings.Fields(foldAccents(name)), " ")]; ok {
			name = en
		}
		if nums != "" {
			name += " " + ptChapterVerseRE.ReplaceAllString(nums, "$1:$2")
		}
		pieces[i] = lead + name
	}
	return strings.Join(pieces, ";")
}

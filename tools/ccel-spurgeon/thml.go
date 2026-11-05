package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// What is read from CCEL's ThML is facts only: the entry's number, title and
// section id, the references it tags and a date from its head. No paragraph of
// a sermon is kept; the words are read to be thrown away.

// Entry is one sermon entry (a div1) of a volume.
type Entry struct {
	Vol        int      // the volume, 1 to 63
	DivID      string   // the div1's id, the section id of CCEL's page
	Numbers    []int    // the sermon numbers it covers: one, or more for a combined entry (385-388)
	Suffix     string   // "A" or "B" where CCEL numbers the halves of a number (1451A)
	Title      string   // the title without "Sermon N."
	Texts      []string // the preaching texts, OSIS ("Rom.5.8"), in the order given; the resolvable ones only
	TextAttrs  []string // the passage attribute of every scripCom type="Sermon" the entry has, resolvable or not
	Cited      []string // every tagged reference of the entry (the preaching text too), in order of first appearance, none repeated
	Date       string   // from the head of the sermon, "1856-11-23", "1856-11" or ""
	Exposition bool     // the title says it is an exposition
	Corrected  string   // what the heading read, where CCEL's number is a slip and the entry is given the number its place shows
}

// slips are headings whose number is a typing slip in CCEL's ThML, read from
// the entry's place: volume 12 has an entry headed "6823" between 682 and 684
// (dated between theirs), where 683 has none.
var slips = []struct {
	vol     int
	divID   string
	printed int
	number  int
}{
	{12, "xvi", 6823, 683},
}

// headParagraphs is how many paragraphs from the start of an entry are read
// for the date it was preached.
const headParagraphs = 14

// ParseVolume reads one volume's ThML and returns its sermon entries in the
// file's order. The last division of a volume, the indexes, is not a sermon.
func ParseVolume(r io.Reader, vol int) ([]Entry, error) {
	dec := xml.NewDecoder(r)
	dec.Strict = false
	dec.Entity = xml.HTMLEntity

	var (
		out   []Entry
		cur   *Entry
		head  strings.Builder
		paras int
		inP   bool
	)
	finish := func() {
		if cur != nil {
			cur.Date = headDate(head.String())
			if hi := combinedTo(head.String(), cur.Numbers[0]); len(cur.Numbers) == 1 && hi > cur.Numbers[0] {
				for n := cur.Numbers[0] + 1; n <= hi; n++ {
					cur.Numbers = append(cur.Numbers, n)
				}
			}
			for _, sl := range slips {
				if sl.vol == cur.Vol && sl.divID == cur.DivID && len(cur.Numbers) == 1 && cur.Numbers[0] == sl.printed {
					cur.Numbers[0] = sl.number
					cur.Corrected = strconv.Itoa(sl.printed)
				}
			}
			out = append(out, *cur)
		}
		cur = nil
	}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("volume %d: %w", vol, err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "div1":
				finish()
				nums, suffix, title, ok := ParseHeading(attr(t, "title"))
				if !ok {
					break // the indexes, or a pastoral letter: not a sermon
				}
				cur = &Entry{Vol: vol, DivID: attr(t, "id"), Numbers: nums, Suffix: suffix, Title: title}
				cur.Exposition = strings.HasPrefix(strings.ToLower(title), "exposition")
				head.Reset()
				paras, inP = 0, false
			case "scripCom":
				if cur == nil || attr(t, "type") != "Sermon" {
					break
				}
				cur.TextAttrs = append(cur.TextAttrs, strings.TrimSpace(attr(t, "passage")))
				for _, o := range osisOf(t) {
					cur.Texts = appendOnce(cur.Texts, o)
				}
			case "scripRef":
				if cur == nil {
					break
				}
				for _, o := range osisOf(t) {
					cur.Cited = appendOnce(cur.Cited, o)
				}
			case "p", "h1":
				if cur != nil && paras < headParagraphs {
					paras++
					inP = true
					head.WriteString("\n")
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "div1":
				finish()
			case "p", "h1":
				inP = false
			}
		case xml.CharData:
			if cur != nil && inP {
				head.Write(t)
			}
		}
	}
	finish()

	return out, nil
}

func attr(t xml.StartElement, name string) string {
	for _, a := range t.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func appendOnce(list []string, s string) []string {
	for _, x := range list {
		if x == s {
			return list
		}
	}
	return append(list, s)
}

// osisOf is the OSIS references a scripRef or scripCom carries: its osisRef
// without "Bible:" (a field of several verses gives them separated by
// spaces), else its machine reading (parsed). None when it has neither.
func osisOf(t xml.StartElement) []string {
	if f := strings.Fields(attr(t, "osisRef")); len(f) > 0 {
		for i := range f {
			f[i] = strings.TrimPrefix(f[i], "Bible:")
		}
		return f
	}
	if o := osisFromParsed(attr(t, "parsed")); o != "" {
		return []string{o}
	}
	return nil
}

// osisFromParsed turns ThML's "|Rom|5|1|6|4" (book, chapter, verse, end
// chapter, end verse; 0 for none) into "Rom.5.1-Rom.6.4".
func osisFromParsed(p string) string {
	f := strings.Split(p, "|")
	if len(f) != 6 || f[0] != "" {
		return ""
	}
	book := f[1]
	n := make([]int, 4)
	for i := range n {
		v, err := strconv.Atoi(f[2+i])
		if err != nil || v < 0 {
			return ""
		}
		n[i] = v
	}
	ch, v, ech, ev := n[0], n[1], n[2], n[3]
	if book == "" {
		return ""
	}
	if ch == 0 && v == 0 {
		// A whole book ("|Jude|0|0|0|0"), unless an end is given without a start.
		if ech == 0 && ev == 0 {
			return book
		}
		return ""
	}
	start := book + "." + strconv.Itoa(ch)
	if v > 0 {
		start += "." + strconv.Itoa(v)
	}
	if ech == 0 && ev == 0 {
		return start
	}
	if ech == 0 {
		ech = ch
	}
	end := book + "." + strconv.Itoa(ech)
	if ev > 0 {
		end += "." + strconv.Itoa(ev)
	}
	return start + "-" + end
}

var headingRE = regexp.MustCompile(`^Sermon\s+([0-9][0-9A-Za-z.\-]*?)\.\s+(.*)$`)
var numberRE = regexp.MustCompile(`^([0-9]+)(?:-([0-9]+))?([A-Z]?)(?:\.htm)?$`)

// ParseHeading reads a div1 title ("Sermon 104. Love's Commendation") into
// the numbers it covers, a letter suffix (1451A), and the title. ok is false
// for the indexes and for an entry that has no number.
func ParseHeading(h string) (nums []int, suffix, title string, ok bool) {
	m := headingRE.FindStringSubmatch(strings.TrimSpace(h))
	if m == nil {
		return nil, "", "", false
	}
	n := numberRE.FindStringSubmatch(m[1])
	if n == nil {
		return nil, "", "", false
	}
	lo, _ := strconv.Atoi(n[1])
	hi := lo
	if n[2] != "" {
		hi, _ = strconv.Atoi(n[2])
		// A short end: "297-8" is 297 to 298.
		if hi < lo && len(n[2]) < len(n[1]) {
			hi, _ = strconv.Atoi(n[1][:len(n[1])-len(n[2])] + n[2])
		}
		if hi < lo || hi-lo > 10 {
			return nil, "", "", false
		}
	}
	for i := lo; i <= hi; i++ {
		nums = append(nums, i)
	}
	return nums, n[3], strings.TrimSpace(m[2]), true
}

var nosRE = regexp.MustCompile(`\bNos\.\s*(\d+)\s*[-\x{2013}\x{2014}]\s*(\d+)\b`)

// combinedTo finds, in the head of an entry numbered lo, the "Nos. 385-88."
// that says the entry covers several numbers, and returns the last of them
// (0 when there is none).
func combinedTo(head string, lo int) int {
	m := nosRE.FindStringSubmatch(head)
	if m == nil {
		return 0
	}
	first, _ := strconv.Atoi(m[1])
	if first != lo {
		return 0
	}
	hi, _ := strconv.Atoi(m[2])
	if hi < first && len(m[2]) < len(m[1]) {
		hi, _ = strconv.Atoi(m[1][:len(m[1])-len(m[2])] + m[2])
	}
	if hi <= first || hi-first > 10 {
		return 0
	}
	return hi
}

var months = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}

var dateRE = regexp.MustCompile(`(?i)\b(January|February|March|April|May|June|July|August|September|October|November|December)\.?(?:\s+(\d{1,2})(?:st|nd|rd|th)?)?\s*,?\s+(\d{4})\b`)

// headDate finds the date a sermon was preached in the introduction of its
// head ("Delivered on Sabbath Morning, November 23, 1856"). The introduction
// ends where the text, a quotation or an editor's note begins; a date after
// "Delivered" is the one wanted, and a date given for the weekly print
// ("Published on") or for the reading ("Intended for Reading on") is not it.
// A year outside the years he preached (1850 to 1891) is a slip in the print
// and gives no date. The result is "1856-11-23", "1856-11" or "".
func headDate(head string) string {
	head = strings.Join(strings.Fields(head), " ")
	if i := strings.IndexAny(head, "\"\u201c\u201d["); i >= 0 {
		head = head[:i]
	}
	lower := strings.ToLower(head)
	from := max(strings.LastIndex(lower, "delivered"), strings.LastIndex(lower, "preached"))
	if from < 0 {
		from = 0
	}
	for _, loc := range dateRE.FindAllStringSubmatchIndex(head[from:], -1) {
		for i := range loc {
			if loc[i] >= 0 {
				loc[i] += from
			}
		}
		before := strings.ToLower(head[max(from, loc[0]-60):loc[0]])
		if strings.Contains(before, "published") || strings.Contains(before, "reading on") {
			continue
		}
		year, _ := strconv.Atoi(head[loc[6]:loc[7]])
		if year < 1850 || year > 1891 {
			return ""
		}
		month := 0
		for i, m := range months {
			if strings.EqualFold(m, head[loc[2]:loc[3]]) {
				month = i + 1
			}
		}
		if loc[4] >= 0 {
			day, _ := strconv.Atoi(head[loc[4]:loc[5]])
			if day >= 1 && day <= 31 {
				return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
			}
		}
		return fmt.Sprintf("%04d-%02d", year, month)
	}
	return ""
}

// VolumeURL is where CCEL serves a volume's ThML.
func VolumeURL(vol int) string {
	return fmt.Sprintf("https://ccel.org/ccel/spurgeon/sermons%02d.xml", vol)
}

// PageURL is the address of an entry's page on CCEL: the volume and the
// div1's section id.
func PageURL(vol int, divID string) string {
	return fmt.Sprintf("https://ccel.org/ccel/spurgeon/sermons%02d/sermons%02d.%s.html", vol, vol, url.PathEscape(divID))
}

// GemsURL is the Spurgeon Gems pdf of a sermon number; name is the file's
// name without "chs" and ".pdf": the number ("104"), the number and a letter
// for the halves of a number ("1451A"), or a range ("297-298").
func GemsURL(name string) string {
	return "https://www.spurgeongems.org/sermon/chs" + name + ".pdf"
}

// gemsRanges are the numbers Spurgeon Gems has under a range of their own
// (read from Gems: chs297-298.pdf answers, chs297.pdf and chs298.pdf do not).
var gemsRanges = map[int]string{297: "297-298", 298: "297-298"}

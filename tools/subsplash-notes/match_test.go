package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// Invented dates: 2031-03-02, 03-09 and 03-16 are Sundays.
func vids(dates ...string) []sources.Row {
	var rows []sources.Row
	for _, d := range dates {
		d := d
		rows = append(rows, sources.Row{ID: "vid-" + d, Title: "Service " + d, Date: &d})
	}
	return rows
}

func note(date string) Note {
	return Note{AlertID: "a", Title: "The Garden Gate", Ref: "Gen.2.8-Gen.2.17", Date: date}
}

func TestSundayOf(t *testing.T) {
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"2031-03-02", "2031-03-02", true}, // Sunday stays
		{"2031-03-01", "2031-03-02", true}, // Saturday -> the next day
		{"2031-03-03", "2031-03-02", true}, // Monday -> the day before
		{"2031-03-04", "", false},          // Tuesday
		{"2031-03-05", "", false},          // Wednesday
		{"2031-03-06", "", false},
		{"2031-03-07", "", false}, // Friday
		{"2031-02-30", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, err := SundayOf(c.in)
		if (err == nil) != c.ok || got != c.want {
			t.Errorf("SundayOf(%q) = %q, %v; want %q ok=%v", c.in, got, err, c.want, c.ok)
		}
	}
	// The error names the weekday, not a guess.
	if _, err := SundayOf("2031-03-05"); err == nil || !strings.Contains(err.Error(), "Wednesday") {
		t.Errorf("error = %v", err)
	}
}

// Acceptance 3 and 4: the record's date is the Sunday, matched to exactly one row.
func TestMatchSundayStaysSunday(t *testing.T) {
	rec, err := Match(note("2031-03-02"), vids("2031-02-23", "2031-03-02", "2031-03-09"))
	if err != nil {
		t.Fatal(err)
	}
	if rec.ID != "vid-2031-03-02" || rec.Date != "2031-03-02" || rec.Title != "The Garden Gate" || rec.Ref != "Gen.2.8-Gen.2.17" {
		t.Errorf("record = %+v", rec)
	}
}

func TestMatchSaturdayTakesTheSundayAfter(t *testing.T) {
	rec, err := Match(note("2031-03-01"), vids("2031-02-23", "2031-03-02"))
	if err != nil {
		t.Fatal(err)
	}
	if rec.Date != "2031-03-02" || rec.ID != "vid-2031-03-02" {
		t.Errorf("record = %+v", rec)
	}
}

func TestMatchMondayTakesTheSundayBefore(t *testing.T) {
	rec, err := Match(note("2031-03-03"), vids("2031-03-02", "2031-03-09"))
	if err != nil {
		t.Fatal(err)
	}
	if rec.Date != "2031-03-02" || rec.ID != "vid-2031-03-02" {
		t.Errorf("record = %+v", rec)
	}
}

// A row dated the Monday (YouTube's UTC publish date after a Sunday evening)
// is within one day of its Sunday.
func TestMatchRowOneDayFromTheSunday(t *testing.T) {
	rec, err := Match(note("2031-03-02"), vids("2031-03-03"))
	if err != nil {
		t.Fatal(err)
	}
	if rec.ID != "vid-2031-03-03" || rec.Date != "2031-03-02" {
		t.Errorf("record = %+v: the date is the Sunday, the id the row's", rec)
	}
}

func TestMatchOtherWeekdayIsReported(t *testing.T) {
	_, err := Match(note("2031-03-05"), vids("2031-03-02", "2031-03-05"))
	if err == nil || !strings.Contains(err.Error(), "Wednesday") {
		t.Errorf("a Wednesday alert: err = %v", err)
	}
}

func TestMatchNoRowIsReported(t *testing.T) {
	// Rows two days from the Sunday, or a week away, are not within one day.
	_, err := Match(note("2031-03-02"), vids("2031-02-28", "2031-03-04", "2031-03-09"))
	if err == nil || !strings.Contains(err.Error(), "no row") {
		t.Errorf("err = %v", err)
	}
}

func TestMatchMoreThanOneRowIsNotGuessed(t *testing.T) {
	cases := map[string]struct {
		date string
		rows []string
	}{
		"Sunday with a Sunday row and a Saturday row": {"2031-03-02", []string{"2031-03-01", "2031-03-02"}},
		"Saturday whose next Sunday has two rows":     {"2031-03-01", []string{"2031-03-02", "2031-03-03"}},
		"Monday whose previous Sunday has two rows":   {"2031-03-03", []string{"2031-03-01", "2031-03-02"}},
		"Monday with Sunday and Monday rows":          {"2031-03-03", []string{"2031-03-02", "2031-03-03"}},
	}
	for name, c := range cases {
		_, err := Match(note(c.date), vids(c.rows...))
		if err == nil || !strings.Contains(err.Error(), "more than one") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestMatchSkipsRowsWithoutADate(t *testing.T) {
	rows := vids("2031-03-02")
	rows = append(rows, sources.Row{ID: "nodate", Title: "No date"})
	rec, err := Match(note("2031-03-02"), rows)
	if err != nil || rec.ID != "vid-2031-03-02" {
		t.Errorf("rec %+v err %v", rec, err)
	}
}

func TestSundayOfIsAgainstTheCalendar(t *testing.T) {
	// The fixtures' invented Sundays really are Sundays.
	for _, d := range []string{"2031-03-02", "2031-03-09", "2031-03-16"} {
		tm, _ := time.Parse("2006-01-02", d)
		if tm.Weekday() != time.Sunday {
			t.Errorf("%s is a %s", d, tm.Weekday())
		}
	}
}

// Run reads the alerts, matches them and reports what it dropped.
func TestRunMatchesAndReports(t *testing.T) {
	rows := vids("2031-03-02", "2031-03-09", "2031-03-16")
	recs, report, err := Run(strings.NewReader(fixture(t)), rows, nil)
	if err != nil {
		t.Fatal(err)
	}
	// a1 (Sunday 03-02) and a2 (Saturday 03-08 -> Sunday 03-09) match; a4 has
	// a bad reference and a5 an attachment field; a3 is another kind.
	if len(recs) != 2 {
		t.Fatalf("records = %+v", recs)
	}
	if recs[0].Date != "2031-03-02" || recs[1].Date != "2031-03-09" || recs[1].ID != "vid-2031-03-09" {
		t.Errorf("records = %+v", recs)
	}
	text := strings.Join(report, "\n")
	for _, want := range []string{"00000000000000a4", "00000000000000a5"} {
		if !strings.Contains(text, want) {
			t.Errorf("the report does not name %s:\n%s", want, text)
		}
	}
	if strings.Contains(text, "00000000000000a3") {
		t.Errorf("the report names the charge alert:\n%s", text)
	}
}

func TestRunUnmatchedIsReportedNotWritten(t *testing.T) {
	recs, report, err := Run(strings.NewReader(fixture(t)), vids("2031-03-02"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].Date != "2031-03-02" {
		t.Errorf("records = %+v", recs)
	}
	if !strings.Contains(strings.Join(report, "\n"), "00000000000000a2") {
		t.Errorf("the unmatched Saturday alert is not reported: %v", report)
	}
}

// Run merges with what is already written, by row id: the same alert twice
// changes nothing, and an alert that disagrees with a kept record is reported
// and does not replace it.
func TestRunMergesWithExistingRecords(t *testing.T) {
	rows := vids("2031-03-02", "2031-03-09")
	first, _, err := Run(strings.NewReader(fixture(t)), rows, nil)
	if err != nil {
		t.Fatal(err)
	}
	again, report, err := Run(strings.NewReader(fixture(t)), rows, first)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != len(first) || again[0] != first[0] || again[1] != first[1] {
		t.Errorf("again %+v, first %+v", again, first)
	}
	for _, r := range report {
		if strings.Contains(r, "disagree") {
			t.Errorf("an unchanged alert disagrees: %s", r)
		}
	}

	changed := []Record{{ID: "vid-2031-03-02", Date: "2031-03-02", Title: "An Older Title", Ref: "Gen.2.8-Gen.2.17"}}
	kept, report, err := Run(strings.NewReader(fixture(t)), rows, changed)
	if err != nil {
		t.Fatal(err)
	}
	if kept[0].Title != "An Older Title" {
		t.Errorf("a kept record was replaced: %+v", kept[0])
	}
	if !strings.Contains(strings.Join(report, "\n"), "disagree") {
		t.Errorf("no report of the disagreement: %v", report)
	}
}

// run writes the records file, and a second run over the same alerts writes
// the same bytes.
func TestRunWritesTheSameBytesTwice(t *testing.T) {
	dir := t.TempDir()
	rowsPath := dir + "/rows.jsonl"
	rowsJSON := `{"id":"v1","title":"One","url":"https://example.com/1","date":"2031-03-02"}` + "\n" +
		`{"id":"v2","title":"Two","url":"https://example.com/2","date":"2031-03-09"}` + "\n"
	if err := os.WriteFile(rowsPath, []byte(rowsJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	alertsPath := dir + "/alerts.json"
	if err := os.WriteFile(alertsPath, []byte(fixture(t)), 0o644); err != nil {
		t.Fatal(err)
	}
	out := dir + "/out/records.jsonl"
	var so, se strings.Builder
	if err := run(alertsPath, rowsPath, out, nil, &so, &se); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id":"v1","date":"2031-03-02","title":"The Garden Gate","ref":"Gen.2.8-Gen.2.17"}` + "\n" +
		`{"id":"v2","date":"2031-03-09","title":"Walking the Hill","ref":"Ps.23"}` + "\n"
	if string(first) != want {
		t.Errorf("records:\n%s\nwant:\n%s", first, want)
	}
	if err := run(alertsPath, rowsPath, out, nil, &so, &se); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(out)
	if string(second) != string(first) {
		t.Errorf("a second run changed the file:\n%s", second)
	}
	if !strings.Contains(se.String(), "00000000000000a4") {
		t.Errorf("stderr does not report the bad alert: %s", se.String())
	}
}

// titledRow is a row dated d with a title and a passage.
func titledRow(id, d, title, ref string) sources.Row {
	return sources.Row{ID: id, Title: title, Ref: &ref, Date: &d}
}

func runOne(t *testing.T, rows []sources.Row) ([]Record, []string) {
	t.Helper()
	alert := alertsJSON(plainAlert("w1", 1, "Blessed in the Fire!", "1 Peter 3:13-17", "2031-03-02T00:00:00-05:00"))
	recs, report, err := Run(strings.NewReader(alert), rows, nil)
	if err != nil {
		t.Fatal(err)
	}
	return recs, report
}

// A mail a week off its row's date, with no row within a day, is written to
// the one row in 14 days with the same title (case and punctuation folded)
// and the same passage, and the report line says how it was matched.
func TestMatchByTitleAndPassageWhenTheDateIsAWeekOff(t *testing.T) {
	rows := append(vids("2031-02-23"), titledRow("blessed", "2031-03-09", "Blessed in the fire", "1Pet.3.13-1Pet.3.17"))
	recs, report := runOne(t, rows)
	if len(recs) != 1 || recs[0].ID != "blessed" || recs[0].Date != "2031-03-02" || recs[0].Ref != "1Pet.3.13-1Pet.3.17" {
		t.Fatalf("records = %+v", recs)
	}
	text := strings.Join(report, "\n")
	for _, w := range []string{"alert w1:", "matched by title and passage, row dated 2031-03-09, mail dated 2031-03-02"} {
		if !strings.Contains(text, w) {
			t.Errorf("the report lacks %q:\n%s", w, text)
		}
	}
}

// A row more than 14 days away is not taken.
func TestMatchByTitleStopsAtFourteenDays(t *testing.T) {
	recs, report := runOne(t, []sources.Row{titledRow("far", "2031-03-17", "Blessed in the Fire", "1Pet.3.13-1Pet.3.17")})
	if len(recs) != 0 || !strings.Contains(strings.Join(report, "\n"), "no row within one day of Sunday 2031-03-02") {
		t.Errorf("records %+v report %v", recs, report)
	}
}

// Two rows with that title and passage are not guessed between: the line
// names both.
func TestMatchByTitleTwoCandidatesAreNotGuessed(t *testing.T) {
	recs, report := runOne(t, []sources.Row{
		titledRow("b1", "2031-03-09", "Blessed in the Fire", "1Pet.3.13-1Pet.3.17"),
		titledRow("b2", "2031-02-23", "Blessed in the fire.", "1Pet.3.13-1Pet.3.17"),
	})
	text := strings.Join(report, "\n")
	if len(recs) != 0 || !strings.Contains(text, "not written") || !strings.Contains(text, "b1") || !strings.Contains(text, "b2") {
		t.Errorf("records %+v report %v", recs, report)
	}
}

// The same title with another passage is not a match: reported, with the row named.
func TestMatchByTitleWithAnotherPassageIsNotWritten(t *testing.T) {
	recs, report := runOne(t, []sources.Row{titledRow("other", "2031-03-09", "Blessed in the Fire", "Ps.23")})
	text := strings.Join(report, "\n")
	if len(recs) != 0 || !strings.Contains(text, "not written") || !strings.Contains(text, "other") || strings.Contains(text, "matched by") {
		t.Errorf("records %+v report %v", recs, report)
	}
}

// A row within a day always wins; the fallback is only for no row there.
func TestMatchByTitleIsNotTriedWhenARowIsWithinADay(t *testing.T) {
	rows := append(vids("2031-03-02"), titledRow("blessed", "2031-03-09", "Blessed in the Fire", "1Pet.3.13-1Pet.3.17"))
	recs, report := runOne(t, rows)
	if len(recs) != 1 || recs[0].ID != "vid-2031-03-02" || strings.Contains(strings.Join(report, "\n"), "matched by") {
		t.Errorf("records %+v report %v", recs, report)
	}
}

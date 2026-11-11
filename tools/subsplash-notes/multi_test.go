package main

import (
	"strings"
	"testing"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// The two mails the first hourly run dropped (mw-qtu3zvz.59.13): one passage
// field naming two places. 2026-06-21 is a Sunday, 2023-11-04 a Saturday.
const (
	pictures    = "Matthew 6:9/Romans 8:14-16"
	partnership = "2 Corinthians 2:15; 10:13-16"
)

func twoPlaceAlerts() string {
	return alertsJSON(
		plainAlert("b682bb042296794b", 1, "Pictures of Our Father", pictures, "2026-06-21T00:00:00-04:00"),
		plainAlert("17201668b55b45b8", 2, "Our Privilege of Partnership", partnership, "2023-11-04T00:00:00-04:00"),
	)
}

func TestTwoPlaceMailsAreNotDropped(t *testing.T) {
	notes, problems, err := ReadAlerts(strings.NewReader(twoPlaceAlerts()))
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 0 || len(notes) != 2 {
		t.Fatalf("notes %+v problems %+v", notes, problems)
	}
}

func TestTwoPlaceMailsMatchTheirRowsByDate(t *testing.T) {
	d1, d2 := "2026-06-21", "2023-11-05"
	rows := []sources.Row{{ID: "pic", Title: "Pictures", Date: &d1}, {ID: "par", Title: "Partnership", Date: &d2}}
	recs, report, err := Run(strings.NewReader(twoPlaceAlerts()), rows, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 || len(report) != 0 {
		t.Fatalf("records %+v report %v", recs, report)
	}
}

// Every place is recorded, in the order written: the first as Ref, the others
// as Also.
func TestTwoPlaceMailsRecordEveryPlace(t *testing.T) {
	notes, _, err := ReadAlerts(strings.NewReader(twoPlaceAlerts()))
	if err != nil {
		t.Fatal(err)
	}
	want := []Note{
		{AlertID: "b682bb042296794b", Title: "Pictures of Our Father", Ref: "Matt.6.9", Also: "Rom.8.14-Rom.8.16", Reference: pictures, Date: "2026-06-21"},
		{AlertID: "17201668b55b45b8", Title: "Our Privilege of Partnership", Ref: "2Cor.2.15", Also: "2Cor.10.13-2Cor.10.16", Reference: partnership, Date: "2023-11-04"},
	}
	if len(notes) != 2 || notes[0] != want[0] || notes[1] != want[1] {
		t.Errorf("notes = %+v, want %+v", notes, want)
	}
}

func TestTwoPlaceMailsRecordsKeepEveryPlace(t *testing.T) {
	d1, d2 := "2026-06-21", "2023-11-05"
	rows := []sources.Row{{ID: "pic", Title: "Pictures", Date: &d1}, {ID: "par", Title: "Partnership", Date: &d2}}
	recs, _, err := Run(strings.NewReader(twoPlaceAlerts()), rows, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Record{}
	for _, r := range recs {
		got[r.ID] = r
	}
	if r := got["pic"]; r.Ref != "Matt.6.9" || r.Also != "Rom.8.14-Rom.8.16" || r.Date != "2026-06-21" {
		t.Errorf("pic = %+v", r)
	}
	if r := got["par"]; r.Ref != "2Cor.2.15" || r.Also != "2Cor.10.13-2Cor.10.16" || r.Date != "2023-11-05" {
		t.Errorf("par = %+v", r)
	}
}

// With no row within a day, a row is taken by its title and any one of the
// places (as its passage or one it cites).
func TestMatchByTitleAndAnyPlace(t *testing.T) {
	for name, row := range map[string]sources.Row{
		"first place as ref":  titledRow("r", "2023-11-12", "Our privilege of partnership", "2Cor.2.15"),
		"second place as ref": titledRow("r", "2023-11-12", "Our privilege of partnership", "2Cor.10.13-2Cor.10.16"),
		"second place cited": func() sources.Row {
			r := titledRow("r", "2023-11-12", "Our privilege of partnership", "2Cor.9.1")
			r.Cited = []string{"2Cor.10.13-2Cor.10.16"}
			return r
		}(),
	} {
		alert := alertsJSON(plainAlert("w", 1, "Our Privilege of Partnership", partnership, "2023-11-04T00:00:00-04:00"))
		recs, report, err := Run(strings.NewReader(alert), []sources.Row{row}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(recs) != 1 || recs[0].ID != "r" || !strings.Contains(strings.Join(report, "\n"), "matched by title and passage") {
			t.Errorf("%s: records %+v report %v", name, recs, report)
		}
	}
	// A row with neither place is still not taken.
	alert := alertsJSON(plainAlert("w", 1, "Our Privilege of Partnership", partnership, "2023-11-04T00:00:00-04:00"))
	recs, _, _ := Run(strings.NewReader(alert), []sources.Row{titledRow("r", "2023-11-12", "Our privilege of partnership", "Ps.23")}, nil)
	if len(recs) != 0 {
		t.Errorf("a row with another passage was taken: %+v", recs)
	}
}

func TestReferencesToOSIS(t *testing.T) {
	cases := map[string][]string{
		"Romans 5:8":                         {"Rom.5.8"},
		"Matthew 6:9/Romans 8:14-16":         {"Matt.6.9", "Rom.8.14-Rom.8.16"},
		"2 Corinthians 2:15; 10:13-16":       {"2Cor.2.15", "2Cor.10.13-2Cor.10.16"},
		"John 3:16, 18":                      {"John.3.16", "John.3.18"},
		"John 3:16, 18-20":                   {"John.3.16", "John.3.18-John.3.20"},
		"Psalm 23; 27":                       {"Ps.23", "Ps.27"},
		"Psalm 23, John 3:16":                {"Ps.23", "John.3.16"},
		"2 Peter 2:4-10a / Jude 1:6":         {"2Pet.2.4-2Pet.2.10", "Jude.1.6"},
		"Genesis 1:1; 2:1-3; Exodus 20:8-11": {"Gen.1.1", "Gen.2.1-Gen.2.3", "Exod.20.8-Exod.20.11"},
	}
	for in, want := range cases {
		got, err := ReferencesToOSIS(in)
		if err != nil || strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("%q = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"", ";", "Matthew 6:9/Hezekiah 3:1", "Matthew 6:9/Romans 99:1", "10:13-16", "Matthew 6:9; 10a:3", "Matthew 6:9/ and more", "Isaiah 40a; 41"} {
		if got, err := ReferencesToOSIS(bad); err == nil {
			t.Errorf("%q accepted as %v", bad, got)
		}
	}
}

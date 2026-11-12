package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/Jonathan-A-White/sermon-index/internal/sources"
)

// testdata/alerts-dropped.json holds the seven alerts the hourly run dropped
// on 2026-11-11 and 2026-11-12 (mw-qtu3zvz.59.13), exactly as GET /api/alerts
// listed them: ids, titles, passages, dates and the notes' hash and size, no
// text of the notes. The rows are the Sundays' rows of the church's list.
func droppedRows() []sources.Row {
	row := func(id, date string) sources.Row {
		d := date
		return sources.Row{ID: id, Title: "Brookfield First - Online Service", Date: &d}
	}
	return []sources.Row{
		row("0jAthJ_RBsE", "2026-06-21"),
		row("DqzKNZy8t7M", "2023-11-05"),
		row("oSKPg6wTCAo", "2025-09-28"),
		row("JWR9_aEJub8", "2025-09-14"),
		row("nJQv2yqXILI", "2025-09-07"),
		row("gSTnbQ8Fe14", "2025-08-31"),
		row("L4mXkwT7tW4", "2025-08-17"),
	}
}

// want maps an alert to the row its record names.
var droppedWant = map[string]string{
	"b682bb042296794b": "0jAthJ_RBsE", // Matthew 6:9/Romans 8:14-16
	"17201668b55b45b8": "DqzKNZy8t7M", // 2 Corinthians 2:15; 10:13-16
	"a2cba2963b118613": "oSKPg6wTCAo", // 2 Cor. 11:16-33
	"a987b6f0faa31866": "JWR9_aEJub8", // 2 Cor. 10:7-18
	"4252d198eb54a85c": "nJQv2yqXILI", // 2 Cor. 10:1-6
	"f8252164bc6808e8": "gSTnbQ8Fe14", // 2 Cor. 9:1-15
	"a7f24e79fdb1ac2b": "L4mXkwT7tW4", // no passage
}

// The saved replies go through merge(), which fetch calls (with the notes
// keeper in the keep callback): none is dropped or reported, each is recorded
// for its row and each would be claimed.
func TestSavedDroppedAlertsAreRecordedForTheirRows(t *testing.T) {
	body, err := os.ReadFile("testdata/alerts-dropped.json")
	if err != nil {
		t.Fatal(err)
	}
	kept := map[string]string{}
	records, report, recorded, err := merge(bytes.NewReader(body), droppedRows(), nil, func(n Note, rec Record) error {
		kept[n.AlertID] = rec.ID
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report) != 0 {
		t.Errorf("reported: %s", strings.Join(report, "\n"))
	}
	if len(records) != len(droppedWant) || len(recorded) != len(droppedWant) {
		t.Errorf("%d records, %d recorded, want %d", len(records), len(recorded), len(droppedWant))
	}
	for alert, row := range droppedWant {
		if kept[alert] != row {
			t.Errorf("alert %s kept for row %q, want %q", alert, kept[alert], row)
		}
	}
}

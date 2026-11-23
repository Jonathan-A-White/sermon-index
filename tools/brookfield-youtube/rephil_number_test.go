package main

import "testing"

// An episode whose title prints no number is the next after the highest
// number printed by the episodes published before it, not its place in the
// list (the live list served 171 for the newest, when the series was at 481).
func TestBuildNumbersAnUnnumberedEpisodeAfterTheHighestBefore(t *testing.T) {
	res := Build([]Item{
		// Newest first, as the playlist gives them.
		{VideoID: "newNoNumber1", Title: "Morning RePhil", Published: "2026-11-23T13:00:00Z"},
		{VideoID: "rephil480aaa", Title: "Morning RePhil #480", Published: "2026-11-21T03:37:00Z"},
		{VideoID: "rephil479aaa", Title: "Morning RePhil #479", Published: "2026-11-13T12:00:00Z"},
		{VideoID: "oldNoNumber1", Title: "Morning RePhil", Published: "2023-01-05T12:00:00Z"},
	})
	got := map[string]int{}
	for _, r := range res.Rephil {
		got[r.URL[len(watchURL):]] = r.Number
	}
	if got["newNoNumber1"] != 481 {
		t.Errorf("the unnumbered episode after #480 is %d, want 481", got["newNoNumber1"])
	}
	if got["oldNoNumber1"] != 1 {
		t.Errorf("the unnumbered episode before every numbered one is %d, want its place, 1", got["oldNoNumber1"])
	}
	if got["rephil480aaa"] != 480 || got["rephil479aaa"] != 479 {
		t.Errorf("numbers a title prints changed: %v", got)
	}
	if len(res.Unnumbered) != 2 {
		t.Errorf("unnumbered %v", res.Unnumbered)
	}
}

// Two unnumbered episodes in a row count on from the first.
func TestBuildNumbersTwoUnnumberedEpisodesInARow(t *testing.T) {
	res := Build([]Item{
		{VideoID: "bbbbbbbbbbb", Title: "Morning RePhil", Published: "2026-11-24T12:00:00Z"},
		{VideoID: "aaaaaaaaaaa", Title: "Morning RePhil", Published: "2026-11-23T12:00:00Z"},
		{VideoID: "rephil480aaa", Title: "Morning RePhil #480", Published: "2026-11-21T12:00:00Z"},
	})
	if n := res.Rephil[1].Number; n != 481 {
		t.Errorf("first unnumbered %d, want 481", n)
	}
	if n := res.Rephil[2].Number; n != 482 {
		t.Errorf("second unnumbered %d, want 482", n)
	}
}

// YouTube gives the publish instant in UTC; the church is in Connecticut. An
// episode published 2026-11-21T03:37Z went up on the evening of the 20th.
func TestBuildDatesAnEpisodeByTheChurchsDay(t *testing.T) {
	res := Build([]Item{
		{VideoID: "rephil480aaa", Title: "Morning RePhil #480", Published: "2026-11-21T03:37:00Z"},
		{VideoID: "rephilSummer1", Title: "Morning RePhil #300", Published: "2025-07-01T02:30:00Z"},
	})
	byNum := map[int]string{}
	for _, r := range res.Rephil {
		byNum[r.Number] = r.Date
	}
	if byNum[480] != "2026-11-20" {
		t.Errorf("#480 dated %q, want 2026-11-20", byNum[480])
	}
	if byNum[300] != "2025-06-30" {
		t.Errorf("#300 dated %q, want 2025-06-30 (daylight time)", byNum[300])
	}
}

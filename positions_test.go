package main

import (
	"reflect"
	"testing"
)

func TestBuildPositionSummariesCountsDraftedAndRemainingTargets(t *testing.T) {
	players := PlayerDatabase{Index: NewPlayerIndex([]Player{
		{ID: "p1", Position: "QB"},
		{ID: "p2", Position: "RB"},
		{ID: "p3", Position: "RB"},
	})}

	got := BuildPositionSummaries(
		map[string]int{"QB": 2, "RB": 3, "WR": 4},
		[]Pick{{PlayerID: "p2"}, {PlayerID: "p1"}, {PlayerID: "p3"}},
		players,
	)

	want := []PositionSummary{
		{Position: "QB", Target: 2, Drafted: 1, Remaining: 1},
		{Position: "RB", Target: 3, Drafted: 2, Remaining: 1},
		{Position: "WR", Target: 4, Drafted: 0, Remaining: 4},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("summaries = %#v, want %#v", got, want)
	}
}

func TestBuildPositionSummariesIncludesOverTargetDraftedPositions(t *testing.T) {
	players := PlayerDatabase{Index: NewPlayerIndex([]Player{
		{ID: "p1", Position: "TE"},
		{ID: "p2", Position: "TE"},
	})}

	got := BuildPositionSummaries(map[string]int{"TE": 1}, []Pick{{PlayerID: "p1"}, {PlayerID: "p2"}}, players)
	want := []PositionSummary{{Position: "TE", Target: 1, Drafted: 2, Remaining: 0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("summaries = %#v, want %#v", got, want)
	}
}

func TestBuildPositionSummariesFallsBackToPickMetadataPosition(t *testing.T) {
	players := PlayerDatabase{Index: NewPlayerIndex(nil)}

	got := BuildPositionSummaries(
		map[string]int{"DEF": 1},
		[]Pick{{PlayerID: "missing", Metadata: map[string]string{"position": "def"}}},
		players,
	)

	want := []PositionSummary{{Position: "DEF", Target: 1, Drafted: 1, Remaining: 0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("summaries = %#v, want %#v", got, want)
	}
}

func TestBuildPositionSummariesIncludesUntargetedDraftedPositions(t *testing.T) {
	players := PlayerDatabase{Index: NewPlayerIndex([]Player{
		{ID: "p1", Position: "K"},
	})}

	got := BuildPositionSummaries(map[string]int{"QB": 1}, []Pick{{PlayerID: "p1"}}, players)
	want := []PositionSummary{
		{Position: "K", Target: 0, Drafted: 1, Remaining: 0},
		{Position: "QB", Target: 1, Drafted: 0, Remaining: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("summaries = %#v, want %#v", got, want)
	}
}

func TestFormatPositionSummaries(t *testing.T) {
	got := FormatPositionSummaries([]PositionSummary{
		{Position: "QB", Target: 2, Drafted: 1, Remaining: 1},
		{Position: "RB", Target: 3, Drafted: 2, Remaining: 1},
	})
	want := "Position summary: QB drafted=1 remaining=1 target=2; RB drafted=2 remaining=1 target=3"
	if got != want {
		t.Fatalf("formatted = %q, want %q", got, want)
	}
}

func TestPlayerIndexLookupByIDTrimsInput(t *testing.T) {
	index := NewPlayerIndex([]Player{{ID: "p1", Name: "Player", Position: "QB"}})

	player, ok := index.LookupByID(" p1 ")
	if !ok {
		t.Fatal("LookupByID returned ok=false")
	}
	if player.ID != "p1" {
		t.Fatalf("ID = %q, want p1", player.ID)
	}
}

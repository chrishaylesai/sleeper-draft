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
	}), Players: []Player{
		{ID: "p1", Position: "QB"},
		{ID: "p2", Position: "RB"},
		{ID: "p3", Position: "RB"},
		{ID: "p4", Position: "WR"},
	}}

	got := BuildPositionSummaries(
		map[string]int{"QB": 2, "RB": 3, "WR": 4},
		[]Pick{
			{PlayerID: "p2", PickedBy: "me"},
			{PlayerID: "p1", PickedBy: "other"},
			{PlayerID: "p3", RosterID: 7},
		},
		players,
		PersonalDraft{UserID: "me", RosterID: 7, Known: true},
	)

	want := []PositionSummary{
		{Position: "QB", Target: 2, PersonalDrafted: 0, PersonalRemaining: 2, TotalDrafted: 1, TotalRemaining: 0},
		{Position: "RB", Target: 3, PersonalDrafted: 2, PersonalRemaining: 1, TotalDrafted: 2, TotalRemaining: 0},
		{Position: "WR", Target: 4, PersonalDrafted: 0, PersonalRemaining: 4, TotalDrafted: 0, TotalRemaining: 1},
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

	got := BuildPositionSummaries(map[string]int{"TE": 1}, []Pick{{PlayerID: "p1", PickedBy: "me"}, {PlayerID: "p2", PickedBy: "me"}}, players, PersonalDraft{UserID: "me", Known: true})
	want := []PositionSummary{{Position: "TE", Target: 1, PersonalDrafted: 2, PersonalRemaining: 0, TotalDrafted: 2, TotalRemaining: 0}}
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
		PersonalDraft{},
	)

	want := []PositionSummary{{Position: "DEF", Target: 1, PersonalDrafted: 0, PersonalRemaining: 1, TotalDrafted: 1, TotalRemaining: 0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("summaries = %#v, want %#v", got, want)
	}
}

func TestBuildPositionSummariesIncludesUntargetedDraftedPositions(t *testing.T) {
	players := PlayerDatabase{Index: NewPlayerIndex([]Player{
		{ID: "p1", Position: "K"},
	})}

	got := BuildPositionSummaries(map[string]int{"QB": 1}, []Pick{{PlayerID: "p1"}}, players, PersonalDraft{})
	want := []PositionSummary{
		{Position: "K", Target: 0, PersonalDrafted: 0, PersonalRemaining: 0, TotalDrafted: 1, TotalRemaining: 0},
		{Position: "QB", Target: 1, PersonalDrafted: 0, PersonalRemaining: 1, TotalDrafted: 0, TotalRemaining: 0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("summaries = %#v, want %#v", got, want)
	}
}

func TestBuildPositionSummariesSkipsExplicitZeroTargetPositions(t *testing.T) {
	players := PlayerDatabase{Index: NewPlayerIndex([]Player{
		{ID: "def", Position: "DEF"},
		{ID: "k", Position: "K"},
		{ID: "qb", Position: "QB"},
	})}

	got := BuildPositionSummaries(
		map[string]int{"DEF": 0, "K": 0, "QB": 2},
		[]Pick{{PlayerID: "def"}, {PlayerID: "k"}, {PlayerID: "qb"}},
		players,
		PersonalDraft{},
	)

	want := []PositionSummary{{Position: "QB", Target: 2, PersonalDrafted: 0, PersonalRemaining: 2, TotalDrafted: 1, TotalRemaining: 0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("summaries = %#v, want %#v", got, want)
	}
}

func TestBuildRelevantPositionSummariesCountsPrunedRemainingPool(t *testing.T) {
	players := []Player{
		{ID: "top-qb", Name: "Top QB", Position: "QB", SearchRank: intPtr(25)},
		{ID: "deep-qb", Name: "Deep QB", Position: "QB", SearchRank: intPtr(900)},
		{ID: "wishlist-qb", Name: "Wishlist QB", Position: "QB", SearchRank: intPtr(1200)},
		{ID: "irrelevant-wr", Name: "Irrelevant WR", Position: "WR", SearchRank: intPtr(900)},
	}

	got := BuildRelevantPositionSummaries(
		Config{
			PositionTargets: map[string]int{"QB": 2, "WR": 1},
			PruneRankCutoff: 300,
		},
		[]Pick{{PlayerID: "deep-qb"}},
		PlayerDatabase{Players: players, Index: NewPlayerIndex(players)},
		PlayerList{},
		PlayerList{Entries: []RankedPlayer{{PlayerID: "wishlist-qb"}}},
		PersonalDraft{},
	)

	want := []PositionSummary{
		{Position: "QB", Target: 2, PersonalDrafted: 0, PersonalRemaining: 2, TotalDrafted: 1, TotalRemaining: 2},
		{Position: "WR", Target: 1, PersonalDrafted: 0, PersonalRemaining: 1, TotalDrafted: 0, TotalRemaining: 0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("summaries = %#v, want %#v", got, want)
	}
}

func TestFormatPositionSummaries(t *testing.T) {
	got := FormatPositionSummaries([]PositionSummary{
		{Position: "QB", Target: 2, PersonalDrafted: 1, PersonalRemaining: 1, TotalDrafted: 4, TotalRemaining: 12},
		{Position: "RB", Target: 3, PersonalDrafted: 2, PersonalRemaining: 1, TotalDrafted: 8, TotalRemaining: 20},
	})
	want := "Position summary: QB personal_drafted=1 personal_remaining=1 target=2 total_drafted=4 total_remaining=12; RB personal_drafted=2 personal_remaining=1 target=3 total_drafted=8 total_remaining=20"
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

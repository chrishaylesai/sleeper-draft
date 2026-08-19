package main

import (
	"reflect"
	"testing"
)

func TestBestAvailableByPositionUsesCustomRankingsFirst(t *testing.T) {
	rank10 := 10
	rank1 := 1
	players := PlayerDatabase{
		Players: []Player{
			{ID: "rb-sleeper", Name: "Sleeper RB", Team: "ATL", Position: "RB", SearchRank: &rank1},
			{ID: "rb-custom", Name: "Custom RB", Team: "DET", Position: "RB", SearchRank: &rank10},
		},
	}
	rankings := PlayerList{Entries: []RankedPlayer{
		{Rank: 1, PlayerID: "rb-custom"},
	}}

	got := BestAvailableByPosition(Config{PositionTargets: map[string]int{"RB": 1}}, players, rankings, nil)

	if len(got) != 1 {
		t.Fatalf("len(best) = %d, want 1", len(got))
	}
	if got[0].Player.ID != "rb-custom" || got[0].Source != "custom" || got[0].Rank != 1 {
		t.Fatalf("best = %#v, want custom-ranked RB", got[0])
	}
}

func TestBestAvailableByPositionFallsBackToSleeperSearchRank(t *testing.T) {
	rank5 := 5
	rank2 := 2
	players := PlayerDatabase{
		Players: []Player{
			{ID: "wr5", Name: "WR Five", Team: "ATL", Position: "WR", SearchRank: &rank5},
			{ID: "wr2", Name: "WR Two", Team: "DET", Position: "WR", SearchRank: &rank2},
		},
	}

	got := BestAvailableByPosition(Config{PositionTargets: map[string]int{"WR": 1}}, players, PlayerList{}, nil)

	if got[0].Player.ID != "wr2" || got[0].Source != "sleeper" || got[0].Rank != 2 {
		t.Fatalf("best = %#v, want lower sleeper rank", got[0])
	}
}

func TestBestAvailableByPositionExcludesDraftedPlayersAndConfigExclusions(t *testing.T) {
	rank1 := 1
	rank2 := 2
	rank3 := 3
	rank4 := 4
	players := PlayerDatabase{
		Players: []Player{
			{ID: "taken", Name: "Taken RB", Team: "ATL", Position: "RB", SearchRank: &rank1},
			{ID: "excluded-id", Name: "Excluded RB", Team: "DET", Position: "RB", SearchRank: &rank2},
			{ID: "team-excluded", Name: "Team RB", Team: "FA", Position: "RB", SearchRank: &rank3},
			{ID: "available", Name: "Available RB", Team: "DAL", Position: "RB", SearchRank: &rank4},
		},
	}

	got := BestAvailableByPosition(
		Config{
			PositionTargets: map[string]int{"RB": 1},
			ExcludedPlayers: []string{"excluded-id"},
			ExcludedTeams:   []string{"FA"},
		},
		players,
		PlayerList{},
		[]Pick{{PlayerID: "taken"}},
	)

	if got[0].Player.ID != "available" {
		t.Fatalf("best = %#v, want available player after exclusions", got[0])
	}
}

func TestBestAvailableByPositionSupportsNameExclusions(t *testing.T) {
	rank1 := 1
	rank2 := 2
	players := PlayerDatabase{
		Players: []Player{
			{ID: "p1", Name: "Excluded Name", Team: "ATL", Position: "TE", SearchRank: &rank1},
			{ID: "p2", Name: "Available TE", Team: "DAL", Position: "TE", SearchRank: &rank2},
		},
	}

	got := BestAvailableByPosition(
		Config{PositionTargets: map[string]int{"TE": 1}, ExcludedPlayers: []string{"excluded name"}},
		players,
		PlayerList{},
		nil,
	)

	if got[0].Player.ID != "p2" {
		t.Fatalf("best = %#v, want name exclusion applied", got[0])
	}
}

func TestBestAvailableByPositionReturnsTargetPositionsInOrder(t *testing.T) {
	rank1 := 1
	players := PlayerDatabase{
		Players: []Player{
			{ID: "wr", Name: "Wide Receiver", Team: "ATL", Position: "WR", SearchRank: &rank1},
			{ID: "qb", Name: "Quarterback", Team: "BUF", Position: "QB", SearchRank: &rank1},
			{ID: "ignored", Name: "Ignored K", Team: "DAL", Position: "K", SearchRank: &rank1},
		},
	}

	got := BestAvailableByPosition(Config{PositionTargets: map[string]int{"WR": 1, "QB": 1}}, players, PlayerList{}, nil)
	want := []string{"QB", "WR"}
	if positions := bestAvailablePositions(got); !reflect.DeepEqual(positions, want) {
		t.Fatalf("positions = %#v, want %#v", positions, want)
	}
}

func TestFormatBestAvailable(t *testing.T) {
	got := FormatBestAvailable([]BestAvailable{
		{Position: "QB", Player: Player{Name: "Quarterback", Team: "BUF"}, Source: "custom", Rank: 1},
		{Position: "RB", Player: Player{Name: "Running Back"}, Source: "sleeper", Rank: 2},
		{Position: "DEF", Player: Player{Name: "Defense", Team: "DAL"}, Source: "unranked"},
	})
	want := "Best available: QB Quarterback (BUF) custom_rank=1; RB Running Back (FA) sleeper_rank=2; DEF Defense (DAL) unranked"
	if got != want {
		t.Fatalf("formatted = %q, want %q", got, want)
	}
}

func bestAvailablePositions(best []BestAvailable) []string {
	positions := make([]string, 0, len(best))
	for _, item := range best {
		positions = append(positions, item.Position)
	}
	return positions
}

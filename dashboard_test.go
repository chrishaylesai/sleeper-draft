package main

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDashboardModelRendersSnapshotSections(t *testing.T) {
	rank1 := 1
	model := NewDashboardModel(AppState{
		Config: Config{
			PositionTargets:        map[string]int{"QB": 1},
			RefreshIntervalSeconds: 5,
		},
		Players: PlayerDatabase{
			Players: []Player{{ID: "qb", Name: "Quarterback", Team: "BUF", Position: "QB", SearchRank: &rank1}},
			Index:   NewPlayerIndex([]Player{{ID: "qb", Name: "Quarterback", Team: "BUF", Position: "QB", SearchRank: &rank1}}),
		},
		Rankings: PlayerList{},
		Wishlist: PlayerList{Entries: []RankedPlayer{
			{Rank: 1, PlayerID: "qb", Player: Player{ID: "qb", Name: "Quarterback", Team: "BUF", Position: "QB"}},
		}},
		DraftID: "draft123",
	})
	model.width = 100
	model.loaded = true
	model.snapshot = DraftSnapshot{
		Draft:        Draft{ID: "draft123", Status: "drafting", Settings: DraftSettings{Teams: 2, Rounds: 1}},
		Picks:        nil,
		TotalPicks:   0,
		NextPickNo:   1,
		CurrentRound: 1,
		CurrentPick:  1,
		UpdatedAt:    time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC),
	}

	view := model.View()
	for _, want := range []string{"Sleeper Draft Dashboard", "Round 1", "Positions", "Best Available", "Wishlist", "Quarterback"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
}

func TestDashboardModelKeepsSnapshotOnRefreshError(t *testing.T) {
	model := NewDashboardModel(AppState{Config: Config{RefreshIntervalSeconds: 5}})
	model.loaded = true
	model.snapshot = DraftSnapshot{TotalPicks: 3}

	updated, _ := model.Update(snapshotMsg{err: errors.New("api unavailable")})
	got := updated.(dashboardModel)

	if !got.loaded {
		t.Fatal("loaded = false, want existing snapshot retained")
	}
	if got.snapshot.TotalPicks != 3 {
		t.Fatalf("TotalPicks = %d, want retained 3", got.snapshot.TotalPicks)
	}
	if got.lastErr == nil || !strings.Contains(got.lastErr.Error(), "api unavailable") {
		t.Fatalf("lastErr = %v, want refresh error", got.lastErr)
	}
}

func TestDashboardModelQuitsOnQ(t *testing.T) {
	model := NewDashboardModel(AppState{})

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("cmd = nil, want quit command")
	}
}

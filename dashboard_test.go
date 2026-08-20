package main

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func TestDashboardStylesUseForegroundColors(t *testing.T) {
	styles := map[string]lipgloss.Style{
		"title":        titleStyle,
		"meta":         metaStyle,
		"sectionTitle": sectionTitle,
		"label":        labelStyle,
		"healthy":      healthyStyle,
		"warning":      warningStyle,
		"danger":       dangerStyle,
		"customRank":   customRankStyle,
		"sleeperRank":  sleeperRankStyle,
		"unranked":     unrankedStyle,
		"footer":       footerStyle,
	}

	for name, style := range styles {
		if _, ok := style.GetForeground().(lipgloss.NoColor); ok {
			t.Fatalf("%s style has no foreground color", name)
		}
	}
}

func TestStyledStatusHelpersKeepReadableText(t *testing.T) {
	if got := styledPersonalRemaining(PositionSummary{Target: 1, PersonalRemaining: 0}); !strings.Contains(got, "0") {
		t.Fatalf("styled remaining = %q, want readable number", got)
	}
	if got := styledRank("sleeper", 7); !strings.Contains(got, "sleeper rank 7") {
		t.Fatalf("styled rank = %q, want readable rank", got)
	}
	item := WishlistItem{Rank: 1, Position: "RB", Player: Player{Name: "Target", Team: "ATL"}, Status: WishlistAvailable}
	if got := styledWishlistItem(item); !strings.Contains(got, "available") {
		t.Fatalf("styled wishlist item = %q, want status text", got)
	}
}

func TestPositionTableRowPadsColoredRemainingByVisibleWidth(t *testing.T) {
	header := positionTableRow("POS", "MY_DRAFTED", "MY_LEFT", "TARGET", "TOTAL_DRAFTED", "TOTAL_LEFT")
	row := positionTableRow("QB", "0", styledPersonalRemaining(PositionSummary{Target: 2, PersonalRemaining: 2}), "2", "4", "12")
	targetColumn := strings.Index(header, "TARGET")
	plainRow := stripANSI(row)
	targetValueIndex := strings.Index(plainRow, "2       2")

	if !strings.Contains(plainRow, "2       2") {
		t.Fatalf("row = %q, want target separated from colored remaining value", row)
	}
	if got := lipgloss.Width(plainRow[:targetValueIndex+8]); got != targetColumn {
		t.Fatalf("target starts at visible column %d, want %d", got, targetColumn)
	}
}

func stripANSI(value string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(value, "")
}

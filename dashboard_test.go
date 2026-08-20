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
		Draft:        Draft{ID: "draft123", Status: "drafting", Type: "snake", Settings: DraftSettings{Teams: 2, Rounds: 1}},
		Picks:        nil,
		TotalPicks:   0,
		NextPickNo:   1,
		CurrentRound: 1,
		CurrentPick:  1,
		UpdatedAt:    time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC),
	}

	view := model.View()
	for _, want := range []string{"Sleeper Draft Dashboard", "Round 1", "My Picks", "Positions", "Best Available", "Wishlist", "Quarterback"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
}

func TestDashboardModelKeepsSnapshotOnRefreshError(t *testing.T) {
	model := NewDashboardModel(AppState{Config: Config{RefreshIntervalSeconds: 5}})
	model.loaded = true
	model.snapshot = DraftSnapshot{TotalPicks: 3}
	model.refreshing = true

	updated, _ := model.Update(snapshotMsg{err: errors.New("api unavailable")})
	got := updated.(dashboardModel)

	if got.refreshing {
		t.Fatal("refreshing = true, want cleared after snapshot response")
	}
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

func TestDashboardRefreshTickKeepsCadenceAndStartsFetchWhenIdle(t *testing.T) {
	model := NewDashboardModel(AppState{Config: Config{RefreshIntervalSeconds: 5}})
	model.refreshing = false

	updated, cmd := model.Update(refreshTickMsg(time.Now()))
	got := updated.(dashboardModel)

	if !got.refreshing {
		t.Fatal("refreshing = false, want tick to start snapshot fetch")
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want next tick and fetch commands")
	}
}

func TestDashboardRefreshTickSkipsOverlappingFetch(t *testing.T) {
	model := NewDashboardModel(AppState{Config: Config{RefreshIntervalSeconds: 5}})
	model.refreshing = true

	updated, cmd := model.Update(refreshTickMsg(time.Now()))
	got := updated.(dashboardModel)

	if !got.refreshing {
		t.Fatal("refreshing = false, want in-flight fetch retained")
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want next tick command")
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
	if got, want := styledPersonalPickNumber(4, 4), dangerStyle.Render("4"); got != want {
		t.Fatalf("used pick style = %q, want %q", got, want)
	}
	if got, want := styledPersonalPickNumber(5, 4), healthyStyle.Render("5"); got != want {
		t.Fatalf("upcoming pick style = %q, want %q", got, want)
	}
	countdownSnapshot := DraftSnapshot{
		Draft:      Draft{Type: "snake", Settings: DraftSettings{Teams: 12, Rounds: 3}},
		TotalPicks: 12,
		NextPickNo: 13,
	}
	if got, want := styledPicksUntilNext(countdownSnapshot, PersonalDraft{DraftSlot: 4, Known: true}), healthyStyle.Render("8"); got != want {
		t.Fatalf("countdown style = %q, want %q", got, want)
	}
	onClockSnapshot := DraftSnapshot{
		Draft:      Draft{Type: "snake", Settings: DraftSettings{Teams: 12, Rounds: 3}},
		TotalPicks: 20,
		NextPickNo: 21,
	}
	if got, want := styledPicksUntilNext(onClockSnapshot, PersonalDraft{DraftSlot: 4, Known: true}), warningStyle.Render("on the clock"); got != want {
		t.Fatalf("on-clock style = %q, want %q", got, want)
	}
	if got, want := styledPicksUntilNext(DraftSnapshot{Complete: true}, PersonalDraft{DraftSlot: 4, Known: true}), unrankedStyle.Render("none"); got != want {
		t.Fatalf("none style = %q, want %q", got, want)
	}
	if got := styledRank("sleeper", 7); !strings.Contains(got, "sleeper rank 7") {
		t.Fatalf("styled rank = %q, want readable rank", got)
	}
	item := WishlistItem{Rank: 1, Position: "RB", Player: Player{Name: "Target", Team: "ATL"}, Status: WishlistAvailable}
	if got := styledWishlistItem(item); !strings.Contains(got, "available") {
		t.Fatalf("styled wishlist item = %q, want status text", got)
	}
	takenByOther := WishlistItem{Rank: 1, Position: "RB", Player: Player{Name: "Taken", Team: "ATL"}, Status: WishlistTaken}
	if got, want := styledWishlistItem(takenByOther), dangerStyle.Render(formatWishlistItem(takenByOther)); got != want {
		t.Fatalf("taken by other style = %q, want %q", got, want)
	}
	takenByMe := WishlistItem{Rank: 1, Position: "RB", Player: Player{Name: "Mine", Team: "ATL"}, Status: WishlistTaken, SelectedByMe: true}
	if got, want := styledWishlistItem(takenByMe), warningStyle.Render(formatWishlistItem(takenByMe)); got != want {
		t.Fatalf("taken by me style = %q, want %q", got, want)
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

func TestRenderDraftSummaryShowsPicksUntilNextPersonalPick(t *testing.T) {
	view := stripANSI(renderDraftSummary(
		DraftSnapshot{
			Draft:        Draft{ID: "draft123", Status: "drafting", Type: "snake", Settings: DraftSettings{Teams: 12, Rounds: 3}},
			TotalPicks:   12,
			NextPickNo:   13,
			CurrentRound: 2,
			CurrentPick:  1,
		},
		PersonalDraft{DraftSlot: 4, Known: true},
	))

	if !strings.Contains(view, "Until yours 8") {
		t.Fatalf("view missing picks-until field:\n%s", view)
	}
}

func TestRenderMyPicksShowsCompactUsedAndUpcomingPicks(t *testing.T) {
	view := renderMyPicks(
		DraftSnapshot{
			Draft:      Draft{Type: "snake", Settings: DraftSettings{Teams: 4, Rounds: 3}},
			TotalPicks: 5,
		},
		PersonalDraft{DraftSlot: 2, Known: true},
	)

	plain := stripANSI(view)
	if !strings.Contains(plain, "My Picks") {
		t.Fatalf("view missing section title:\n%s", view)
	}
	if !strings.Contains(plain, "2 7 10") {
		t.Fatalf("view missing compact pick row:\n%s", view)
	}
}

func TestRenderMyPicksShowsUnavailableWithoutPersonalDraft(t *testing.T) {
	view := stripANSI(renderMyPicks(DraftSnapshot{Draft: Draft{Settings: DraftSettings{Teams: 4, Rounds: 3}}}, PersonalDraft{}))

	if !strings.Contains(view, "Unavailable") {
		t.Fatalf("view = %q, want unavailable message", view)
	}
}

func TestRenderWishlistSplitsItemsIntoPositionColumns(t *testing.T) {
	report := WishlistReport{Items: []WishlistItem{
		{Rank: 2, Position: "WR", Player: Player{Name: "WR Two", Team: "DET"}, Status: WishlistAvailable},
		{Rank: 1, Position: "RB", Player: Player{Name: "RB One", Team: "ATL"}, Status: WishlistTaken, TakenBy: "user1", PickNo: 5},
		{Rank: 1, Position: "WR", Player: Player{Name: "WR One", Team: "LAR"}, Status: WishlistAvailable},
	}}
	report.TopAvailable = &report.Items[2]
	report.TopAvailableByPosition = []WishlistItem{report.Items[0], report.Items[2]}

	view := stripANSI(renderWishlist(report, map[string]int{"RB": 1, "WR": 1}))

	if !strings.Contains(view, "RB") || !strings.Contains(view, "WR") {
		t.Fatalf("view missing position column headers:\n%s", view)
	}
	if !strings.Contains(view, "#1 RB RB One (ATL) taken") {
		t.Fatalf("view missing RB wishlist item:\n%s", view)
	}
	if strings.Contains(view, "taken_by=") || strings.Contains(view, "pick=5") {
		t.Fatalf("view includes verbose taken details:\n%s", view)
	}
	if strings.Index(view, "WR One") > strings.Index(view, "WR Two") {
		t.Fatalf("WR items not in rank order:\n%s", view)
	}
}

func TestRenderWishlistShowsEmptyPositiveTargetColumns(t *testing.T) {
	view := stripANSI(renderWishlist(WishlistReport{}, map[string]int{"QB": 2, "RB": 0, "TE": 1}))

	if !strings.Contains(view, "QB") || !strings.Contains(view, "TE") {
		t.Fatalf("view missing empty positive-target columns:\n%s", view)
	}
	if strings.Contains(view, "RB") {
		t.Fatalf("view includes zero-target RB column:\n%s", view)
	}
	if count := strings.Count(view, "empty"); count != 2 {
		t.Fatalf("empty column count = %d, want 2:\n%s", count, view)
	}
}

func TestRenderWishlistOmitsZeroTargetPositionsWithItems(t *testing.T) {
	report := WishlistReport{Items: []WishlistItem{
		{Rank: 1, Position: "K", Player: Player{Name: "Kicker", Team: "DAL"}, Status: WishlistAvailable},
		{Rank: 2, Position: "QB", Player: Player{Name: "Quarterback", Team: "BUF"}, Status: WishlistAvailable},
	}}

	view := stripANSI(renderWishlist(report, map[string]int{"K": 0, "QB": 1}))

	if strings.Contains(view, "Kicker") || strings.Contains(view, "\nK") {
		t.Fatalf("view includes zero-target K column or item:\n%s", view)
	}
	if !strings.Contains(view, "Quarterback") {
		t.Fatalf("view missing positive-target QB item:\n%s", view)
	}
}

func stripANSI(value string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(value, "")
}

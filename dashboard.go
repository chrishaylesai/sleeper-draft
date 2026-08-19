package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type dashboardModel struct {
	state    AppState
	loaded   bool
	snapshot DraftSnapshot
	lastErr  error
	width    int
	height   int
}

type snapshotMsg struct {
	snapshot DraftSnapshot
	err      error
}

type refreshTickMsg time.Time

func NewDashboardModel(state AppState) dashboardModel {
	return dashboardModel{state: state}
}

func (m dashboardModel) Init() tea.Cmd {
	return m.fetchSnapshot()
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case snapshotMsg:
		if msg.err != nil {
			m.lastErr = msg.err
		} else {
			m.loaded = true
			m.snapshot = msg.snapshot
			m.lastErr = nil
		}
		return m, m.scheduleRefresh()
	case refreshTickMsg:
		return m, m.fetchSnapshot()
	}

	return m, nil
}

func (m dashboardModel) View() string {
	width := m.width
	if width < 80 {
		width = 80
	}

	if !m.loaded {
		return dashboardFrame(width,
			renderHeader(m, "Loading draft..."),
			renderError(m.lastErr),
			footerView(),
		)
	}

	positionSummaries := BuildPositionSummaries(m.state.Config.PositionTargets, m.snapshot.Picks, m.state.Players)
	bestAvailable := BestAvailableByPosition(m.state.Config, m.state.Players, m.state.Rankings, m.snapshot.Picks)
	wishlistReport := BuildWishlistReport(m.state.Config, m.state.Wishlist, m.snapshot.Picks)

	content := []string{
		renderHeader(m, "Live"),
		renderError(m.lastErr),
		renderDraftSummary(m.snapshot),
		renderPositionTable(positionSummaries),
		renderBestAvailable(bestAvailable),
		renderWishlist(wishlistReport),
		footerView(),
	}

	return dashboardFrame(width, content...)
}

func (m dashboardModel) fetchSnapshot() tea.Cmd {
	return func() tea.Msg {
		snapshot, err := m.state.Drafts.Snapshot(context.Background(), m.state.DraftID)
		return snapshotMsg{snapshot: snapshot, err: err}
	}
}

func (m dashboardModel) scheduleRefresh() tea.Cmd {
	return tea.Tick(m.state.RefreshInterval(), func(t time.Time) tea.Msg {
		return refreshTickMsg(t)
	})
}

func dashboardFrame(width int, sections ...string) string {
	bodyWidth := width - 4
	if bodyWidth < 60 {
		bodyWidth = 60
	}

	cleaned := make([]string, 0, len(sections))
	for _, section := range sections {
		if strings.TrimSpace(section) != "" {
			cleaned = append(cleaned, section)
		}
	}
	return lipgloss.NewStyle().Width(bodyWidth).Padding(1, 2).Render(strings.Join(cleaned, "\n\n"))
}

func renderHeader(m dashboardModel, status string) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("Sleeper Draft Dashboard")
	meta := fmt.Sprintf("draft=%s refresh=%s status=%s", m.state.DraftID, m.state.RefreshInterval(), status)
	return title + "\n" + meta
}

func renderError(err error) string {
	if err == nil {
		return ""
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("Last refresh error: " + err.Error())
}

func renderDraftSummary(snapshot DraftSnapshot) string {
	updated := "unknown"
	if !snapshot.UpdatedAt.IsZero() {
		updated = snapshot.UpdatedAt.Format("15:04:05")
	}

	state := "active"
	if snapshot.Complete {
		state = "complete"
	}

	return sectionStyle().Render(fmt.Sprintf(
		"Draft\nRound %d  Pick %d  Next %d  Total picks %d  Status %s  Updated %s",
		snapshot.CurrentRound,
		snapshot.CurrentPick,
		snapshot.NextPickNo,
		snapshot.TotalPicks,
		state,
		updated,
	))
}

func renderPositionTable(summaries []PositionSummary) string {
	if len(summaries) == 0 {
		return sectionStyle().Render("Positions\nNo position targets configured.")
	}

	rows := []string{"Positions", "POS  DRAFTED  REMAINING  TARGET"}
	for _, summary := range summaries {
		rows = append(rows, fmt.Sprintf("%-4s %-8d %-10d %d", summary.Position, summary.Drafted, summary.Remaining, summary.Target))
	}
	return sectionStyle().Render(strings.Join(rows, "\n"))
}

func renderBestAvailable(best []BestAvailable) string {
	if len(best) == 0 {
		return sectionStyle().Render("Best Available\nNone")
	}

	rows := []string{"Best Available"}
	for _, item := range best {
		rank := item.Source
		if item.Source != "unranked" {
			rank = fmt.Sprintf("%s rank %d", item.Source, item.Rank)
		}
		rows = append(rows, fmt.Sprintf("%-4s %-28s %-4s %s", item.Position, item.Player.Name, emptyTeamAsFA(item.Player.Team), rank))
	}
	return sectionStyle().Render(strings.Join(rows, "\n"))
}

func renderWishlist(report WishlistReport) string {
	if len(report.Items) == 0 {
		return sectionStyle().Render("Wishlist\nNo wishlist entries.")
	}

	rows := []string{"Wishlist"}
	if report.TopAvailable != nil {
		rows = append(rows, "Top: "+wishlistPlayerLabel(*report.TopAvailable))
	} else {
		rows = append(rows, "Top: none")
	}
	if len(report.TopAvailableByPosition) > 0 {
		rows = append(rows, "By position: "+formatTopWishlistByPosition(report.TopAvailableByPosition))
	}
	for _, item := range report.Items {
		rows = append(rows, "  "+formatWishlistItem(item))
	}
	return sectionStyle().Render(strings.Join(rows, "\n"))
}

func footerView() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("q / Ctrl-C quit")
}

func sectionStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)
}

func emptyTeamAsFA(team string) string {
	if strings.TrimSpace(team) == "" {
		return "FA"
	}
	return team
}

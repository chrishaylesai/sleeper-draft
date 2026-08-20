package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
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

var (
	titleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	metaStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sectionTitle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	labelStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	healthyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warningStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	dangerStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	customRankStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	sleeperRankStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("45"))
	unrankedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	footerStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

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

	positionSummaries := BuildRelevantPositionSummaries(m.state.Config, m.snapshot.Picks, m.state.Players, m.state.Rankings, m.state.Wishlist, m.state.Personal)
	bestAvailable := BestAvailableByPosition(m.state.Config, m.state.Players, m.state.Rankings, m.snapshot.Picks)
	wishlistReport := BuildWishlistReport(m.state.Config, m.state.Wishlist, m.snapshot.Picks)

	content := []string{
		renderHeader(m, "Live"),
		renderError(m.lastErr),
		renderDraftSummary(m.snapshot),
		renderMyPicks(m.snapshot, m.state.Personal),
		renderPositionTable(positionSummaries),
		renderBestAvailable(bestAvailable),
		renderWishlist(wishlistReport, m.state.Config.PositionTargets),
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
	title := titleStyle.Render("Sleeper Draft Dashboard")
	meta := metaStyle.Render(fmt.Sprintf("draft=%s refresh=%s status=%s", m.state.DraftID, m.state.RefreshInterval(), status))
	return title + "\n" + meta
}

func renderError(err error) string {
	if err == nil {
		return ""
	}
	return dangerStyle.Render("Last refresh error: " + err.Error())
}

func renderDraftSummary(snapshot DraftSnapshot) string {
	updated := "unknown"
	if !snapshot.UpdatedAt.IsZero() {
		updated = snapshot.UpdatedAt.Format("15:04:05")
	}

	state := healthyStyle.Render("active")
	if snapshot.Complete {
		state = warningStyle.Render("complete")
	}

	return sectionStyle().Render(fmt.Sprintf(
		"%s\n%s %d  %s %d  %s %d  %s %d  %s %s  %s %s",
		sectionTitle.Render("Draft"),
		labelStyle.Render("Round"),
		snapshot.CurrentRound,
		labelStyle.Render("Pick"),
		snapshot.CurrentPick,
		labelStyle.Render("Next"),
		snapshot.NextPickNo,
		labelStyle.Render("Total picks"),
		snapshot.TotalPicks,
		labelStyle.Render("Status"),
		state,
		labelStyle.Render("Updated"),
		updated,
	))
}

func renderMyPicks(snapshot DraftSnapshot, personal PersonalDraft) string {
	rows := []string{sectionTitle.Render("My Picks")}
	picks := PersonalPickNumbers(snapshot.Draft, personal)
	if len(picks) == 0 {
		rows = append(rows, unrankedStyle.Render("Unavailable"))
		return sectionStyle().Render(strings.Join(rows, "\n"))
	}

	values := make([]string, 0, len(picks))
	for _, pickNo := range picks {
		values = append(values, styledPersonalPickNumber(pickNo, snapshot.TotalPicks))
	}
	rows = append(rows, strings.Join(values, " "))
	return sectionStyle().Render(strings.Join(rows, "\n"))
}

func renderPositionTable(summaries []PositionSummary) string {
	if len(summaries) == 0 {
		return sectionStyle().Render(sectionTitle.Render("Positions") + "\nNo position targets configured.")
	}

	rows := []string{
		sectionTitle.Render("Positions"),
		labelStyle.Render(positionTableRow("POS", "MY_DRAFTED", "MY_LEFT", "TARGET", "TOTAL_DRAFTED", "TOTAL_LEFT")),
	}
	for _, summary := range summaries {
		remaining := styledPersonalRemaining(summary)
		rows = append(rows, positionTableRow(
			summary.Position,
			strconv.Itoa(summary.PersonalDrafted),
			remaining,
			strconv.Itoa(summary.Target),
			strconv.Itoa(summary.TotalDrafted),
			strconv.Itoa(summary.TotalRemaining),
		))
	}
	return sectionStyle().Render(strings.Join(rows, "\n"))
}

func renderBestAvailable(best []BestAvailable) string {
	if len(best) == 0 {
		return sectionStyle().Render(sectionTitle.Render("Best Available") + "\nNone")
	}

	rows := []string{sectionTitle.Render("Best Available")}
	for _, item := range best {
		rank := styledRank(item.Source, item.Rank)
		rows = append(rows, fmt.Sprintf("%-4s %-28s %-4s %s", item.Position, item.Player.Name, emptyTeamAsFA(item.Player.Team), rank))
	}
	return sectionStyle().Render(strings.Join(rows, "\n"))
}

func renderWishlist(report WishlistReport, targets map[string]int) string {
	rows := []string{sectionTitle.Render("Wishlist")}
	if report.TopAvailable != nil {
		rows = append(rows, labelStyle.Render("Top: ")+healthyStyle.Render(wishlistPlayerLabel(*report.TopAvailable)))
	} else {
		rows = append(rows, labelStyle.Render("Top: ")+"none")
	}
	if len(report.TopAvailableByPosition) > 0 {
		rows = append(rows, labelStyle.Render("By position: ")+healthyStyle.Render(formatTopWishlistByPosition(report.TopAvailableByPosition)))
	}

	columns := renderWishlistColumns(report.Items, targets)
	if strings.TrimSpace(columns) == "" {
		rows = append(rows, "No positive position targets configured.")
	} else {
		rows = append(rows, columns)
	}
	return sectionStyle().Render(strings.Join(rows, "\n"))
}

func footerView() string {
	return footerStyle.Render("q / Ctrl-C quit")
}

func sectionStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)
}

func styledPersonalRemaining(summary PositionSummary) string {
	text := strconv.Itoa(summary.PersonalRemaining)
	if summary.PersonalRemaining == 0 {
		return dangerStyle.Render(text)
	}
	if summary.Target > 0 && summary.PersonalRemaining <= 1 {
		return warningStyle.Render(text)
	}
	return healthyStyle.Render(text)
}

func styledPersonalPickNumber(pickNo, totalPicks int) string {
	text := strconv.Itoa(pickNo)
	if pickNo <= totalPicks {
		return dangerStyle.Render(text)
	}
	return healthyStyle.Render(text)
}

func styledRank(source string, rank int) string {
	switch source {
	case "custom":
		return customRankStyle.Render(fmt.Sprintf("custom rank %d", rank))
	case "sleeper":
		return sleeperRankStyle.Render(fmt.Sprintf("sleeper rank %d", rank))
	default:
		return unrankedStyle.Render("unranked")
	}
}

func styledWishlistItem(item WishlistItem) string {
	text := formatWishlistItem(item)
	switch item.Status {
	case WishlistAvailable:
		return healthyStyle.Render(text)
	case WishlistTaken:
		return dangerStyle.Render(text)
	case WishlistExcluded:
		return warningStyle.Render(text)
	default:
		return text
	}
}

func renderWishlistColumns(items []WishlistItem, targets map[string]int) string {
	positions := positiveTargetPositions(targets)
	if len(positions) == 0 {
		return ""
	}

	itemsByPosition := wishlistItemsByPosition(items)
	columns := make([]string, 0, len(positions))
	for _, position := range positions {
		rows := []string{sectionTitle.Render(position)}
		positionItems := itemsByPosition[position]
		if len(positionItems) == 0 {
			rows = append(rows, unrankedStyle.Render("empty"))
		} else {
			for _, item := range positionItems {
				rows = append(rows, styledWishlistItem(item))
			}
		}
		columns = append(columns, lipgloss.NewStyle().Width(44).Render(strings.Join(rows, "\n")))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, columns...)
}

func positiveTargetPositions(targets map[string]int) []string {
	normalized := normalizePositionTargets(targets)
	positions := make([]string, 0, len(normalized))
	for position, target := range normalized {
		if target > 0 {
			positions = append(positions, position)
		}
	}
	sort.Strings(positions)
	return positions
}

func wishlistItemsByPosition(items []WishlistItem) map[string][]WishlistItem {
	itemsByPosition := make(map[string][]WishlistItem)
	for _, item := range items {
		position := normalizePosition(item.Position)
		if position == "" {
			continue
		}
		itemsByPosition[position] = append(itemsByPosition[position], item)
	}
	for position := range itemsByPosition {
		sort.SliceStable(itemsByPosition[position], func(i, j int) bool {
			return itemsByPosition[position][i].Rank < itemsByPosition[position][j].Rank
		})
	}
	return itemsByPosition
}

func positionTableRow(position, personalDrafted, personalRemaining, target, totalDrafted, totalRemaining string) string {
	return strings.Join([]string{
		padRightVisible(position, 4),
		padRightVisible(personalDrafted, 10),
		padRightVisible(personalRemaining, 7),
		padRightVisible(target, 6),
		padRightVisible(totalDrafted, 13),
		totalRemaining,
	}, " ")
}

func padRightVisible(value string, width int) string {
	padding := width - lipgloss.Width(value)
	if padding <= 0 {
		return value
	}
	return value + strings.Repeat(" ", padding)
}

func emptyTeamAsFA(team string) string {
	if strings.TrimSpace(team) == "" {
		return "FA"
	}
	return team
}

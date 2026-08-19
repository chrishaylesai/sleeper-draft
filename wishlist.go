package main

import (
	"fmt"
	"sort"
	"strings"
)

type WishlistStatus string

const (
	WishlistAvailable WishlistStatus = "available"
	WishlistTaken     WishlistStatus = "taken"
	WishlistExcluded  WishlistStatus = "excluded"
)

type WishlistItem struct {
	Rank     int
	Player   Player
	Status   WishlistStatus
	TakenBy  string
	PickNo   int
	Position string
}

type WishlistReport struct {
	Items                  []WishlistItem
	TopAvailable           *WishlistItem
	TopAvailableByPosition []WishlistItem
}

func BuildWishlistReport(cfg Config, wishlist PlayerList, picks []Pick) WishlistReport {
	picksByPlayerID := picksByPlayerID(picks)
	excludedPlayers := normalizedSet(cfg.ExcludedPlayers)
	excludedTeams := normalizedSet(cfg.ExcludedTeams)

	items := make([]WishlistItem, 0, len(wishlist.Entries))
	for _, entry := range wishlist.Entries {
		item := WishlistItem{
			Rank:     entry.Rank,
			Player:   entry.Player,
			Status:   WishlistAvailable,
			Position: normalizePosition(entry.Player.Position),
		}

		if pick, ok := picksByPlayerID[normalizeLookupText(entry.PlayerID)]; ok {
			item.Status = WishlistTaken
			item.TakenBy = pickedByLabel(pick)
			item.PickNo = pick.PickNo
		} else if playerIsExcluded(entry.Player, excludedPlayers, excludedTeams) {
			item.Status = WishlistExcluded
		}

		items = append(items, item)
	}

	report := WishlistReport{Items: items}
	for i := range items {
		if items[i].Status == WishlistAvailable {
			report.TopAvailable = &items[i]
			break
		}
	}
	report.TopAvailableByPosition = topWishlistByPosition(items)
	return report
}

func FormatWishlistReport(report WishlistReport) string {
	if len(report.Items) == 0 {
		return "Wishlist: none"
	}

	parts := make([]string, 0, len(report.Items))
	for _, item := range report.Items {
		parts = append(parts, formatWishlistItem(item))
	}

	top := "top=none"
	if report.TopAvailable != nil {
		top = fmt.Sprintf("top=%s", wishlistPlayerLabel(*report.TopAvailable))
	}
	byPosition := formatTopWishlistByPosition(report.TopAvailableByPosition)

	return fmt.Sprintf("Wishlist: %s; %s; top_by_position=%s", strings.Join(parts, "; "), top, byPosition)
}

func formatWishlistItem(item WishlistItem) string {
	label := wishlistPlayerLabel(item)
	switch item.Status {
	case WishlistTaken:
		return fmt.Sprintf("#%d %s taken_by=%s pick=%d", item.Rank, label, item.TakenBy, item.PickNo)
	case WishlistExcluded:
		return fmt.Sprintf("#%d %s excluded", item.Rank, label)
	default:
		return fmt.Sprintf("#%d %s available", item.Rank, label)
	}
}

func wishlistPlayerLabel(item WishlistItem) string {
	team := item.Player.Team
	if team == "" {
		team = "FA"
	}
	return fmt.Sprintf("%s %s (%s)", item.Position, item.Player.Name, team)
}

func formatTopWishlistByPosition(items []WishlistItem) string {
	if len(items) == 0 {
		return "none"
	}

	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, wishlistPlayerLabel(item))
	}
	return strings.Join(parts, ", ")
}

func topWishlistByPosition(items []WishlistItem) []WishlistItem {
	best := make(map[string]WishlistItem)
	for _, item := range items {
		if item.Status != WishlistAvailable || item.Position == "" {
			continue
		}
		if current, ok := best[item.Position]; !ok || item.Rank < current.Rank {
			best[item.Position] = item
		}
	}

	positions := make([]string, 0, len(best))
	for position := range best {
		positions = append(positions, position)
	}
	sort.Strings(positions)

	ordered := make([]WishlistItem, 0, len(positions))
	for _, position := range positions {
		ordered = append(ordered, best[position])
	}
	return ordered
}

func picksByPlayerID(picks []Pick) map[string]Pick {
	byID := make(map[string]Pick, len(picks))
	for _, pick := range picks {
		if id := normalizeLookupText(pick.PlayerID); id != "" {
			byID[id] = pick
		}
	}
	return byID
}

func pickedByLabel(pick Pick) string {
	if pickedBy := strings.TrimSpace(pick.PickedBy); pickedBy != "" {
		return pickedBy
	}
	if pick.RosterID > 0 {
		return fmt.Sprintf("roster:%d", pick.RosterID)
	}
	return "unknown"
}

func playerIsExcluded(player Player, excludedPlayers, excludedTeams map[string]bool) bool {
	return excludedPlayers[normalizeLookupText(player.ID)] ||
		excludedPlayers[normalizeLookupText(player.Name)] ||
		excludedTeams[normalizeLookupText(player.Team)]
}

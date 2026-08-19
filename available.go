package main

import (
	"fmt"
	"sort"
	"strings"
)

type BestAvailable struct {
	Position string
	Player   Player
	Source   string
	Rank     int
}

func BestAvailableByPosition(cfg Config, players PlayerDatabase, rankings PlayerList, picks []Pick) []BestAvailable {
	drafted := draftedPlayerIDs(picks)
	excludedPlayers := normalizedSet(cfg.ExcludedPlayers)
	excludedTeams := normalizedSet(cfg.ExcludedTeams)
	rankingByID := rankingMap(rankings)

	positions := targetPositions(cfg.PositionTargets)
	best := make(map[string]BestAvailable)

	for _, player := range players.Players {
		position := normalizePosition(player.Position)
		if position == "" {
			continue
		}
		if _, tracked := positions[position]; !tracked {
			continue
		}
		if drafted[normalizeLookupText(player.ID)] {
			continue
		}
		if playerIsExcluded(player, excludedPlayers, excludedTeams) {
			continue
		}

		candidate := availableCandidate(player, rankingByID)
		current, ok := best[position]
		if !ok || compareBestAvailable(candidate, current) < 0 {
			best[position] = candidate
		}
	}

	ordered := make([]string, 0, len(positions))
	for position := range positions {
		ordered = append(ordered, position)
	}
	sort.Strings(ordered)

	available := make([]BestAvailable, 0, len(ordered))
	for _, position := range ordered {
		if bestPlayer, ok := best[position]; ok {
			available = append(available, bestPlayer)
		}
	}
	return available
}

func FormatBestAvailable(best []BestAvailable) string {
	if len(best) == 0 {
		return "Best available: none"
	}

	parts := make([]string, 0, len(best))
	for _, item := range best {
		team := item.Player.Team
		if team == "" {
			team = "FA"
		}
		rank := item.Source
		if item.Source != "unranked" {
			rank = fmt.Sprintf("%s_rank=%d", item.Source, item.Rank)
		}
		parts = append(parts, fmt.Sprintf("%s %s (%s) %s", item.Position, item.Player.Name, team, rank))
	}
	return "Best available: " + strings.Join(parts, "; ")
}

func availableCandidate(player Player, rankingByID map[string]int) BestAvailable {
	position := normalizePosition(player.Position)
	if rank, ok := rankingByID[player.ID]; ok {
		return BestAvailable{Position: position, Player: player, Source: "custom", Rank: rank}
	}
	if player.SearchRank != nil {
		return BestAvailable{Position: position, Player: player, Source: "sleeper", Rank: *player.SearchRank}
	}
	return BestAvailable{Position: position, Player: player, Source: "unranked", Rank: 0}
}

func compareBestAvailable(a, b BestAvailable) int {
	if priority := rankSourcePriority(a.Source) - rankSourcePriority(b.Source); priority != 0 {
		return priority
	}
	if a.Rank != b.Rank {
		if a.Rank < b.Rank {
			return -1
		}
		return 1
	}
	if nameCompare := strings.Compare(a.Player.Name, b.Player.Name); nameCompare != 0 {
		return nameCompare
	}
	return strings.Compare(a.Player.ID, b.Player.ID)
}

func rankSourcePriority(source string) int {
	switch source {
	case "custom":
		return 0
	case "sleeper":
		return 1
	default:
		return 2
	}
}

func rankingMap(rankings PlayerList) map[string]int {
	ranks := make(map[string]int, len(rankings.Entries))
	for _, entry := range rankings.Entries {
		if _, exists := ranks[entry.PlayerID]; !exists {
			ranks[entry.PlayerID] = entry.Rank
		}
	}
	return ranks
}

func draftedPlayerIDs(picks []Pick) map[string]bool {
	drafted := make(map[string]bool, len(picks))
	for _, pick := range picks {
		if id := normalizeLookupText(pick.PlayerID); id != "" {
			drafted[id] = true
		}
	}
	return drafted
}

func normalizedSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		if normalized := normalizeLookupText(value); normalized != "" {
			set[normalized] = true
		}
	}
	return set
}

func targetPositions(targets map[string]int) map[string]struct{} {
	positions := make(map[string]struct{}, len(targets))
	for position, target := range targets {
		if target <= 0 {
			continue
		}
		if normalized := normalizePosition(position); normalized != "" {
			positions[normalized] = struct{}{}
		}
	}
	return positions
}

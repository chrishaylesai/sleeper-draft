package main

import (
	"fmt"
	"sort"
	"strings"
)

type PositionSummary struct {
	Position          string
	Target            int
	PersonalDrafted   int
	PersonalRemaining int
	TotalDrafted      int
	TotalRemaining    int
}

func BuildPositionSummaries(targets map[string]int, picks []Pick, players PlayerDatabase, personal PersonalDraft) []PositionSummary {
	return buildPositionSummaries(targets, picks, players, players.Players, personal)
}

func BuildRelevantPositionSummaries(cfg Config, picks []Pick, players PlayerDatabase, rankings, wishlist PlayerList, personal PersonalDraft) []PositionSummary {
	return buildPositionSummaries(cfg.PositionTargets, picks, players, relevantPlayers(cfg, players.Players, rankings, wishlist), personal)
}

func buildPositionSummaries(targets map[string]int, picks []Pick, players PlayerDatabase, playerPool []Player, personal PersonalDraft) []PositionSummary {
	configuredTargets := normalizePositionTargets(targets)
	totalDrafted := make(map[string]int)
	personalDrafted := make(map[string]int)
	for _, pick := range picks {
		position := positionForPick(pick, players.Index)
		if position == "" {
			continue
		}
		totalDrafted[position]++
		if personal.MatchesPick(pick) {
			personalDrafted[position]++
		}
	}
	totalAvailable := availablePlayersByPosition(playerPool, picks)

	positions := make(map[string]struct{})
	for position, target := range configuredTargets {
		if target > 0 {
			positions[position] = struct{}{}
		}
	}
	for position := range totalDrafted {
		target, configured := configuredTargets[position]
		if configured && target <= 0 {
			continue
		}
		positions[position] = struct{}{}
	}

	ordered := make([]string, 0, len(positions))
	for position := range positions {
		ordered = append(ordered, position)
	}
	sort.Strings(ordered)

	summaries := make([]PositionSummary, 0, len(ordered))
	for _, position := range ordered {
		target := configuredTargets[position]
		personalCount := personalDrafted[position]
		personalRemaining := target - personalCount
		if personalRemaining < 0 {
			personalRemaining = 0
		}
		summaries = append(summaries, PositionSummary{
			Position:          position,
			Target:            target,
			PersonalDrafted:   personalCount,
			PersonalRemaining: personalRemaining,
			TotalDrafted:      totalDrafted[position],
			TotalRemaining:    totalAvailable[position],
		})
	}
	return summaries
}

func FormatPositionSummaries(summaries []PositionSummary) string {
	if len(summaries) == 0 {
		return "Position summary: none"
	}

	parts := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		parts = append(parts, fmt.Sprintf(
			"%s personal_drafted=%d personal_remaining=%d target=%d total_drafted=%d total_remaining=%d",
			summary.Position,
			summary.PersonalDrafted,
			summary.PersonalRemaining,
			summary.Target,
			summary.TotalDrafted,
			summary.TotalRemaining,
		))
	}
	return "Position summary: " + strings.Join(parts, "; ")
}

func positionForPick(pick Pick, index PlayerIndex) string {
	if player, ok := index.LookupByID(pick.PlayerID); ok {
		if position := normalizePosition(player.Position); position != "" {
			return position
		}
	}
	return normalizePosition(pick.Metadata["position"])
}

func normalizePositionTargets(targets map[string]int) map[string]int {
	normalized := make(map[string]int, len(targets))
	for position, target := range targets {
		if normalizedPosition := normalizePosition(position); normalizedPosition != "" {
			normalized[normalizedPosition] = target
		}
	}
	return normalized
}

func availablePlayersByPosition(players []Player, picks []Pick) map[string]int {
	drafted := draftedPlayerIDs(picks)
	totals := make(map[string]int)
	for _, player := range players {
		if drafted[normalizeLookupText(player.ID)] {
			continue
		}
		if position := normalizePosition(player.Position); position != "" {
			totals[position]++
		}
	}
	return totals
}

func normalizePosition(position string) string {
	return strings.ToUpper(strings.TrimSpace(position))
}

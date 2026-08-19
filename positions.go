package main

import (
	"fmt"
	"sort"
	"strings"
)

type PositionSummary struct {
	Position  string
	Target    int
	Drafted   int
	Remaining int
}

func BuildPositionSummaries(targets map[string]int, picks []Pick, players PlayerDatabase) []PositionSummary {
	drafted := make(map[string]int)
	for _, pick := range picks {
		position := positionForPick(pick, players.Index)
		if position == "" {
			continue
		}
		drafted[position]++
	}

	positions := make(map[string]struct{})
	for position := range targets {
		normalized := normalizePosition(position)
		if normalized != "" {
			positions[normalized] = struct{}{}
		}
	}
	for position := range drafted {
		positions[position] = struct{}{}
	}

	ordered := make([]string, 0, len(positions))
	for position := range positions {
		ordered = append(ordered, position)
	}
	sort.Strings(ordered)

	summaries := make([]PositionSummary, 0, len(ordered))
	for _, position := range ordered {
		target := targetForPosition(targets, position)
		count := drafted[position]
		remaining := target - count
		if remaining < 0 {
			remaining = 0
		}
		summaries = append(summaries, PositionSummary{
			Position:  position,
			Target:    target,
			Drafted:   count,
			Remaining: remaining,
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
			"%s drafted=%d remaining=%d target=%d",
			summary.Position,
			summary.Drafted,
			summary.Remaining,
			summary.Target,
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

func targetForPosition(targets map[string]int, position string) int {
	for rawPosition, target := range targets {
		if normalizePosition(rawPosition) == position {
			return target
		}
	}
	return 0
}

func normalizePosition(position string) string {
	return strings.ToUpper(strings.TrimSpace(position))
}

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	defaultSport = "nfl"

	// defaultPruneRankCutoff is the Sleeper `search_rank` cutoff used by -prune
	// when config.json omits `prune_rank_cutoff`. It keeps a deep
	// fantasy-relevant pool while dropping most of the player database.
	defaultPruneRankCutoff = 500
)

type Config struct {
	DraftID                string         `json:"draft_id"`
	Username               string         `json:"username"`
	Season                 string         `json:"season"`
	Sport                  string         `json:"sport"`
	PositionTargets        map[string]int `json:"position_targets"`
	ExcludedPlayers        []string       `json:"excluded_players"`
	ExcludedTeams          []string       `json:"excluded_teams"`
	RefreshIntervalSeconds int            `json:"refresh_interval_seconds"`
	PruneRankCutoff        int            `json:"prune_rank_cutoff"`
	PlayersPath            string         `json:"players_path"`
	RankingsPath           string         `json:"rankings_path"`
	WishlistPath           string         `json:"wishlist_path"`
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}

	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config %q: %w", path, err)
	}

	return cfg, nil
}

func (c *Config) ApplyDefaults() {
	c.DraftID = strings.TrimSpace(c.DraftID)
	c.Username = strings.TrimSpace(c.Username)
	c.Season = strings.TrimSpace(c.Season)
	c.Sport = strings.TrimSpace(c.Sport)
	c.PlayersPath = strings.TrimSpace(c.PlayersPath)
	c.RankingsPath = strings.TrimSpace(c.RankingsPath)
	c.WishlistPath = strings.TrimSpace(c.WishlistPath)

	if c.Sport == "" {
		c.Sport = defaultSport
	}
	if c.PruneRankCutoff == 0 {
		c.PruneRankCutoff = defaultPruneRankCutoff
	}
}

func (c Config) Validate() error {
	var problems []string

	if c.DraftID == "" && (c.Username == "" || c.Season == "") {
		problems = append(problems, "provide either draft_id or both username and season")
	}
	if c.Sport == "" {
		problems = append(problems, "sport is required")
	}
	if c.RefreshIntervalSeconds <= 0 {
		problems = append(problems, "refresh_interval_seconds must be greater than 0")
	}
	if c.PruneRankCutoff <= 0 {
		problems = append(problems, "prune_rank_cutoff must be greater than 0")
	}
	if c.PlayersPath == "" {
		problems = append(problems, "players_path is required")
	}
	if c.RankingsPath == "" {
		problems = append(problems, "rankings_path is required")
	}
	if c.WishlistPath == "" {
		problems = append(problems, "wishlist_path is required")
	}

	for pos, target := range c.PositionTargets {
		if strings.TrimSpace(pos) == "" {
			problems = append(problems, "position_targets cannot contain an empty position")
		}
		if target < 0 {
			problems = append(problems, fmt.Sprintf("position_targets.%s must be nonnegative", pos))
		}
	}

	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func (c Config) StartupSummary() string {
	target := c.DraftID
	if target == "" {
		target = fmt.Sprintf("%s/%s", c.Username, c.Season)
	}

	return fmt.Sprintf(
		"Config loaded: target=%s sport=%s refresh=%ds positions=%s players=%s rankings=%s wishlist=%s",
		target,
		c.Sport,
		c.RefreshIntervalSeconds,
		formatPositionTargets(c.PositionTargets),
		c.PlayersPath,
		c.RankingsPath,
		c.WishlistPath,
	)
}

func formatPositionTargets(targets map[string]int) string {
	if len(targets) == 0 {
		return "none"
	}

	positions := make([]string, 0, len(targets))
	for position := range targets {
		positions = append(positions, position)
	}
	sort.Strings(positions)

	parts := make([]string, 0, len(positions))
	for _, position := range positions {
		parts = append(parts, fmt.Sprintf("%s:%d", position, targets[position]))
	}
	return strings.Join(parts, ",")
}

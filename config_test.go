package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigWithDraftID(t *testing.T) {
	cfg := loadTestConfig(t, `{
		"draft_id": "123",
		"position_targets": {"QB": 2, "RB": 5},
		"refresh_interval_seconds": 10,
		"players_path": "players.json",
		"rankings_path": "rankings.csv",
		"wishlist_path": "wishlist.csv"
	}`)

	if cfg.DraftID != "123" {
		t.Fatalf("DraftID = %q, want 123", cfg.DraftID)
	}
	if cfg.Sport != defaultSport {
		t.Fatalf("Sport = %q, want %q", cfg.Sport, defaultSport)
	}
}

func TestLoadConfigWithUsernameAndSeason(t *testing.T) {
	cfg := loadTestConfig(t, `{
		"username": "drafter",
		"season": "2026",
		"sport": "nfl",
		"position_targets": {"WR": 6},
		"refresh_interval_seconds": 5,
		"players_path": "players.json",
		"rankings_path": "rankings.csv",
		"wishlist_path": "wishlist.csv"
	}`)

	if cfg.Username != "drafter" || cfg.Season != "2026" {
		t.Fatalf("resolved identity = %q/%q, want drafter/2026", cfg.Username, cfg.Season)
	}
}

func TestLoadConfigDefaultsPruneRankCutoff(t *testing.T) {
	cfg := loadTestConfig(t, `{
		"draft_id": "123",
		"position_targets": {"QB": 2},
		"refresh_interval_seconds": 10,
		"players_path": "players.json",
		"rankings_path": "rankings.csv",
		"wishlist_path": "wishlist.csv"
	}`)

	if cfg.PruneRankCutoff != defaultPruneRankCutoff {
		t.Fatalf("PruneRankCutoff = %d, want %d", cfg.PruneRankCutoff, defaultPruneRankCutoff)
	}
}

func TestLoadConfigUsesConfiguredPruneRankCutoff(t *testing.T) {
	cfg := loadTestConfig(t, `{
		"draft_id": "123",
		"position_targets": {"QB": 2},
		"refresh_interval_seconds": 10,
		"prune_rank_cutoff": 300,
		"players_path": "players.json",
		"rankings_path": "rankings.csv",
		"wishlist_path": "wishlist.csv"
	}`)

	if cfg.PruneRankCutoff != 300 {
		t.Fatalf("PruneRankCutoff = %d, want 300", cfg.PruneRankCutoff)
	}
}

func TestLoadConfigRejectsNegativePruneRankCutoff(t *testing.T) {
	err := loadConfigError(t, `{
		"draft_id": "123",
		"position_targets": {"QB": 1},
		"refresh_interval_seconds": 10,
		"prune_rank_cutoff": -1,
		"players_path": "players.json",
		"rankings_path": "rankings.csv",
		"wishlist_path": "wishlist.csv"
	}`)

	assertErrorContains(t, err, "prune_rank_cutoff must be greater than 0")
}

func TestLoadConfigRejectsMissingDraftIdentity(t *testing.T) {
	err := loadConfigError(t, `{
		"position_targets": {"QB": 1},
		"refresh_interval_seconds": 10,
		"players_path": "players.json",
		"rankings_path": "rankings.csv",
		"wishlist_path": "wishlist.csv"
	}`)

	assertErrorContains(t, err, "provide either draft_id or both username and season")
}

func TestLoadConfigRejectsInvalidRefreshInterval(t *testing.T) {
	err := loadConfigError(t, `{
		"draft_id": "123",
		"position_targets": {"QB": 1},
		"refresh_interval_seconds": 0,
		"players_path": "players.json",
		"rankings_path": "rankings.csv",
		"wishlist_path": "wishlist.csv"
	}`)

	assertErrorContains(t, err, "refresh_interval_seconds must be greater than 0")
}

func TestLoadConfigRejectsMissingPaths(t *testing.T) {
	err := loadConfigError(t, `{
		"draft_id": "123",
		"position_targets": {"QB": 1},
		"refresh_interval_seconds": 10
	}`)

	assertErrorContains(t, err, "players_path is required")
	assertErrorContains(t, err, "rankings_path is required")
	assertErrorContains(t, err, "wishlist_path is required")
}

func TestLoadConfigRejectsInvalidJSON(t *testing.T) {
	err := loadConfigError(t, `{"draft_id":`)

	assertErrorContains(t, err, "parse config")
}

func TestLoadConfigRejectsNegativePositionTarget(t *testing.T) {
	err := loadConfigError(t, `{
		"draft_id": "123",
		"position_targets": {"QB": -1},
		"refresh_interval_seconds": 10,
		"players_path": "players.json",
		"rankings_path": "rankings.csv",
		"wishlist_path": "wishlist.csv"
	}`)

	assertErrorContains(t, err, "position_targets.QB must be nonnegative")
}

func loadTestConfig(t *testing.T, content string) Config {
	t.Helper()

	cfg, err := LoadConfig(writeTestConfig(t, content))
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	return cfg
}

func loadConfigError(t *testing.T, content string) error {
	t.Helper()

	_, err := LoadConfig(writeTestConfig(t, content))
	if err == nil {
		t.Fatal("LoadConfig returned nil error")
	}
	return err
}

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write test config: %v", err)
	}
	return path
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()

	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want substring %q", err.Error(), want)
	}
}

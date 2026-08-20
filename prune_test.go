package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPrunePlayerCacheKeepsRelevantPlayersUnderCutoff(t *testing.T) {
	players := []Player{
		{ID: "1", Name: "Bijan Robinson", Team: "ATL", Position: "RB", SearchRank: intPtr(3)},
		{ID: "2", Name: "Edge Case", Team: "SF", Position: "WR", SearchRank: intPtr(500)},
		{ID: "3", Name: "Deep Sleeper", Team: "NYJ", Position: "WR", SearchRank: intPtr(501)},
		{ID: "4", Name: "Retired Guy", Team: "", Position: "LB", SearchRank: intPtr(9999999)},
		{ID: "5", Name: "No Rank", Team: "DEN", Position: "TE", SearchRank: nil},
		{ID: "ARI", Name: "Arizona Cardinals", Team: "ARI", Position: "DEF", SearchRank: nil},
		{ID: "6", Name: "Wishlist Longshot", Team: "CHI", Position: "RB", SearchRank: intPtr(1200)},
		{ID: "7", Name: "Rankless Longshot", Team: "GB", Position: "QB", SearchRank: nil},
	}
	cfg := prunableConfig(t, 500, players, nil, []string{
		"Wishlist Longshot,RB,CHI",
		"Rankless Longshot,QB,GB",
	})

	result, err := PrunePlayerCache(context.Background(), cfg, prunePlayerService(t), PruneOptions{})
	if err != nil {
		t.Fatalf("PrunePlayerCache returned error: %v", err)
	}

	kept := keptPlayerIDs(t, cfg.PlayersPath)
	want := []string{"1", "2", "ARI", "6", "7"}
	if strings.Join(kept, ",") != strings.Join(want, ",") {
		t.Fatalf("kept players = %v, want %v", kept, want)
	}

	if result.Before != len(players) {
		t.Fatalf("Before = %d, want %d", result.Before, len(players))
	}
	if result.After != len(want) {
		t.Fatalf("After = %d, want %d", result.After, len(want))
	}
	if result.Removed != len(players)-len(want) {
		t.Fatalf("Removed = %d, want %d", result.Removed, len(players)-len(want))
	}
	if result.KeptRanked != 2 {
		t.Fatalf("KeptRanked = %d, want 2", result.KeptRanked)
	}
	if result.KeptRankless != 1 {
		t.Fatalf("KeptRankless = %d, want 1", result.KeptRankless)
	}
	if result.KeptListed != 2 {
		t.Fatalf("KeptListed = %d, want 2", result.KeptListed)
	}
}

func TestPrunePlayerCacheHonoursConfiguredCutoff(t *testing.T) {
	players := []Player{
		{ID: "1", Name: "Bijan Robinson", Team: "ATL", Position: "RB", SearchRank: intPtr(3)},
		{ID: "2", Name: "Mid Round", Team: "SF", Position: "WR", SearchRank: intPtr(400)},
	}
	cfg := prunableConfig(t, 300, players, nil, nil)

	result, err := PrunePlayerCache(context.Background(), cfg, prunePlayerService(t), PruneOptions{})
	if err != nil {
		t.Fatalf("PrunePlayerCache returned error: %v", err)
	}

	if result.Cutoff != 300 {
		t.Fatalf("Cutoff = %d, want 300", result.Cutoff)
	}
	if kept := keptPlayerIDs(t, cfg.PlayersPath); strings.Join(kept, ",") != "1" {
		t.Fatalf("kept players = %v, want [1]", kept)
	}
}

func TestPrunePlayerCachePreservesCachedAt(t *testing.T) {
	cachedAt := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	players := []Player{
		{ID: "1", Name: "Bijan Robinson", Team: "ATL", Position: "RB", SearchRank: intPtr(3)},
		{ID: "2", Name: "Retired Guy", Team: "", Position: "LB", SearchRank: intPtr(9999999)},
	}
	cfg := prunableConfig(t, 500, players, nil, nil)
	if err := writePlayerCache(cfg.PlayersPath, playerCacheFile{CachedAt: cachedAt, Players: players}); err != nil {
		t.Fatalf("write players cache: %v", err)
	}
	if err := os.Chtimes(cfg.PlayersPath, cachedAt, cachedAt); err != nil {
		t.Fatalf("set cache time: %v", err)
	}

	service := prunePlayerService(t)
	service.Now = func() time.Time { return cachedAt.Add(time.Hour) }

	if _, err := PrunePlayerCache(context.Background(), cfg, service, PruneOptions{}); err != nil {
		t.Fatalf("PrunePlayerCache returned error: %v", err)
	}

	cache, err := readPlayerCacheFile(cfg.PlayersPath)
	if err != nil {
		t.Fatalf("readPlayerCacheFile returned error: %v", err)
	}
	if !cache.CachedAt.Equal(cachedAt) {
		t.Fatalf("CachedAt = %s, want %s", cache.CachedAt, cachedAt)
	}
}

func TestPrunePlayerCacheDryRunLeavesCacheUnchanged(t *testing.T) {
	players := []Player{
		{ID: "1", Name: "Bijan Robinson", Team: "ATL", Position: "RB", SearchRank: intPtr(3)},
		{ID: "2", Name: "Retired Guy", Team: "", Position: "LB", SearchRank: intPtr(9999999)},
	}
	cfg := prunableConfig(t, 500, players, nil, nil)

	before, err := os.ReadFile(cfg.PlayersPath)
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}

	result, err := PrunePlayerCache(context.Background(), cfg, prunePlayerService(t), PruneOptions{DryRun: true})
	if err != nil {
		t.Fatalf("PrunePlayerCache returned error: %v", err)
	}
	if result.After != 1 || result.Removed != 1 {
		t.Fatalf("result = %#v, want after=1 removed=1", result)
	}

	after, err := os.ReadFile(cfg.PlayersPath)
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	if string(before) != string(after) {
		t.Fatal("dry run rewrote the players cache")
	}
	if !strings.Contains(result.Summary(), "Prune dry run for") {
		t.Fatalf("Summary() = %q, want dry run wording", result.Summary())
	}
}

func TestPrunePlayerCacheRejectsNonPositiveConfiguredCutoff(t *testing.T) {
	cfg := prunableConfig(t, 0, []Player{
		{ID: "1", Name: "Bijan Robinson", Team: "ATL", Position: "RB", SearchRank: intPtr(3)},
	}, nil, nil)

	if _, err := PrunePlayerCache(context.Background(), cfg, prunePlayerService(t), PruneOptions{}); err == nil {
		t.Fatal("PrunePlayerCache returned nil error for prune_rank_cutoff 0")
	}
}

func TestPruneResultSummary(t *testing.T) {
	result := PruneResult{
		Path:          "players.json",
		Cutoff:        500,
		CutoffApplied: true,
		Before:        12221,
		After:         1175,
		Removed:       11046,
		KeptRanked:    1128,
		KeptRankless:  32,
		KeptListed:    15,
	}

	want := "Pruned players.json: cutoff=500 before=12221 after=1175 removed=11046 " +
		"kept_ranked=1128 kept_rankless=32 kept_listed=15"
	if got := result.Summary(); got != want {
		t.Fatalf("Summary() = %q, want %q", got, want)
	}
}

func TestPrunePlayerCacheIgnoresCutoffWhenRankingsSupplied(t *testing.T) {
	players := []Player{
		{ID: "1", Name: "Bijan Robinson", Team: "ATL", Position: "RB", SearchRank: intPtr(3)},
		{ID: "2", Name: "Deep Sleeper", Team: "NYJ", Position: "WR", SearchRank: intPtr(1200)},
		{ID: "3", Name: "Top Pick", Team: "CIN", Position: "WR", SearchRank: intPtr(1)},
		{ID: "ARI", Name: "Arizona Cardinals", Team: "ARI", Position: "DEF", SearchRank: nil},
	}
	cfg := prunableConfig(t, 500, players,
		[]string{"Deep Sleeper,WR,NYJ"},
		nil,
	)

	result, err := PrunePlayerCache(context.Background(), cfg, prunePlayerService(t), PruneOptions{})
	if err != nil {
		t.Fatalf("PrunePlayerCache returned error: %v", err)
	}

	// "Top Pick" and "Bijan Robinson" both beat the cutoff, but the custom
	// rankings replace it, so only the ranked player and the defense survive.
	kept := keptPlayerIDs(t, cfg.PlayersPath)
	want := []string{"2", "ARI"}
	if strings.Join(kept, ",") != strings.Join(want, ",") {
		t.Fatalf("kept players = %v, want %v", kept, want)
	}
	if result.CutoffApplied {
		t.Fatal("CutoffApplied = true, want false when rankings are supplied")
	}
	if result.KeptRanked != 0 {
		t.Fatalf("KeptRanked = %d, want 0", result.KeptRanked)
	}
	if !strings.Contains(result.Summary(), "cutoff=ignored") {
		t.Fatalf("Summary() = %q, want cutoff=ignored", result.Summary())
	}
}

func TestPrunePlayerCacheIgnoresInvalidCutoffWhenRankingsSupplied(t *testing.T) {
	players := []Player{
		{ID: "1", Name: "Bijan Robinson", Team: "ATL", Position: "RB", SearchRank: intPtr(3)},
		{ID: "2", Name: "Deep Sleeper", Team: "NYJ", Position: "WR", SearchRank: intPtr(1200)},
	}
	cfg := prunableConfig(t, 0, players, []string{"Deep Sleeper,WR,NYJ"}, nil)

	result, err := PrunePlayerCache(context.Background(), cfg, prunePlayerService(t), PruneOptions{})
	if err != nil {
		t.Fatalf("PrunePlayerCache returned error: %v", err)
	}
	if result.After != 1 || result.KeptListed != 1 {
		t.Fatalf("result = %#v, want after=1 kept_listed=1", result)
	}
}

func TestPrunePlayerCacheAppliesCutoffWhenOnlyWishlistSupplied(t *testing.T) {
	players := []Player{
		{ID: "1", Name: "Bijan Robinson", Team: "ATL", Position: "RB", SearchRank: intPtr(3)},
		{ID: "2", Name: "Deep Sleeper", Team: "NYJ", Position: "WR", SearchRank: intPtr(1200)},
	}
	cfg := prunableConfig(t, 500, players, nil, []string{"Deep Sleeper,WR,NYJ"})

	result, err := PrunePlayerCache(context.Background(), cfg, prunePlayerService(t), PruneOptions{})
	if err != nil {
		t.Fatalf("PrunePlayerCache returned error: %v", err)
	}
	if !result.CutoffApplied {
		t.Fatal("CutoffApplied = false, want true when only a wishlist is supplied")
	}
	if kept := keptPlayerIDs(t, cfg.PlayersPath); strings.Join(kept, ",") != "1,2" {
		t.Fatalf("kept players = %v, want [1 2]", kept)
	}
}

func prunableConfig(t *testing.T, cutoff int, players []Player, rankings, wishlist []string) Config {
	t.Helper()

	dir := t.TempDir()
	cachePath := filepath.Join(dir, "players.json")
	if err := writePlayerCache(cachePath, playerCacheFile{
		CachedAt: time.Now().UTC(),
		Players:  players,
	}); err != nil {
		t.Fatalf("write players cache: %v", err)
	}

	return Config{
		Sport:           "nfl",
		PruneRankCutoff: cutoff,
		PlayersPath:     cachePath,
		RankingsPath:    writePlayerListFixture(t, filepath.Join(dir, "rankings.csv"), rankings),
		WishlistPath:    writePlayerListFixture(t, filepath.Join(dir, "wishlist.csv"), wishlist),
	}
}

func writePlayerListFixture(t *testing.T, path string, rows []string) string {
	t.Helper()

	lines := append([]string{strings.Join(requiredPlayerListHeader, ",")}, rows...)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write player list %q: %v", path, err)
	}
	return path
}

func prunePlayerService(t *testing.T) PlayerService {
	t.Helper()

	return testPlayerService(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP request while pruning a fresh cache")
		return nil, nil
	})
}

func keptPlayerIDs(t *testing.T, path string) []string {
	t.Helper()

	cache, err := readPlayerCacheFile(path)
	if err != nil {
		t.Fatalf("readPlayerCacheFile returned error: %v", err)
	}

	ids := make([]string, 0, len(cache.Players))
	for _, player := range cache.Players {
		ids = append(ids, player.ID)
	}
	return ids
}

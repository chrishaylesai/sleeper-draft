package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPlayerServiceUsesFreshCache(t *testing.T) {
	cachePath := writePlayerCacheFixture(t, []Player{
		{ID: "1", Name: "Bijan Robinson", Team: "ATL", Position: "RB", SearchRank: intPtr(3)},
	}, time.Now())

	service := testPlayerService(func(r *http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP request for fresh cache")
		return nil, nil
	})

	db, err := service.Load(context.Background(), "nfl", cachePath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if db.Source != "cache" {
		t.Fatalf("Source = %q, want cache", db.Source)
	}
	if len(db.Players) != 1 || db.Players[0].ID != "1" {
		t.Fatalf("Players = %#v, want cached player", db.Players)
	}
}

func TestPlayerServiceFetchesAndCachesMissingCache(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "players.json")
	service := testPlayerService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/players/nfl" {
			t.Fatalf("path = %q, want /players/nfl", r.URL.Path)
		}
		return jsonResponse(http.StatusOK, `{
			"1": {"full_name": "Amon-Ra St. Brown", "team": "DET", "position": "WR", "search_rank": 4},
			"2": {"first_name": "Bijan", "last_name": "Robinson", "team": "ATL", "position": "RB", "search_rank": "3"}
		}`), nil
	})

	db, err := service.Load(context.Background(), "nfl", cachePath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if db.Source != "api" {
		t.Fatalf("Source = %q, want api", db.Source)
	}
	if len(db.Players) != 2 {
		t.Fatalf("len(Players) = %d, want 2", len(db.Players))
	}
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("expected cache file: %v", err)
	}

	player, err := db.Index.Lookup("bijan robinson", "rb", "atl")
	if err != nil {
		t.Fatalf("Lookup returned error: %v", err)
	}
	if player.ID != "2" || player.SearchRank == nil || *player.SearchRank != 3 {
		t.Fatalf("Lookup player = %#v, want ID 2 rank 3", player)
	}
}

func TestPlayerServiceRefreshesStaleCache(t *testing.T) {
	cachePath := writePlayerCacheFixture(t, []Player{
		{ID: "old", Name: "Old Player", Team: "FA", Position: "WR"},
	}, time.Now().Add(-25*time.Hour))

	service := testPlayerService(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{
			"new": {"full_name": "New Player", "team": "DAL", "position": "WR", "search_rank": 99}
		}`), nil
	})

	db, err := service.Load(context.Background(), "nfl", cachePath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if db.Source != "api" {
		t.Fatalf("Source = %q, want api", db.Source)
	}
	if len(db.Players) != 1 || db.Players[0].ID != "new" {
		t.Fatalf("Players = %#v, want refreshed player", db.Players)
	}
}

func TestPlayerServiceRejectsUnexpectedStatus(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "players.json")
	service := testPlayerService(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusTooManyRequests, "nope"), nil
	})

	_, err := service.Load(context.Background(), "nfl", cachePath)
	if err == nil {
		t.Fatal("Load returned nil error")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Fatalf("error = %q, want status code", err.Error())
	}
}

func TestPlayerIndexLookupRequiresTeamForAmbiguousMatch(t *testing.T) {
	idx := NewPlayerIndex([]Player{
		{ID: "1", Name: "Duplicate Name", Team: "AAA", Position: "WR"},
		{ID: "2", Name: "Duplicate Name", Team: "BBB", Position: "WR"},
	})

	if _, err := idx.Lookup("Duplicate Name", "WR", ""); err == nil {
		t.Fatal("Lookup returned nil error for ambiguous player")
	}

	player, err := idx.Lookup("Duplicate Name", "WR", "BBB")
	if err != nil {
		t.Fatalf("Lookup returned error: %v", err)
	}
	if player.ID != "2" {
		t.Fatalf("ID = %q, want 2", player.ID)
	}
}

func testPlayerService(roundTrip func(*http.Request) (*http.Response, error)) PlayerService {
	return PlayerService{
		Client:      &http.Client{Transport: roundTripFunc(roundTrip)},
		BaseURL:     "https://sleeper.test",
		CacheMaxAge: playersCacheMaxAge,
		Now:         time.Now,
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     strconv.Itoa(status) + " " + http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func writePlayerCacheFixture(t *testing.T, players []Player, modTime time.Time) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "players.json")
	data, err := json.Marshal(playerCacheFile{
		CachedAt: modTime.UTC(),
		Players:  players,
	})
	if err != nil {
		t.Fatalf("marshal cache: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("set cache time: %v", err)
	}
	return path
}

func intPtr(value int) *int {
	return &value
}

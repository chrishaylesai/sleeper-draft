package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	sleeperAPIBaseURL  = "https://api.sleeper.app/v1"
	playersCacheMaxAge = 24 * time.Hour
)

type Player struct {
	ID         string `json:"player_id"`
	Name       string `json:"name"`
	Team       string `json:"team"`
	Position   string `json:"position"`
	SearchRank *int   `json:"search_rank"`
}

type PlayerDatabase struct {
	Players []Player
	Index   PlayerIndex
	Source  string
}

type PlayerIndex struct {
	byNamePosition map[string][]Player
}

type PlayerService struct {
	Client      *http.Client
	BaseURL     string
	CacheMaxAge time.Duration
	Now         func() time.Time
}

type playerCacheFile struct {
	CachedAt time.Time `json:"cached_at"`
	Players  []Player  `json:"players"`
}

type sleeperPlayer struct {
	FullName       string          `json:"full_name"`
	FirstName      string          `json:"first_name"`
	LastName       string          `json:"last_name"`
	SearchFullName string          `json:"search_full_name"`
	Team           string          `json:"team"`
	Position       string          `json:"position"`
	SearchRank     json.RawMessage `json:"search_rank"`
}

func NewPlayerService(client *http.Client) PlayerService {
	return PlayerService{
		Client:      client,
		BaseURL:     sleeperAPIBaseURL,
		CacheMaxAge: playersCacheMaxAge,
		Now:         time.Now,
	}
}

func (s PlayerService) Load(ctx context.Context, sport, cachePath string) (PlayerDatabase, error) {
	s = s.withDefaults()

	if fresh, err := s.cacheIsFresh(cachePath); err != nil {
		return PlayerDatabase{}, err
	} else if fresh {
		db, err := readPlayerCache(cachePath)
		if err != nil {
			return PlayerDatabase{}, err
		}
		db.Source = "cache"
		return db, nil
	}

	players, err := s.fetchPlayers(ctx, sport)
	if err != nil {
		return PlayerDatabase{}, err
	}
	if err := writePlayerCache(cachePath, playerCacheFile{
		CachedAt: s.Now().UTC(),
		Players:  players,
	}); err != nil {
		return PlayerDatabase{}, err
	}

	return PlayerDatabase{
		Players: players,
		Index:   NewPlayerIndex(players),
		Source:  "api",
	}, nil
}

func (s PlayerService) withDefaults() PlayerService {
	if s.Client == nil {
		s.Client = http.DefaultClient
	}
	if s.BaseURL == "" {
		s.BaseURL = sleeperAPIBaseURL
	}
	if s.CacheMaxAge == 0 {
		s.CacheMaxAge = playersCacheMaxAge
	}
	if s.Now == nil {
		s.Now = time.Now
	}
	return s
}

func (s PlayerService) cacheIsFresh(path string) (bool, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat player cache %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("player cache %q is not a regular file", path)
	}
	return s.Now().Sub(info.ModTime()) < s.CacheMaxAge, nil
}

func (s PlayerService) fetchPlayers(ctx context.Context, sport string) ([]Player, error) {
	endpoint := fmt.Sprintf("%s/players/%s", strings.TrimRight(s.BaseURL, "/"), sport)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create players request: %w", err)
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch players: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch players: unexpected status %s", resp.Status)
	}

	var raw map[string]sleeperPlayer
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode players response: %w", err)
	}

	players := make([]Player, 0, len(raw))
	for id, sleeper := range raw {
		name := playerName(sleeper)
		if strings.TrimSpace(id) == "" || name == "" {
			continue
		}

		searchRank, err := parseSearchRank(sleeper.SearchRank)
		if err != nil {
			return nil, fmt.Errorf("parse search_rank for player %s: %w", id, err)
		}

		players = append(players, Player{
			ID:         strings.TrimSpace(id),
			Name:       name,
			Team:       strings.TrimSpace(sleeper.Team),
			Position:   strings.TrimSpace(sleeper.Position),
			SearchRank: searchRank,
		})
	}

	sort.Slice(players, func(i, j int) bool {
		return players[i].ID < players[j].ID
	})
	return players, nil
}

func playerName(player sleeperPlayer) string {
	if name := strings.TrimSpace(player.FullName); name != "" {
		return name
	}
	if name := strings.TrimSpace(player.SearchFullName); name != "" {
		return name
	}
	return strings.TrimSpace(strings.Join([]string{
		strings.TrimSpace(player.FirstName),
		strings.TrimSpace(player.LastName),
	}, " "))
}

func parseSearchRank(raw json.RawMessage) (*int, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var value any
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}

	switch v := value.(type) {
	case json.Number:
		if i, err := v.Int64(); err == nil {
			rank := int(i)
			return &rank, nil
		}
		f, err := v.Float64()
		if err != nil {
			return nil, err
		}
		rank := int(f)
		return &rank, nil
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, nil
		}
		i, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		return &i, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", value)
	}
}

func readPlayerCache(path string) (PlayerDatabase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PlayerDatabase{}, fmt.Errorf("read player cache %q: %w", path, err)
	}

	var cache playerCacheFile
	if err := json.Unmarshal(data, &cache); err != nil {
		return PlayerDatabase{}, fmt.Errorf("parse player cache %q: %w", path, err)
	}

	return PlayerDatabase{
		Players: cache.Players,
		Index:   NewPlayerIndex(cache.Players),
		Source:  "cache",
	}, nil
}

func writePlayerCache(path string, cache playerCacheFile) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create player cache directory %q: %w", dir, err)
		}
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("encode player cache: %w", err)
	}
	data = append(data, '\n')

	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o644); err != nil {
		return fmt.Errorf("write player cache temp file %q: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace player cache %q: %w", path, err)
	}
	return nil
}

func NewPlayerIndex(players []Player) PlayerIndex {
	index := PlayerIndex{byNamePosition: make(map[string][]Player)}
	for _, player := range players {
		if strings.TrimSpace(player.Name) == "" || strings.TrimSpace(player.Position) == "" {
			continue
		}
		key := playerLookupKey(player.Name, player.Position)
		index.byNamePosition[key] = append(index.byNamePosition[key], player)
	}
	return index
}

func (idx PlayerIndex) Lookup(name, position, team string) (Player, error) {
	name = strings.TrimSpace(name)
	position = strings.TrimSpace(position)
	team = strings.TrimSpace(team)
	if name == "" {
		return Player{}, errors.New("player name is required")
	}
	if position == "" {
		return Player{}, errors.New("player position is required")
	}

	candidates := idx.byNamePosition[playerLookupKey(name, position)]
	if team != "" {
		filtered := make([]Player, 0, len(candidates))
		for _, candidate := range candidates {
			if normalizeLookupText(candidate.Team) == normalizeLookupText(team) {
				filtered = append(filtered, candidate)
			}
		}
		candidates = filtered
	}

	switch len(candidates) {
	case 0:
		return Player{}, fmt.Errorf("no player matched name=%q position=%q team=%q", name, position, team)
	case 1:
		return candidates[0], nil
	default:
		return Player{}, fmt.Errorf("multiple players matched name=%q position=%q; provide player_team", name, position)
	}
}

func playerLookupKey(name, position string) string {
	return normalizeLookupText(name) + "\x00" + normalizeLookupText(position)
}

func normalizeLookupText(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

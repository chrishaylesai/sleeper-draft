package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type AppState struct {
	Config   Config
	Players  PlayerDatabase
	Rankings PlayerList
	Wishlist PlayerList
	Drafts   DraftService
	DraftID  string
}

func LoadAppState(ctx context.Context, cfg Config, client *http.Client) (AppState, error) {
	players, err := NewPlayerService(client).Load(ctx, cfg.Sport, cfg.PlayersPath)
	if err != nil {
		return AppState{}, err
	}

	rankings, wishlist, err := LoadPlayerLists(cfg, players)
	if err != nil {
		return AppState{}, err
	}

	drafts := NewDraftService(client)
	draft, err := drafts.Resolve(ctx, cfg)
	if err != nil {
		return AppState{}, err
	}

	return AppState{
		Config:   cfg,
		Players:  players,
		Rankings: rankings,
		Wishlist: wishlist,
		Drafts:   drafts,
		DraftID:  draft.ID,
	}, nil
}

func (s AppState) TextStartupSummary() string {
	return fmt.Sprintf(
		"%s\nPlayers loaded: %d source=%s\nPlayer lists loaded: rankings=%d wishlist=%d",
		s.Config.StartupSummary(),
		len(s.Players.Players),
		s.Players.Source,
		len(s.Rankings.Entries),
		len(s.Wishlist.Entries),
	)
}

func (s AppState) SnapshotSummary(snapshot DraftSnapshot) string {
	return fmt.Sprintf(
		"%s\n%s\n%s\n%s",
		snapshot.Summary(),
		FormatPositionSummaries(BuildPositionSummaries(s.Config.PositionTargets, snapshot.Picks, s.Players)),
		FormatBestAvailable(BestAvailableByPosition(s.Config, s.Players, s.Rankings, snapshot.Picks)),
		FormatWishlistReport(BuildWishlistReport(s.Config, s.Wishlist, snapshot.Picks)),
	)
}

func (s AppState) RefreshInterval() time.Duration {
	return time.Duration(s.Config.RefreshIntervalSeconds) * time.Second
}

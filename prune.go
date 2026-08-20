package main

import (
	"context"
	"fmt"
)

// ranklessKeepPositions lists positions that Sleeper does not give a
// `search_rank`, but that are still draftable. Team defenses are the notable
// case: every `DEF` entry has a null `search_rank`.
var ranklessKeepPositions = map[string]struct{}{
	"DEF": {},
}

type PruneOptions struct {
	// DryRun reports what would be removed without rewriting the cache.
	DryRun bool
}

type PruneResult struct {
	Path string
	// Cutoff is the configured `prune_rank_cutoff`. It is only meaningful when
	// CutoffApplied is true.
	Cutoff int
	// CutoffApplied is false when custom rankings replaced the cutoff as the
	// definition of a relevant player.
	CutoffApplied bool
	DryRun        bool
	Before        int
	After         int
	Removed       int
	KeptRanked    int
	KeptRankless  int
	KeptListed    int
}

// PrunePlayerCache rewrites the local players cache, keeping only players that
// are fantasy relevant.
//
// Every player referenced by the configured rankings or wishlist CSVs is kept,
// as is every position Sleeper leaves unranked but that is still draftable (see
// ranklessKeepPositions). The rest of the pool is decided by Sleeper's
// `search_rank`: players ranked at or above `prune_rank_cutoff` survive.
//
// Supplying a custom rankings list turns that last rule off entirely. Rankings
// are the app's primary ordering, so when they exist they — not Sleeper's
// `search_rank` — define which players matter, and the cutoff is ignored.
func PrunePlayerCache(ctx context.Context, cfg Config, players PlayerService, opts PruneOptions) (PruneResult, error) {
	if _, err := players.Load(ctx, cfg.Sport, cfg.PlayersPath); err != nil {
		return PruneResult{}, err
	}

	cache, err := readPlayerCacheFile(cfg.PlayersPath)
	if err != nil {
		return PruneResult{}, err
	}

	listed, err := listedPlayers(cfg, cache.Players)
	if err != nil {
		return PruneResult{}, err
	}

	cutoff := cfg.PruneRankCutoff
	cutoffApplied := !listed.rankingsSupplied
	if cutoffApplied && cutoff <= 0 {
		return PruneResult{}, fmt.Errorf("prune_rank_cutoff must be greater than 0, got %d", cutoff)
	}

	result := PruneResult{
		Path:          cfg.PlayersPath,
		Cutoff:        cutoff,
		CutoffApplied: cutoffApplied,
		DryRun:        opts.DryRun,
		Before:        len(cache.Players),
	}

	kept := make([]Player, 0, len(cache.Players))
	for _, player := range cache.Players {
		switch relevance := playerRelevance(player, listed, cutoff, cutoffApplied); relevance {
		case relevanceListed:
			result.KeptListed++
		case relevanceRanked:
			result.KeptRanked++
		case relevanceRankless:
			result.KeptRankless++
		default:
			continue
		}
		kept = append(kept, player)
	}

	result.After = len(kept)
	result.Removed = result.Before - result.After

	if opts.DryRun {
		return result, nil
	}

	if err := writePlayerCache(cfg.PlayersPath, playerCacheFile{
		CachedAt: cache.CachedAt,
		Players:  kept,
	}); err != nil {
		return PruneResult{}, err
	}
	return result, nil
}

func (r PruneResult) Summary() string {
	action := "Pruned"
	if r.DryRun {
		action = "Prune dry run for"
	}

	cutoff := "ignored"
	if r.CutoffApplied {
		cutoff = fmt.Sprintf("%d", r.Cutoff)
	}

	return fmt.Sprintf(
		"%s %s: cutoff=%s before=%d after=%d removed=%d kept_ranked=%d kept_rankless=%d kept_listed=%d",
		action,
		r.Path,
		cutoff,
		r.Before,
		r.After,
		r.Removed,
		r.KeptRanked,
		r.KeptRankless,
		r.KeptListed,
	)
}

func keepWhenRankless(player Player) bool {
	_, keep := ranklessKeepPositions[normalizePosition(player.Position)]
	return keep
}

type relevanceReason int

const (
	relevanceNone relevanceReason = iota
	relevanceListed
	relevanceRanked
	relevanceRankless
)

func playerRelevance(player Player, listed listedPlayerSet, cutoff int, cutoffApplied bool) relevanceReason {
	switch {
	case listed.ids[normalizeLookupText(player.ID)]:
		return relevanceListed
	case cutoffApplied && player.SearchRank != nil && *player.SearchRank <= cutoff:
		return relevanceRanked
	case player.SearchRank == nil && keepWhenRankless(player):
		return relevanceRankless
	default:
		return relevanceNone
	}
}

func relevantPlayers(cfg Config, players []Player, rankings, wishlist PlayerList) []Player {
	listed := listedPlayersFromLists(rankings, wishlist)
	cutoffApplied := !listed.rankingsSupplied

	relevant := make([]Player, 0, len(players))
	for _, player := range players {
		if playerRelevance(player, listed, cfg.PruneRankCutoff, cutoffApplied) == relevanceNone {
			continue
		}
		relevant = append(relevant, player)
	}
	return relevant
}

type listedPlayerSet struct {
	ids              map[string]bool
	rankingsSupplied bool
}

// listedPlayers resolves every rankings and wishlist row against the unpruned
// player set, so those players survive whatever else is removed.
func listedPlayers(cfg Config, players []Player) (listedPlayerSet, error) {
	db := PlayerDatabase{Players: players, Index: NewPlayerIndex(players)}
	rankings, wishlist, err := LoadPlayerLists(cfg, db)
	if err != nil {
		return listedPlayerSet{}, err
	}

	return listedPlayersFromLists(rankings, wishlist), nil
}

func listedPlayersFromLists(rankings, wishlist PlayerList) listedPlayerSet {
	listed := listedPlayerSet{
		ids:              make(map[string]bool, len(rankings.Entries)+len(wishlist.Entries)),
		rankingsSupplied: len(rankings.Entries) > 0,
	}
	for _, list := range []PlayerList{rankings, wishlist} {
		for _, entry := range list.Entries {
			if id := normalizeLookupText(entry.PlayerID); id != "" {
				listed.ids[id] = true
			}
		}
	}
	return listed
}

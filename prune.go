package main

import (
	"context"
	"fmt"
)

// defaultPruneSearchRank is the suggested Sleeper `search_rank` cutoff. It keeps
// a deep fantasy-relevant pool while dropping most of the player database.
const defaultPruneSearchRank = 500

// ranklessKeepPositions lists positions that Sleeper does not give a
// `search_rank`, but that are still draftable. Team defenses are the notable
// case: every `DEF` entry has a null `search_rank`.
var ranklessKeepPositions = map[string]struct{}{
	"DEF": {},
}

type PruneOptions struct {
	// Cutoff is the highest (worst) Sleeper `search_rank` to keep.
	Cutoff int
	// DryRun reports what would be removed without rewriting the cache.
	DryRun bool
}

type PruneResult struct {
	Path         string
	Cutoff       int
	DryRun       bool
	Before       int
	After        int
	Removed      int
	KeptRanked   int
	KeptRankless int
	KeptListed   int
}

// PrunePlayerCache rewrites the local players cache, keeping only players that
// are fantasy relevant: those ranked at or above the cutoff, positions Sleeper
// leaves unranked but that are still draftable (see ranklessKeepPositions), and
// every player referenced by the configured rankings or wishlist CSVs.
func PrunePlayerCache(ctx context.Context, cfg Config, players PlayerService, opts PruneOptions) (PruneResult, error) {
	if opts.Cutoff <= 0 {
		return PruneResult{}, fmt.Errorf("prune rank cutoff must be greater than 0, got %d", opts.Cutoff)
	}

	if _, err := players.Load(ctx, cfg.Sport, cfg.PlayersPath); err != nil {
		return PruneResult{}, err
	}

	cache, err := readPlayerCacheFile(cfg.PlayersPath)
	if err != nil {
		return PruneResult{}, err
	}

	protected, err := listedPlayerIDs(cfg, cache.Players)
	if err != nil {
		return PruneResult{}, err
	}

	result := PruneResult{
		Path:   cfg.PlayersPath,
		Cutoff: opts.Cutoff,
		DryRun: opts.DryRun,
		Before: len(cache.Players),
	}

	kept := make([]Player, 0, len(cache.Players))
	for _, player := range cache.Players {
		switch {
		case protected[normalizeLookupText(player.ID)]:
			result.KeptListed++
		case player.SearchRank != nil && *player.SearchRank <= opts.Cutoff:
			result.KeptRanked++
		case player.SearchRank == nil && keepWhenRankless(player):
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

	return fmt.Sprintf(
		"%s %s: cutoff=%d before=%d after=%d removed=%d kept_ranked=%d kept_rankless=%d kept_listed=%d",
		action,
		r.Path,
		r.Cutoff,
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

// listedPlayerIDs resolves every rankings and wishlist row against the unpruned
// player set so those players survive the cutoff.
func listedPlayerIDs(cfg Config, players []Player) (map[string]bool, error) {
	db := PlayerDatabase{Players: players, Index: NewPlayerIndex(players)}
	rankings, wishlist, err := LoadPlayerLists(cfg, db)
	if err != nil {
		return nil, err
	}

	listed := make(map[string]bool, len(rankings.Entries)+len(wishlist.Entries))
	for _, list := range []PlayerList{rankings, wishlist} {
		for _, entry := range list.Entries {
			if id := normalizeLookupText(entry.PlayerID); id != "" {
				listed[id] = true
			}
		}
	}
	return listed, nil
}

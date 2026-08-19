package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

const defaultConfigPath = "config.json"

func main() {
	configPath := flag.String("config", defaultConfigPath, "path to config JSON file")
	poll := flag.Bool("poll", false, "poll the draft continuously using refresh_interval_seconds")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(cfg.StartupSummary())

	players, err := NewPlayerService(http.DefaultClient).Load(context.Background(), cfg.Sport, cfg.PlayersPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Players loaded: %d source=%s\n", len(players.Players), players.Source)

	rankings, wishlist, err := LoadPlayerLists(cfg, players)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Player lists loaded: rankings=%d wishlist=%d\n", len(rankings.Entries), len(wishlist.Entries))

	drafts := NewDraftService(http.DefaultClient)
	draft, err := drafts.Resolve(context.Background(), cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *poll {
		interval := time.Duration(cfg.RefreshIntervalSeconds) * time.Second
		err := drafts.Poll(context.Background(), draft.ID, interval, func(snapshot DraftSnapshot, err error) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "poll error: %v\n", err)
				return
			}
			fmt.Println(snapshot.Summary())
			fmt.Println(FormatPositionSummaries(BuildPositionSummaries(cfg.PositionTargets, snapshot.Picks, players)))
			fmt.Println(FormatBestAvailable(BestAvailableByPosition(cfg, players, rankings, snapshot.Picks)))
			fmt.Println(FormatWishlistReport(BuildWishlistReport(cfg, wishlist, snapshot.Picks)))
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	snapshot, err := drafts.Snapshot(context.Background(), draft.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(snapshot.Summary())
	fmt.Println(FormatPositionSummaries(BuildPositionSummaries(cfg.PositionTargets, snapshot.Picks, players)))
	fmt.Println(FormatBestAvailable(BestAvailableByPosition(cfg, players, rankings, snapshot.Picks)))
	fmt.Println(FormatWishlistReport(BuildWishlistReport(cfg, wishlist, snapshot.Picks)))
}

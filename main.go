package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
)

const defaultConfigPath = "config.json"

func main() {
	configPath := flag.String("config", defaultConfigPath, "path to config JSON file")
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
}

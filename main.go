package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

const defaultConfigPath = "config.json"

func main() {
	configPath := flag.String("config", defaultConfigPath, "path to config JSON file")
	once := flag.Bool("once", false, "print one snapshot and exit instead of launching the TUI")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	state, err := LoadAppState(context.Background(), cfg, http.DefaultClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *once {
		fmt.Println(state.TextStartupSummary())
		snapshot, err := state.Drafts.Snapshot(context.Background(), state.DraftID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(state.SnapshotSummary(snapshot))
		return
	}

	if _, err := tea.NewProgram(NewDashboardModel(state), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

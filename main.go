package main

import (
	"flag"
	"fmt"
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
}

package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var requiredPlayerListHeader = []string{"player_name", "player_position", "player_team"}

type RankedPlayer struct {
	Rank     int
	PlayerID string
	Player   Player
	Name     string
	Position string
	Team     string
}

type PlayerList struct {
	Path    string
	Entries []RankedPlayer
}

type PlayerListImportError struct {
	Path     string
	Problems []string
}

func (e PlayerListImportError) Error() string {
	return fmt.Sprintf("import %s: %s", e.Path, strings.Join(e.Problems, "; "))
}

func LoadPlayerLists(cfg Config, players PlayerDatabase) (rankings PlayerList, wishlist PlayerList, err error) {
	rankings, err = ImportPlayerList(cfg.RankingsPath, players.Index)
	if err != nil {
		return PlayerList{}, PlayerList{}, err
	}

	wishlist, err = ImportPlayerList(cfg.WishlistPath, players.Index)
	if err != nil {
		return PlayerList{}, PlayerList{}, err
	}

	return rankings, wishlist, nil
}

func ImportPlayerList(path string, index PlayerIndex) (PlayerList, error) {
	file, err := os.Open(path)
	if err != nil {
		return PlayerList{}, fmt.Errorf("open player list %q: %w", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if errors.Is(err, io.EOF) {
		return PlayerList{}, fmt.Errorf("import %s: missing header", path)
	}
	if err != nil {
		return PlayerList{}, fmt.Errorf("read header %q: %w", path, err)
	}
	if err := validatePlayerListHeader(header); err != nil {
		return PlayerList{}, fmt.Errorf("import %s: %w", path, err)
	}
	reader.FieldsPerRecord = len(requiredPlayerListHeader)

	var entries []RankedPlayer
	var problems []string
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return PlayerList{}, fmt.Errorf("read %s: %w", path, err)
		}

		line, _ := reader.FieldPos(0)
		name := strings.TrimSpace(record[0])
		position := strings.TrimSpace(record[1])
		team := strings.TrimSpace(record[2])

		player, err := index.Lookup(name, position, team)
		if err != nil {
			problems = append(problems, fmt.Sprintf("line %d: %v", line, err))
			continue
		}

		entries = append(entries, RankedPlayer{
			Rank:     len(entries) + 1,
			PlayerID: player.ID,
			Player:   player,
			Name:     name,
			Position: position,
			Team:     team,
		})
	}

	if len(problems) > 0 {
		return PlayerList{}, PlayerListImportError{Path: path, Problems: problems}
	}

	return PlayerList{Path: path, Entries: entries}, nil
}

func validatePlayerListHeader(header []string) error {
	if len(header) != len(requiredPlayerListHeader) {
		return fmt.Errorf("header must be %s", strings.Join(requiredPlayerListHeader, ","))
	}

	for i, want := range requiredPlayerListHeader {
		got := strings.TrimSpace(header[i])
		if i == 0 {
			got = strings.TrimPrefix(got, "\ufeff")
		}
		if got != want {
			return fmt.Errorf("header column %d must be %q, got %q", i+1, want, header[i])
		}
	}
	return nil
}

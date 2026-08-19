package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportPlayerListResolvesRowsInRankOrder(t *testing.T) {
	index := NewPlayerIndex([]Player{
		{ID: "1", Name: "Bijan Robinson", Team: "ATL", Position: "RB"},
		{ID: "2", Name: "Amon-Ra St. Brown", Team: "DET", Position: "WR"},
	})
	path := writePlayerList(t, `player_name,player_position,player_team
Bijan Robinson,RB,ATL
Amon-Ra St. Brown,WR,
`)

	list, err := ImportPlayerList(path, index)
	if err != nil {
		t.Fatalf("ImportPlayerList returned error: %v", err)
	}

	if len(list.Entries) != 2 {
		t.Fatalf("len(Entries) = %d, want 2", len(list.Entries))
	}
	if list.Entries[0].Rank != 1 || list.Entries[0].PlayerID != "1" {
		t.Fatalf("first entry = %#v, want rank 1 player 1", list.Entries[0])
	}
	if list.Entries[1].Rank != 2 || list.Entries[1].PlayerID != "2" {
		t.Fatalf("second entry = %#v, want rank 2 player 2", list.Entries[1])
	}
}

func TestImportPlayerListUsesTeamToDisambiguateDuplicates(t *testing.T) {
	index := NewPlayerIndex([]Player{
		{ID: "1", Name: "Duplicate Name", Team: "AAA", Position: "WR"},
		{ID: "2", Name: "Duplicate Name", Team: "BBB", Position: "WR"},
	})
	path := writePlayerList(t, `player_name,player_position,player_team
Duplicate Name,WR,BBB
`)

	list, err := ImportPlayerList(path, index)
	if err != nil {
		t.Fatalf("ImportPlayerList returned error: %v", err)
	}
	if list.Entries[0].PlayerID != "2" {
		t.Fatalf("PlayerID = %q, want 2", list.Entries[0].PlayerID)
	}
}

func TestImportPlayerListReportsUnmatchedRows(t *testing.T) {
	index := NewPlayerIndex([]Player{
		{ID: "1", Name: "Bijan Robinson", Team: "ATL", Position: "RB"},
	})
	path := writePlayerList(t, `player_name,player_position,player_team
Missing Player,RB,ATL
Bijan Robinson,WR,ATL
`)

	_, err := ImportPlayerList(path, index)
	if err == nil {
		t.Fatal("ImportPlayerList returned nil error")
	}
	if !strings.Contains(err.Error(), "line 2") || !strings.Contains(err.Error(), "Missing Player") {
		t.Fatalf("error = %q, want first unmatched row", err.Error())
	}
	if !strings.Contains(err.Error(), "line 3") || !strings.Contains(err.Error(), "Bijan Robinson") {
		t.Fatalf("error = %q, want second unmatched row", err.Error())
	}
}

func TestImportPlayerListRejectsAmbiguousRowsWithoutTeam(t *testing.T) {
	index := NewPlayerIndex([]Player{
		{ID: "1", Name: "Duplicate Name", Team: "AAA", Position: "WR"},
		{ID: "2", Name: "Duplicate Name", Team: "BBB", Position: "WR"},
	})
	path := writePlayerList(t, `player_name,player_position,player_team
Duplicate Name,WR,
`)

	_, err := ImportPlayerList(path, index)
	if err == nil {
		t.Fatal("ImportPlayerList returned nil error")
	}
	if !strings.Contains(err.Error(), "provide player_team") {
		t.Fatalf("error = %q, want disambiguation message", err.Error())
	}
}

func TestImportPlayerListRejectsInvalidHeader(t *testing.T) {
	index := NewPlayerIndex(nil)
	path := writePlayerList(t, `name,position,team
`)

	_, err := ImportPlayerList(path, index)
	if err == nil {
		t.Fatal("ImportPlayerList returned nil error")
	}
	if !strings.Contains(err.Error(), `header column 1 must be "player_name"`) {
		t.Fatalf("error = %q, want header message", err.Error())
	}
}

func TestLoadPlayerListsImportsRankingsAndWishlist(t *testing.T) {
	db := PlayerDatabase{Index: NewPlayerIndex([]Player{
		{ID: "1", Name: "Bijan Robinson", Team: "ATL", Position: "RB"},
		{ID: "2", Name: "Amon-Ra St. Brown", Team: "DET", Position: "WR"},
	})}
	rankingsPath := writePlayerList(t, `player_name,player_position,player_team
Bijan Robinson,RB,ATL
`)
	wishlistPath := writePlayerList(t, `player_name,player_position,player_team
Amon-Ra St. Brown,WR,DET
`)

	rankings, wishlist, err := LoadPlayerLists(Config{
		RankingsPath: rankingsPath,
		WishlistPath: wishlistPath,
	}, db)
	if err != nil {
		t.Fatalf("LoadPlayerLists returned error: %v", err)
	}
	if rankings.Entries[0].PlayerID != "1" || wishlist.Entries[0].PlayerID != "2" {
		t.Fatalf("rankings/wishlist = %#v/%#v, want resolved players", rankings, wishlist)
	}
}

func writePlayerList(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "players.csv")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write player list: %v", err)
	}
	return path
}

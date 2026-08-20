package main

import (
	"reflect"
	"testing"
)

func TestBuildWishlistReportTracksAvailableTakenAndExcluded(t *testing.T) {
	wishlist := PlayerList{Entries: []RankedPlayer{
		wishlistEntry(1, Player{ID: "taken", Name: "Taken RB", Team: "ATL", Position: "RB"}),
		wishlistEntry(2, Player{ID: "available", Name: "Available WR", Team: "DET", Position: "WR"}),
		wishlistEntry(3, Player{ID: "excluded", Name: "Excluded QB", Team: "BUF", Position: "QB"}),
	}}

	report := BuildWishlistReport(
		Config{ExcludedPlayers: []string{"excluded"}},
		wishlist,
		[]Pick{{PlayerID: "taken", PickedBy: "user123", RosterID: 7, PickNo: 5}},
	)

	gotStatuses := []WishlistStatus{report.Items[0].Status, report.Items[1].Status, report.Items[2].Status}
	wantStatuses := []WishlistStatus{WishlistTaken, WishlistAvailable, WishlistExcluded}
	if !reflect.DeepEqual(gotStatuses, wantStatuses) {
		t.Fatalf("statuses = %#v, want %#v", gotStatuses, wantStatuses)
	}
	if report.Items[0].TakenBy != "user123" || report.Items[0].PickNo != 5 {
		t.Fatalf("taken item = %#v, want user123 pick 5", report.Items[0])
	}
	if report.TopAvailable == nil || report.TopAvailable.Player.ID != "available" {
		t.Fatalf("TopAvailable = %#v, want available", report.TopAvailable)
	}
}

func TestBuildWishlistReportUsesRosterWhenPickedByMissing(t *testing.T) {
	wishlist := PlayerList{Entries: []RankedPlayer{
		wishlistEntry(1, Player{ID: "taken", Name: "Taken RB", Team: "ATL", Position: "RB"}),
	}}

	report := BuildWishlistReport(Config{}, wishlist, []Pick{{PlayerID: "taken", RosterID: 9, PickNo: 12}})

	if report.Items[0].TakenBy != "roster:9" {
		t.Fatalf("TakenBy = %q, want roster:9", report.Items[0].TakenBy)
	}
}

func TestBuildWishlistReportMarksPersonalTakenPlayer(t *testing.T) {
	wishlist := PlayerList{Entries: []RankedPlayer{
		wishlistEntry(1, Player{ID: "mine", Name: "My RB", Team: "ATL", Position: "RB"}),
		wishlistEntry(2, Player{ID: "theirs", Name: "Their WR", Team: "DET", Position: "WR"}),
	}}

	report := BuildWishlistReportForPersonal(
		Config{},
		wishlist,
		[]Pick{
			{PlayerID: "mine", PickedBy: "me", PickNo: 5},
			{PlayerID: "theirs", PickedBy: "other", PickNo: 6},
		},
		PersonalDraft{UserID: "me", Known: true},
	)

	if !report.Items[0].SelectedByMe {
		t.Fatalf("SelectedByMe = false for personal pick: %#v", report.Items[0])
	}
	if report.Items[1].SelectedByMe {
		t.Fatalf("SelectedByMe = true for other user's pick: %#v", report.Items[1])
	}
}

func TestBuildWishlistReportTopAvailableByPosition(t *testing.T) {
	wishlist := PlayerList{Entries: []RankedPlayer{
		wishlistEntry(1, Player{ID: "qb1", Name: "QB One", Team: "BUF", Position: "QB"}),
		wishlistEntry(2, Player{ID: "rb1", Name: "RB One", Team: "ATL", Position: "RB"}),
		wishlistEntry(3, Player{ID: "rb2", Name: "RB Two", Team: "DAL", Position: "RB"}),
	}}

	report := BuildWishlistReport(Config{}, wishlist, []Pick{{PlayerID: "rb1", PickNo: 1}})

	got := wishlistItemIDs(report.TopAvailableByPosition)
	want := []string{"qb1", "rb2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("top by position = %#v, want %#v", got, want)
	}
}

func TestBuildWishlistReportExcludesTeamsFromTopAvailable(t *testing.T) {
	wishlist := PlayerList{Entries: []RankedPlayer{
		wishlistEntry(1, Player{ID: "p1", Name: "Excluded Team", Team: "FA", Position: "WR"}),
		wishlistEntry(2, Player{ID: "p2", Name: "Available Team", Team: "DET", Position: "WR"}),
	}}

	report := BuildWishlistReport(Config{ExcludedTeams: []string{"FA"}}, wishlist, nil)

	if report.Items[0].Status != WishlistExcluded {
		t.Fatalf("first status = %q, want excluded", report.Items[0].Status)
	}
	if report.TopAvailable == nil || report.TopAvailable.Player.ID != "p2" {
		t.Fatalf("TopAvailable = %#v, want p2", report.TopAvailable)
	}
}

func TestFormatWishlistReport(t *testing.T) {
	report := WishlistReport{
		Items: []WishlistItem{
			{Rank: 1, Player: Player{Name: "Taken RB", Team: "ATL"}, Position: "RB", Status: WishlistTaken, TakenBy: "user1", PickNo: 3},
			{Rank: 2, Player: Player{Name: "Available WR", Team: "DET"}, Position: "WR", Status: WishlistAvailable},
		},
	}
	report.TopAvailable = &report.Items[1]
	report.TopAvailableByPosition = []WishlistItem{report.Items[1]}

	got := FormatWishlistReport(report)
	want := "Wishlist: #1 RB Taken RB (ATL) taken; #2 WR Available WR (DET) available; top=WR Available WR (DET); top_by_position=WR Available WR (DET)"
	if got != want {
		t.Fatalf("formatted = %q, want %q", got, want)
	}
}

func TestFormatWishlistItemTakenOmitsPickDetails(t *testing.T) {
	got := formatWishlistItem(WishlistItem{
		Rank:     1,
		Player:   Player{Name: "Taken RB", Team: "ATL"},
		Position: "RB",
		Status:   WishlistTaken,
		TakenBy:  "user1",
		PickNo:   3,
	})

	want := "#1 RB Taken RB (ATL) taken"
	if got != want {
		t.Fatalf("formatted = %q, want %q", got, want)
	}
}

func TestFormatWishlistReportEmpty(t *testing.T) {
	if got := FormatWishlistReport(WishlistReport{}); got != "Wishlist: none" {
		t.Fatalf("formatted = %q, want Wishlist: none", got)
	}
}

func wishlistEntry(rank int, player Player) RankedPlayer {
	return RankedPlayer{
		Rank:     rank,
		PlayerID: player.ID,
		Player:   player,
		Name:     player.Name,
		Position: player.Position,
		Team:     player.Team,
	}
}

func wishlistItemIDs(items []WishlistItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.Player.ID)
	}
	return ids
}

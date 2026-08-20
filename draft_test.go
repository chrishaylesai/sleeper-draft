package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDraftServiceResolveWithDraftID(t *testing.T) {
	service := testDraftService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/draft/draft123" {
			t.Fatalf("path = %q, want /draft/draft123", r.URL.Path)
		}
		return jsonResponse(http.StatusOK, `{"draft_id":"draft123","status":"drafting","settings":{"teams":12,"rounds":15}}`), nil
	})

	draft, err := service.Resolve(context.Background(), Config{DraftID: "draft123", Sport: "nfl"})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if draft.ID != "draft123" || draft.Settings.Teams != 12 {
		t.Fatalf("draft = %#v, want draft123 with 12 teams", draft)
	}
}

func TestDraftServiceResolveWithUsernameAndSeason(t *testing.T) {
	var paths []string
	service := testDraftService(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/user/drafter":
			return jsonResponse(http.StatusOK, `{"user_id":"user123"}`), nil
		case "/user/user123/drafts/nfl/2026":
			return jsonResponse(http.StatusOK, `[{"draft_id":"draft456"}]`), nil
		case "/draft/draft456":
			return jsonResponse(http.StatusOK, `{"draft_id":"draft456","status":"drafting","settings":{"teams":10,"rounds":16}}`), nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
			return nil, nil
		}
	})

	draft, err := service.Resolve(context.Background(), Config{Username: "drafter", Season: "2026", Sport: "nfl"})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if draft.ID != "draft456" {
		t.Fatalf("ID = %q, want draft456", draft.ID)
	}

	wantPaths := []string{"/user/drafter", "/user/user123/drafts/nfl/2026", "/draft/draft456"}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("paths = %#v, want %#v", paths, wantPaths)
	}
}

func TestDraftServiceResolveRejectsMultipleUserDrafts(t *testing.T) {
	service := testDraftService(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/user/drafter":
			return jsonResponse(http.StatusOK, `{"user_id":"user123"}`), nil
		case "/user/user123/drafts/nfl/2026":
			return jsonResponse(http.StatusOK, `[{"draft_id":"b"},{"draft_id":"a"}]`), nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
			return nil, nil
		}
	})

	_, err := service.Resolve(context.Background(), Config{Username: "drafter", Season: "2026", Sport: "nfl"})
	if err == nil {
		t.Fatal("Resolve returned nil error")
	}
	if !strings.Contains(err.Error(), "multiple nfl drafts") || !strings.Contains(err.Error(), "a, b") {
		t.Fatalf("error = %q, want multiple draft IDs", err.Error())
	}
}

func TestDraftServiceSnapshotReadsDraftAndPicks(t *testing.T) {
	updatedAt := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	service := testDraftService(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/draft/draft123":
			return jsonResponse(http.StatusOK, `{"draft_id":"draft123","status":"drafting","settings":{"teams":3,"rounds":2}}`), nil
		case "/draft/draft123/picks":
			return jsonResponse(http.StatusOK, `[
				{"player_id":"p2","picked_by":"u2","roster_id":2,"round":1,"draft_slot":2,"pick_no":2},
				{"player_id":"p1","picked_by":"u1","roster_id":1,"round":1,"draft_slot":1,"pick_no":1}
			]`), nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
			return nil, nil
		}
	})
	service.Now = func() time.Time { return updatedAt }

	snapshot, err := service.Snapshot(context.Background(), "draft123")
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}

	if snapshot.TotalPicks != 2 || snapshot.NextPickNo != 3 {
		t.Fatalf("snapshot picks = total %d next %d, want 2 and 3", snapshot.TotalPicks, snapshot.NextPickNo)
	}
	if snapshot.CurrentRound != 1 || snapshot.CurrentPick != 3 {
		t.Fatalf("current = round %d pick %d, want 1/3", snapshot.CurrentRound, snapshot.CurrentPick)
	}
	if snapshot.Picks[0].PickNo != 1 || snapshot.Picks[1].PickNo != 2 {
		t.Fatalf("picks not sorted by pick_no: %#v", snapshot.Picks)
	}
	if !snapshot.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("UpdatedAt = %v, want %v", snapshot.UpdatedAt, updatedAt)
	}
}

func TestBuildDraftSnapshotInfersTeamsAndCompletion(t *testing.T) {
	snapshot := BuildDraftSnapshot(
		Draft{
			ID:             "draft123",
			Status:         "complete",
			SlotToRosterID: map[string]int{"1": 10, "2": 20},
			Settings:       DraftSettings{Rounds: 1},
		},
		[]Pick{
			{PickNo: 2, DraftSlot: 2},
			{PickNo: 1, DraftSlot: 1},
		},
		time.Time{},
	)

	if !snapshot.Complete {
		t.Fatal("Complete = false, want true")
	}
	if snapshot.CurrentRound != 1 || snapshot.CurrentPick != 2 || snapshot.NextPickNo != 2 {
		t.Fatalf("current/next = %d/%d next %d, want 1/2 next 2", snapshot.CurrentRound, snapshot.CurrentPick, snapshot.NextPickNo)
	}
}

func TestDraftServicePollReportsTransientErrorsAndContinues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	requests := 0
	service := testDraftService(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		requests++

		if requests == 1 {
			return jsonResponse(http.StatusTooManyRequests, "slow down"), nil
		}
		switch r.URL.Path {
		case "/draft/draft123":
			return jsonResponse(http.StatusOK, `{"draft_id":"draft123","status":"drafting","settings":{"teams":2,"rounds":1}}`), nil
		case "/draft/draft123/picks":
			return jsonResponse(http.StatusOK, `[{"player_id":"p1","pick_no":1,"draft_slot":1,"round":1}]`), nil
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
			return nil, nil
		}
	})

	var sawError bool
	var sawSnapshot bool
	err := service.Poll(ctx, "draft123", time.Millisecond, func(snapshot DraftSnapshot, err error) {
		if err != nil {
			sawError = true
			return
		}
		if snapshot.TotalPicks == 1 {
			sawSnapshot = true
			cancel()
		}
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Poll error = %v, want context.Canceled", err)
	}
	if !sawError || !sawSnapshot {
		t.Fatalf("sawError=%v sawSnapshot=%v, want both true", sawError, sawSnapshot)
	}
}

func testDraftService(roundTrip func(*http.Request) (*http.Response, error)) DraftService {
	return DraftService{
		Client:  &http.Client{Transport: roundTripFunc(roundTrip)},
		BaseURL: "https://sleeper.test",
		Now:     time.Now,
	}
}

func TestDraftPickMetadataDecodesStringValues(t *testing.T) {
	var pick Pick
	if err := json.Unmarshal([]byte(`{"metadata":{"first_name":"Bijan","last_name":"Robinson"}}`), &pick); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if pick.Metadata["first_name"] != "Bijan" {
		t.Fatalf("metadata = %#v, want first_name", pick.Metadata)
	}
}

func TestResolvePersonalDraftUsesDraftOrderAndRosterMap(t *testing.T) {
	personal, err := ResolvePersonalDraft(
		Draft{
			ID:             "draft123",
			DraftOrder:     map[string]int{"user1": 3},
			SlotToRosterID: map[string]int{"3": 30},
		},
		SleeperUser{UserID: "user1"},
	)
	if err != nil {
		t.Fatalf("ResolvePersonalDraft returned error: %v", err)
	}
	if personal.UserID != "user1" || personal.DraftSlot != 3 || personal.RosterID != 30 || !personal.Known {
		t.Fatalf("personal = %#v, want user1 slot 3 roster 30", personal)
	}
	if !personal.MatchesPick(Pick{PickedBy: "user1"}) {
		t.Fatal("MatchesPick returned false for picked_by user")
	}
	if !personal.MatchesPick(Pick{RosterID: 30}) {
		t.Fatal("MatchesPick returned false for roster_id")
	}
}

func TestResolvePersonalDraftRejectsUserOutsideDraft(t *testing.T) {
	_, err := ResolvePersonalDraft(Draft{ID: "draft123", DraftOrder: map[string]int{"other": 1}}, SleeperUser{UserID: "user1"})
	if err == nil {
		t.Fatal("ResolvePersonalDraft returned nil error")
	}
	if !strings.Contains(err.Error(), "not in draft_order") {
		t.Fatalf("error = %q, want draft_order message", err.Error())
	}
}

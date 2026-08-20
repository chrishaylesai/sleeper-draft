package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type DraftService struct {
	Client  *http.Client
	BaseURL string
	Now     func() time.Time
}

type SleeperUser struct {
	UserID string `json:"user_id"`
}

type Draft struct {
	ID             string         `json:"draft_id"`
	Status         string         `json:"status"`
	DraftOrder     map[string]int `json:"draft_order"`
	SlotToRosterID map[string]int `json:"slot_to_roster_id"`
	Settings       DraftSettings  `json:"settings"`
}

type DraftSettings struct {
	Teams     int `json:"teams"`
	Rounds    int `json:"rounds"`
	PickTimer int `json:"pick_timer"`
}

type Pick struct {
	PlayerID  string            `json:"player_id"`
	PickedBy  string            `json:"picked_by"`
	RosterID  int               `json:"roster_id"`
	Round     int               `json:"round"`
	DraftSlot int               `json:"draft_slot"`
	PickNo    int               `json:"pick_no"`
	Metadata  map[string]string `json:"metadata"`
}

type DraftSnapshot struct {
	Draft        Draft
	Picks        []Pick
	TotalPicks   int
	NextPickNo   int
	CurrentRound int
	CurrentPick  int
	Complete     bool
	UpdatedAt    time.Time
}

type PersonalDraft struct {
	UserID    string
	DraftSlot int
	RosterID  int
	Known     bool
}

type SnapshotHandler func(DraftSnapshot, error)

func NewDraftService(client *http.Client) DraftService {
	return DraftService{
		Client:  client,
		BaseURL: sleeperAPIBaseURL,
		Now:     time.Now,
	}
}

func (s DraftService) Resolve(ctx context.Context, cfg Config) (Draft, error) {
	s = s.withDefaults()

	if cfg.DraftID != "" {
		return s.GetDraft(ctx, cfg.DraftID)
	}

	user, err := s.getUser(ctx, cfg.Username)
	if err != nil {
		return Draft{}, err
	}

	drafts, err := s.getUserDrafts(ctx, user.UserID, cfg.Sport, cfg.Season)
	if err != nil {
		return Draft{}, err
	}

	switch len(drafts) {
	case 0:
		return Draft{}, fmt.Errorf("no %s drafts found for username %q season %q", cfg.Sport, cfg.Username, cfg.Season)
	case 1:
		return s.GetDraft(ctx, drafts[0].ID)
	default:
		ids := make([]string, 0, len(drafts))
		for _, draft := range drafts {
			ids = append(ids, draft.ID)
		}
		sort.Strings(ids)
		return Draft{}, fmt.Errorf("multiple %s drafts found for username %q season %q: %s; set draft_id explicitly", cfg.Sport, cfg.Username, cfg.Season, strings.Join(ids, ", "))
	}
}

func (s DraftService) Snapshot(ctx context.Context, draftID string) (DraftSnapshot, error) {
	s = s.withDefaults()

	draft, err := s.GetDraft(ctx, draftID)
	if err != nil {
		return DraftSnapshot{}, err
	}

	picks, err := s.GetPicks(ctx, draftID)
	if err != nil {
		return DraftSnapshot{}, err
	}

	return BuildDraftSnapshot(draft, picks, s.Now()), nil
}

func (s DraftService) Poll(ctx context.Context, draftID string, interval time.Duration, handler SnapshotHandler) error {
	if interval <= 0 {
		return errors.New("poll interval must be greater than 0")
	}
	if handler == nil {
		return errors.New("snapshot handler is required")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		snapshot, err := s.Snapshot(ctx, draftID)
		handler(snapshot, err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s DraftService) GetDraft(ctx context.Context, draftID string) (Draft, error) {
	var draft Draft
	if err := s.getJSON(ctx, "/draft/"+url.PathEscape(draftID), &draft); err != nil {
		return Draft{}, err
	}
	if draft.ID == "" {
		draft.ID = draftID
	}
	return draft, nil
}

func (s DraftService) GetPicks(ctx context.Context, draftID string) ([]Pick, error) {
	var picks []Pick
	if err := s.getJSON(ctx, "/draft/"+url.PathEscape(draftID)+"/picks", &picks); err != nil {
		return nil, err
	}
	sort.Slice(picks, func(i, j int) bool {
		return picks[i].PickNo < picks[j].PickNo
	})
	return picks, nil
}

func (s DraftService) getUser(ctx context.Context, username string) (SleeperUser, error) {
	var user SleeperUser
	if err := s.getJSON(ctx, "/user/"+url.PathEscape(username), &user); err != nil {
		return SleeperUser{}, err
	}
	if user.UserID == "" {
		return SleeperUser{}, fmt.Errorf("user %q response did not include user_id", username)
	}
	return user, nil
}

func (s DraftService) getUserDrafts(ctx context.Context, userID, sport, season string) ([]Draft, error) {
	var drafts []Draft
	path := fmt.Sprintf("/user/%s/drafts/%s/%s", url.PathEscape(userID), url.PathEscape(sport), url.PathEscape(season))
	if err := s.getJSON(ctx, path, &drafts); err != nil {
		return nil, err
	}
	return drafts, nil
}

func (s DraftService) getJSON(ctx context.Context, path string, target any) error {
	s = s.withDefaults()

	endpoint := strings.TrimRight(s.BaseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create request %s: %w", path, err)
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("get %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get %s: unexpected status %s", path, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func (s DraftService) withDefaults() DraftService {
	if s.Client == nil {
		s.Client = http.DefaultClient
	}
	if s.BaseURL == "" {
		s.BaseURL = sleeperAPIBaseURL
	}
	if s.Now == nil {
		s.Now = time.Now
	}
	return s
}

func BuildDraftSnapshot(draft Draft, picks []Pick, updatedAt time.Time) DraftSnapshot {
	sort.Slice(picks, func(i, j int) bool {
		return picks[i].PickNo < picks[j].PickNo
	})

	teams := draft.Settings.Teams
	if teams <= 0 {
		teams = inferTeamCount(draft, picks)
	}

	totalPicks := len(picks)
	nextPickNo := totalPicks + 1
	maxPicks := 0
	if teams > 0 && draft.Settings.Rounds > 0 {
		maxPicks = teams * draft.Settings.Rounds
	}

	complete := strings.EqualFold(draft.Status, "complete") || (maxPicks > 0 && totalPicks >= maxPicks)
	if complete {
		nextPickNo = totalPicks
	}

	currentRound := 0
	currentPick := 0
	if teams > 0 {
		pickForPosition := nextPickNo
		if complete {
			pickForPosition = totalPicks
		}
		currentRound = ((pickForPosition - 1) / teams) + 1
		currentPick = ((pickForPosition - 1) % teams) + 1
	}

	return DraftSnapshot{
		Draft:        draft,
		Picks:        picks,
		TotalPicks:   totalPicks,
		NextPickNo:   nextPickNo,
		CurrentRound: currentRound,
		CurrentPick:  currentPick,
		Complete:     complete,
		UpdatedAt:    updatedAt,
	}
}

func (s DraftSnapshot) Summary() string {
	state := "active"
	if s.Complete {
		state = "complete"
	}

	return fmt.Sprintf(
		"Draft loaded: draft_id=%s status=%s round=%d pick=%d next_pick=%d total_picks=%d",
		s.Draft.ID,
		state,
		s.CurrentRound,
		s.CurrentPick,
		s.NextPickNo,
		s.TotalPicks,
	)
}

func ResolvePersonalDraft(draft Draft, user SleeperUser) (PersonalDraft, error) {
	if user.UserID == "" {
		return PersonalDraft{}, errors.New("user_id is required to resolve personal draft slot")
	}

	slot, ok := draft.DraftOrder[user.UserID]
	if !ok {
		return PersonalDraft{}, fmt.Errorf("user_id %q is not in draft_order for draft %q", user.UserID, draft.ID)
	}

	personal := PersonalDraft{
		UserID:    user.UserID,
		DraftSlot: slot,
		Known:     true,
	}
	if draft.SlotToRosterID != nil {
		personal.RosterID = draft.SlotToRosterID[strconv.Itoa(slot)]
	}
	return personal, nil
}

func (p PersonalDraft) MatchesPick(pick Pick) bool {
	if !p.Known {
		return false
	}
	if p.UserID != "" && pick.PickedBy == p.UserID {
		return true
	}
	return p.RosterID > 0 && pick.RosterID == p.RosterID
}

func inferTeamCount(draft Draft, picks []Pick) int {
	count := len(draft.SlotToRosterID)
	if len(draft.DraftOrder) > count {
		count = len(draft.DraftOrder)
	}
	for _, pick := range picks {
		if pick.DraftSlot > count {
			count = pick.DraftSlot
		}
	}
	return count
}

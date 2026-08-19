# Tasks

This is the backlog for **sleeper-draft**. See [`../AGENTS.md`](../AGENTS.md) for
the project overview.

## How to use this file

- Work is broken down by **feature** (a user-facing capability), **not** by
  application layer.
- Every task has four fields: **Description**, **Status**, **Assigned to**,
  **Notes**.
- **Status** is exactly one of: `TODO`, `IN PROGRESS`, `DONE`.
- **Before starting** a task: set **Status** → `IN PROGRESS` and fill in
  **Assigned to**.
- **On completion**: set **Status** → `DONE` and record the outcome in **Notes**.

Tasks are ordered so earlier ones unblock later ones, but each is intended to be
a shippable feature on its own.

---

### T1. Configuration & startup

- **Description:** Load a JSON `config.json` and start the app from it. Config
  covers: `draft_id`, `username`, `season`, `sport`, per-position draft targets,
  excluded players, excluded teams, refresh interval, and file paths
  (`players.json`, `rankings.csv`, `wishlist.csv`). Ship a documented sample
  `config.json`. Delivers the "configurable" requirement.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Implemented initial Go CLI, JSON config loading, validation,
  `-config` path override, sample `config.json`, and focused config tests.
  Verified with `go test ./...`, `go build ./...`, and
  `go run ./... -config config.json`.

### T2. Player database sync

- **Description:** Fetch Sleeper `GET /players/nfl` and cache it to
  `players.json` (`player_id, name, team, position`, plus `search_rank`). Refresh
  only when the cache is stale (at most once per day) and stay within rate
  limits. Build a name(+position/team) lookup index used for CSV matching.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Implemented Sleeper `/players/<sport>` sync, compact
  `players.json` cache with `search_rank`, 24-hour freshness checks, and a
  name+position(+team) lookup index for CSV matching. Verified with
  `go test ./...`, `go build ./...`, and a cached `go run ./... -config ...`
  smoke test.

### T3. Draft resolution & live pick polling

- **Description:** Resolve the target draft either from an explicit `draft_id` or
  by looking it up from `username` + `season` (`/user/<username>` →
  `/user/<user_id>/drafts/nfl/<season>`). Read draft settings and `/picks`, and
  poll on the configured refresh interval. Derive current round/pick and total
  picks made (using `slot_to_roster_id` / `draft_order` where needed).
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Implemented draft resolution from explicit `draft_id` or
  username+season, draft settings and picks reads, current round/pick snapshot
  derivation, and `-poll` refresh-loop support that reports transient API
  errors and continues. Verified with `go test ./...` and `go build ./...`.

### T4. Rankings & wishlist import

- **Description:** Parse the two CSVs — `rankings.csv` and `wishlist.csv` — each
  with header `player_name, player_position, player_team` (team optional).
  Validate every row against `players.json`; report any unmatched row as an
  error. Resolve each row to a `player_id` and preserve file order as rank.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Implemented shared CSV import for `rankings.csv` and
  `wishlist.csv`, exact header validation, player resolution through
  `players.json`, row-order ranks, aggregated unmatched-row errors, and
  header-only starter files. Verified with `go test ./...`, `go build ./...`,
  and a live startup smoke test against the configured mock draft.

### T5. Position summary

- **Description:** For each position, compute how many have been drafted so far
  (from the picks in T3) and how many remain relative to the per-position targets
  from config (T1). Present drafted vs. remaining per position.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Implemented per-position drafted counts from current picks,
  remaining counts against configured targets, deterministic summary formatting,
  and startup/polling output. Verified with `go test ./...`, `go build ./...`,
  and a live startup smoke test against the configured mock draft.

### T6. Best available per position

- **Description:** For each position, determine the highest-rated available
  player. Order primarily by `rankings.csv` (T4); for players not in the rankings,
  fall back to Sleeper `search_rank`. Exclude already-drafted players and any
  players/teams excluded in config.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Implemented best-available selection per configured target
  position, custom rankings precedence, Sleeper `search_rank` fallback,
  drafted-player filtering, and config player/team exclusions. Verified with
  `go test ./...`, `go build ./...`, and a live startup smoke test against the
  configured mock draft.

### T7. Wishlist tracking

- **Description:** For the `wishlist.csv` list, show each target's availability
  (available, or taken and by whom) and surface the top available wishlist
  player(s), overall and/or per position.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Implemented wishlist availability reporting in wishlist rank order,
  taken-by/pick annotations, exclusion handling, top available overall, and top
  available by position. Verified with `go test ./...`, `go build ./...`, and a
  live startup smoke test against the configured mock draft.

### T8. Live TUI dashboard

- **Description:** A Bubble Tea full-screen dashboard that lays out the current
  round/pick, position summary (T5), best-available per position (T6), and
  wishlist tracking (T7), refreshing on the configured interval. Handle API
  errors gracefully and exit cleanly on Ctrl-C.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Implemented the default Bubble Tea/Lip Gloss full-screen dashboard
  with current pick, position summary, best available, wishlist tracking,
  refresh ticks, transient error display, and `q`/Ctrl-C exit. Kept
  non-interactive output behind `-once`. Verified with `go test ./...`,
  `go build ./...`, and a live `go run ./... -config config.json -once` smoke
  test against the configured mock draft.

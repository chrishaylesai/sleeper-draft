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

### T9. Colorized TUI readability

- **Description:** Add purposeful color styling to the Bubble Tea dashboard to
  make the live draft state easier to scan. Use Lip Gloss styles for section
  headings, draft status, position remaining counts, best-available ranking
  sources, wishlist statuses, and refresh errors. Keep colors restrained and
  readable in typical terminal themes.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Added centralized Lip Gloss color styles for dashboard title,
  metadata, section headings, status/error text, remaining counts, ranking
  sources, wishlist statuses, and footer help. Verified with `go test ./...`,
  `go build ./...`, and `go run ./... -config config.json -once`.

### T10. Hide zero-target positions from position tally

- **Description:** Do not display positions in the Positions tally when their
  configured `position_targets` value is `0`. This should apply to both the TUI
  Positions section and the non-interactive `-once` position summary output.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Updated position summary generation to omit explicitly zero-target
  positions from both TUI and `-once` output while preserving positive targets
  and omitted-but-drafted positions. Verified with `go test ./...`.

### T11. Fix Positions table column alignment

- **Description:** Fix the Bubble Tea Positions table spacing so the `TARGET`
  column is visually separated from `REMAINING`. The target value should align
  under the `TARGET` header instead of appearing immediately next to the
  remaining count.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Updated `renderPositionTable` to pad columns by visible terminal
  width so ANSI-colored remaining counts do not collapse the `TARGET` column.
  Added rendering coverage for colored remaining values. Verified with
  `go test ./...`, `go build ./...`, and a `-once` smoke test.

### T12. Split wishlist into position columns

- **Description:** Update the Bubble Tea Wishlist section to display wishlist
  players in separate columns by position instead of one combined list. Include
  an empty column for every position whose configured `position_targets` value is
  greater than `0`, even if there are no wishlist players for that position.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Updated the TUI wishlist renderer to group items into position
  columns, include empty columns for positive `position_targets`, omit
  zero-target positions, and preserve rank order plus status styling. Verified
  with `go test ./...`, `go build ./...`, and a `-once` smoke test.

### T13. Prune irrelevant players from players cache

- **Description:** Create a script or CLI command that removes irrelevant players
  from `players.json` based on Sleeper `search_rank`, producing a smaller local
  cache for faster startup and easier inspection.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Added a `-prune` CLI command (`-prune-dry-run` previews without
  rewriting) that rewrites `players.json` in place. Keeps every player
  referenced by `rankings.csv` or `wishlist.csv`, rankless-but-draftable
  positions (`DEF`, which Sleeper leaves without a `search_rank`), and players
  ranked at or below the `prune_rank_cutoff` config setting (default `500`).
  Supplying a custom `rankings.csv` ignores the cutoff entirely — rankings are
  the app's primary ordering, so when present they define the relevant pool and
  the summary reports `cutoff=ignored`. Both lists are resolved against the
  unpruned database first, so CSV validation keeps working, and the pruned file
  preserves `cached_at` so the app still treats it as a fresh cache. On the real
  cache the cutoff path cuts 12,221 players to 1,160 (1.7 MB to 162 KB); a
  61-row rankings file cuts it to 96. Verified with `go test ./...`,
  `go build ./...`, and dry-run plus real prune runs against a copy of the live
  cache, with and without rankings. Follow-up fix: dashboard/text position
  summaries now apply the same relevance rules at render time, so total
  remaining counts are pruned even when the local `players.json` cache has not
  been rewritten. Verified with `go test ./...`, `go build ./...`, and a live
  `-once` smoke run.

### T14. Display my overall pick numbers

- **Description:** Add a dashboard section that displays this user's overall
  draft pick numbers. Picks already used should render in red, and upcoming
  picks should render in green.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Derive the user's draft slot/roster from the existing personal
  draft resolution logic. Use draft settings and total teams/rounds to calculate
  all overall pick numbers for that slot, including snake-draft order if
  applicable from Sleeper draft metadata/settings. Implemented a compact `My
  Picks` dashboard section that renders used overall picks in red and upcoming
  picks in green. Added reusable personal pick-number calculation for snake and
  linear drafts. Verified with `go test ./...`, `go build ./...`, and a live
  `-once` smoke run.

### T15. Show picks until my next pick

- **Description:** Add to the existing Draft section the number of picks until
  this user's next pick.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Reuse the same personal pick-number calculation from T14. Display
  `on the clock` or `0` when the next pick belongs to the user. Handle completed
  drafts and cases where no future user pick remains. Implemented an `Until
  yours` field in the TUI Draft section using the T14 personal pick-number
  calculation. It renders `on the clock` when the next pick is the user's, a
  green numeric countdown for later picks, and `none` when no future personal
  pick is available. Verified with `go test ./...`, `go build ./...`, and a
  live `-once` smoke run.

### T16. Highlight wishlist players drafted by me

- **Description:** In the Bubble Tea Wishlist section, keep available players
  green and taken players red, except when a taken player was selected by this
  user. Those personally drafted wishlist players should render yellow/gold.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Use the resolved personal draft identity to distinguish taken by
  this user from taken by other teams without changing the compact wishlist
  text labels. Added a `SelectedByMe` flag to wishlist items and render those
  taken items with the existing gold warning style, while keeping available
  green and taken-by-others red. Verified with `go test ./...`,
  `go build ./...`, and a live `-once` smoke run.

### T17. Keep dashboard refresh cadence on configured interval

- **Description:** Fix the Bubble Tea dashboard refresh loop so live draft
  data, including the picks-until-next-pick countdown, refreshes on the
  configured interval instead of waiting an extra interval after each API
  request finishes.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Preserve the configured refresh interval while avoiding overlapping
  Sleeper API requests. Updated the Bubble Tea model to schedule refresh ticks
  independently from snapshot completion and track when a fetch is already in
  flight. This keeps polling aligned to `refresh_interval_seconds` whenever API
  calls complete within the interval, without overlapping requests. Verified
  with `go test ./...`, `go build ./...`, and a live `-once` smoke run.

### T18. Record each team's starting RB and WR roles

- **Description:** Create a maintained player list covering every NFL team and
  identifying its current WR1, WR2, RB1, and RB2. Store the data in a simple,
  machine-readable format suitable for use by the app, match each entry to the
  Sleeper player database where possible, and document the source and as-of date
  so ambiguous depth charts, committees, injuries, and later changes can be
  reviewed.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Added `2026_team_skill_position_roles.csv` at the repository root
  with 128 rows covering WR1, WR2, RB1, and RB2 for all 32 teams. Used the Draft
  Punk 2026 depth charts updated 2026-08-19, preserved source names and dates,
  marked every role as projected, and matched all 128 entries to the 2026-08-20
  Sleeper cache. Documented the schema and preseason limitations in README.md.
  Validated exact team/role coverage, non-empty Sleeper IDs, CSV row count, and
  a spreadsheet inspection/render pass.

### T19. Create simple team strength-of-schedule ratings

- **Description:** Produce a simple strength-of-schedule rating for every NFL
  team using a transparent, documented method and consistent source data. Store
  both the underlying score and an easiest-to-hardest or hardest-to-easiest rank
  in a machine-readable file, with the season and as-of date recorded.
- **Status:** DONE
- **Assigned to:** Codex
- **Notes:** Added `2026_team_strength_of_schedule.csv` at the repository root
  with all 32 teams ranked hardest to easiest by their 17 scheduled 2026
  opponents' combined 2025 regular-season W-L-T records. The file stores each
  raw 289-game opponent aggregate, exact and published winning percentages,
  competition rank, methodology, primary and cross-check source URLs, and
  research dates. Validated all computed percentages and ranks against the CBS
  Sports and official New York Giants tables, checked for formula errors, and
  completed spreadsheet inspection and visual render passes. Documented the
  method and limitations in README.md.

### T20. Research team defenses against the run and pass

- **Description:** Research and rate every NFL team defense in two independent
  categories: run defense and pass defense. Use documented, objective source
  statistics; produce separate scores, rankings, or tiers for the two
  categories; and store the results in a machine-readable file with the season,
  methodology, sources, and as-of date recorded.
- **Status:** TODO
- **Assigned to:** Unassigned
- **Notes:** Do not collapse run and pass defense into a single grade. Flag small
  samples or conflicting indicators rather than implying false precision.


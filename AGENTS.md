# sleeper-draft

A command-line application, written in Go, that runs live during a fantasy
football draft on [Sleeper.com](https://sleeper.com) and helps you make picks.
Using Sleeper's **read-only** API, it refreshes often enough to catch picks close
to when they happen and displays:

- the **current round / pick**,
- how many of **each position** have been drafted so far and how many remain
  (relative to your configured targets),
- the **highest-rated available player** for each position,
- the **same statistics for a personal wishlist** / draft-priority list.

It is configurable for how many of each position you want to draft, which players
or teams to exclude, and how frequently to refresh.

## Tech stack

- **Language:** Go.
- **TUI:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) (with
  [Lip Gloss](https://github.com/charmbracelet/lipgloss) for layout) — a live,
  full-screen dashboard redrawn on each refresh.
- **HTTP:** standard library `net/http` against the Sleeper API.
- **Config:** JSON.

## Data & files

| File                | Purpose                                                                 |
| ------------------- | ----------------------------------------------------------------------- |
| `config.json`       | Settings: draft target, per-position targets, exclusions, refresh, paths |
| `players.json`      | Local cache of the Sleeper player DB (`player_id, name, team, position`) |
| `rankings.csv`      | Your overall custom rankings (ordered)                                   |
| `wishlist.csv`      | Your personal wishlist / draft-priority list (ordered)                   |

- **`players.json`** is fetched and cached by the app from Sleeper's
  `/players/nfl` endpoint (refreshed at most once per day — see rate limits) and
  is the source of truth for player identity.
- **`rankings.csv`** and **`wishlist.csv`** are two *separate* files sharing the
  header `player_name, player_position, player_team` (team optional, used to
  disambiguate duplicate names). On import, **every row is validated against
  `players.json`; any row that fails to match raises an error.**
- **Rankings are pluggable:** `rankings.csv` is the primary ordering for
  "highest-rated available"; players not listed fall back to Sleeper's
  `search_rank`.
- The **target draft** is identified either by an explicit `draft_id` (from the
  Sleeper draft URL) or by looking it up from your Sleeper **username + season**.

## Sleeper API reference

Read-only. Base URL: `https://api.sleeper.app/v1`

| Endpoint                                        | Returns                                                                 |
| ----------------------------------------------- | ----------------------------------------------------------------------- |
| `GET /draft/<draft_id>`                         | Draft settings (teams, rounds, pick_timer), status, `draft_order`, `slot_to_roster_id` |
| `GET /draft/<draft_id>/picks`                   | Picks: `player_id, picked_by, roster_id, round, draft_slot, pick_no, metadata` |
| `GET /players/nfl`                              | Dict keyed by `player_id`: `position, team, status, fantasy_positions, search_rank` (~5MB) |
| `GET /user/<username>`                          | User object incl. `user_id`                                             |
| `GET /user/<user_id>/drafts/nfl/<season>`       | The user's drafts for a season (username lookup)                        |

**Rate limits:** stay under **1000 calls/minute** or risk an IP block. The
`/players/nfl` response is large — call it **at most once per day**.

## Task tracking

The backlog lives in **[`./context/TASKS.md`](./context/TASKS.md)** — it is the
source of truth for what to build next. Work is organized by **feature** (a
user-facing capability), **not by application layer**.

Each task has four fields:

- **Description** — what the feature does.
- **Status** — exactly one of `TODO`, `IN PROGRESS`, `DONE`.
- **Assigned to** — who is working on it.
- **Notes** — context, decisions, and outcome.

**Workflow — follow this every time:**

1. **Before starting a task:** set its **Status** to `IN PROGRESS` and fill in
   **Assigned to**.
2. **On completion:** set its **Status** to `DONE` and update **Notes** with the
   outcome.
3. **On completion** Update README.md 

Do not begin implementation work on a task whose status is still `TODO` without
first updating it.

## Build & run

```sh
go build ./...        # build
go run ./...          # run against the draft configured in config.json
```

The app reads `config.json` from the working directory (or a path passed via
flag). Configure your `draft_id` (or username + season), per-position targets,
exclusions, and refresh interval there before launching.

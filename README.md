# sleeper-draft

`sleeper-draft` is a Go command-line application for live fantasy football draft
support using Sleeper's read-only API.

The current implementation loads configuration, validates startup settings, and
syncs Sleeper's NFL player database into a local `players.json` cache. Later
tasks will add draft polling, rankings import, and the live TUI dashboard.

## Requirements

- Go 1.24 or newer
- Network access to `https://api.sleeper.app/v1` when `players.json` is missing
  or older than 24 hours

## Configure

The app reads `config.json` from the working directory by default. A sample file
is included in this repository.

Set either:

- `draft_id`, from the Sleeper draft URL
- or both `username` and `season`

Also configure position targets, exclusions, refresh interval, and data file
paths:

```json
{
  "draft_id": "your_draft_id_from_sleeper_url",
  "username": "",
  "season": "2026",
  "sport": "nfl",
  "position_targets": {
    "QB": 2,
    "RB": 5,
    "WR": 5,
    "TE": 2,
    "K": 1,
    "DEF": 1
  },
  "excluded_players": ["example_player_id"],
  "excluded_teams": ["FA"],
  "refresh_interval_seconds": 5,
  "players_path": "players.json",
  "rankings_path": "rankings.csv",
  "wishlist_path": "wishlist.csv"
}
```

`players_path` is used for the Sleeper player cache. The app refreshes it at
most once per 24 hours. `rankings_path` and `wishlist_path` are configured now
for custom player lists.

## Rankings and Wishlist

`rankings.csv` and `wishlist.csv` use the same header:

```csv
player_name,player_position,player_team
```

`player_team` can be empty unless it is needed to disambiguate duplicate player
names. Every non-header row is validated against `players.json`; startup fails
clearly if any row cannot be matched.

## Build

```sh
go build ./...
```

To write the binary outside the repository root:

```sh
go build -o /tmp/sleeper-draft ./...
```

## Run

Run with the default `config.json`:

```sh
go run ./...
```

Run with an explicit config path:

```sh
go run ./... -config path/to/config.json
```

On startup, the app prints a config summary and loads players from either the
fresh local cache or the Sleeper API. It then resolves the configured draft and
prints the current draft snapshot:

```text
Config loaded: target=... sport=nfl refresh=5s positions=...
Players loaded: 12345 source=cache
Player lists loaded: rankings=0 wishlist=0
Draft loaded: draft_id=... status=active round=1 pick=4 next_pick=4 total_picks=3
Position summary: QB drafted=0 remaining=2 target=2; RB drafted=1 remaining=4 target=5
Best available: QB Josh Allen (BUF) sleeper_rank=4; RB Derrick Henry (BAL) sleeper_rank=7
```

If `players.json` is missing or stale, the first run will fetch
`/players/nfl` from Sleeper and write a compact local cache.

To keep polling the draft on `refresh_interval_seconds`, pass `-poll`:

```sh
go run ./... -poll
```

## Test

Run the full test suite:

```sh
go test ./...
```

In restricted environments where Go cannot write to the default build cache, set
`GOCACHE` to a writable temp directory:

```sh
GOCACHE=/private/tmp/sleeper-draft-gocache go test ./...
GOCACHE=/private/tmp/sleeper-draft-gocache go build ./...
```

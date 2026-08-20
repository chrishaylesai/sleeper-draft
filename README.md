# sleeper-draft

`sleeper-draft` is a Go terminal dashboard for live fantasy football draft
support using Sleeper's read-only API.

It loads your draft configuration, syncs Sleeper's NFL player database into a
local `players.json` cache, imports custom rankings and wishlist CSVs, and
refreshes a full-screen Bubble Tea dashboard during the draft.

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

## Prune the player cache

Sleeper's player database is around 12,000 players, but only a small slice is
fantasy relevant. `-prune` rewrites `players.json` in place, keeping only:

- players whose Sleeper `search_rank` is at or below the cutoff
  (`search_rank` is lower-is-better; unranked players carry values like
  `9999999`),
- team defenses (`DEF`), which Sleeper leaves without a `search_rank`,
- every player referenced by `rankings.csv` or `wishlist.csv`, regardless of
  rank.

Preview the result without touching the file:

```sh
go run ./... -prune -prune-dry-run
```

```text
Prune dry run for players.json: cutoff=500 before=12221 after=1160 removed=11061 kept_ranked=1113 kept_rankless=32 kept_listed=15
```

Then prune for real, optionally choosing your own cutoff:

```sh
go run ./... -prune                 # default cutoff 500
go run ./... -prune -prune-rank 300 # tighter pool
```

Pruning preserves the cache's `cached_at` value and refreshes its modification
time, so the app treats the smaller file as a fresh cache. The next time the
cache goes stale (24 hours), the app refetches the full player database from
Sleeper — rerun `-prune` after that if you want the smaller cache back.

Rankings and wishlist rows are resolved against the *unpruned* database before
anything is removed, so pruning never breaks CSV validation for the lists you
have today. If you later add a player who was pruned out, delete
`players.json` to resync from Sleeper.

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

The dashboard refreshes on `refresh_interval_seconds` and exits cleanly with
`q` or `Ctrl-C`.

For a non-interactive smoke test, print one snapshot and exit:

```sh
go run ./... -once
```

In `-once` mode, the app prints a config summary, loads players from either the
fresh local cache or the Sleeper API, resolves the configured draft, and prints
the current draft snapshot:

```text
Config loaded: target=... sport=nfl refresh=5s positions=...
Players loaded: 12345 source=cache
Player lists loaded: rankings=0 wishlist=0
Draft loaded: draft_id=... status=active round=1 pick=4 next_pick=4 total_picks=3
Position summary: QB personal_drafted=0 personal_remaining=2 target=2 total_drafted=0 total_remaining=474; RB personal_drafted=1 personal_remaining=4 target=5 total_drafted=4 total_remaining=923
Best available: QB Josh Allen (BUF) sleeper_rank=4; RB Derrick Henry (BAL) sleeper_rank=7
Wishlist: none
```

If `players.json` is missing or stale, the first run will fetch
`/players/nfl` from Sleeper and write a compact local cache.

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

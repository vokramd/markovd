# Task: Read a run's callback events back (`GET /runs/{id}/events`)

## Status

Done (2026-11-08). For markov-ide plan 003 phase 3 (follow a markovd run inside the IDE).

## Summary

markovd kept every callback event in `events` but had no way to read them back in order. The IDE
replays them through the same code it uses for a local run, so it needs them raw.

- `GET /api/v1/runs/{id}/events?after=N&limit=M`: the run's events and its sub-runs'
  (`<id>-<suffix>`, not another run that only shares the prefix), oldest first, after event id N.
  Answers `{events: [{id, event}], next, status}`; `next` stays put when nothing is new, and
  `status` tells a poller when to stop. Limit defaults to 500, at most 2000. 404 for an unknown
  run, 400 for a bad cursor. `db.ListRunEvents`.
- `markovd-cli runs events <id> [--after N] [--limit M]`: one line per event, then
  `run <status>; next --after N`; `--output json` prints the page.
- README API table and `docs/reference/markovd-cli.md`.

## Verification

- `TestRunEventsPassesTheCursorAndPrintsThePage` (CLI, fake server).
- `TestRunEventsReplayARunWithItsSubRuns` (API against Postgres: paging, sub-runs included,
  prefix-sharing run excluded, empty poll keeps the cursor, 404, 400).
- Live on breadboard's k3s after `make host-rebuild-markovd`: `runs events` on a completed run, and
  the IDE following quickstart (90 events) to completion.

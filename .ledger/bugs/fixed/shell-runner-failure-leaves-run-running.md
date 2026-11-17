---
title: Run stays "running" when markov exits before emitting events
severity: medium
component: runner
---

## Problem

With the shell runner, if the `markov` process exits with an error before it sends any callback event, markovd keeps the run in `running` indefinitely. The error is only visible in the server log.

## Reproduction

1. Start markovd with `MARKOVD_RUNNER=shell` and leave `TMPDIR` at `/tmp` on a machine where `/tmp` contains a socket (for example `/tmp/.ICE-unix/*`).
2. Create and trigger any workflow.
3. markov fails at startup: `fingerprinting workflow source: ... unsupported workflow source entry "/tmp/.ICE-unix/2944"`.
4. `GET /api/v1/runs/{id}` stays `running` (observed for 2 minutes) with no steps.

## Expected

The run is marked `failed` with the process error when the runner process exits non-zero and no terminal event was received.

## Actual

Status never leaves `running`.

## Notes

Found while verifying [claude step progress](../../tasks/done/claude-step-progress-ui.md). The trigger here is the shell runner materializing the workflow into the shared temp directory; the missing failure propagation is the bug. A separate improvement would be to materialize workflows into a private directory.

## Additional production evidence — 2026-11-17

Operations VM markovd-ops, v0.2.0 engine: simultaneous scheduled consumer and
hourly discovery share the engine's default SQLite checkpoint file. Consumer
markov-run-32e4af7bbc18e63b7506f17c remained running with no recorded steps.
The preceding consumer recorded SQLITE_BUSY persisting a completed step,
although its separately launched discovery completed. Verify process exit
and propagate a terminal status even when no callback arrives; do not blindly
retry a dispatch whose external effect already happened. Separate core bug:
markov/.ledger/bugs/open/sqlite-concurrent-processes-busy.md.

## Source resolution — 2026-11-17

Fixed on `fix/shell-runner-exit-status`: persist API runs before launch, observe
shell launch failures and reaped exits for API/scheduled runs, atomically fail
only unfinished rows with a safe diagnostic event. Existing terminal statuses
are preserved; zero exit without a terminal callback cannot prove completion.
Reaped processes leave the registry and oversized logs are drained. No replay.

The early-exit regression fails against original source and passes with the fix.
The full Go suite passed under the race detector with an isolated PostgreSQL 16
fixture, including actual and synchronous early exits, missing binary, scheduled
exit, safe events, idempotency and final-state preservation. Vet and builds pass.
See [task](../../tasks/done/shell-exit-status.md) and
[shell lifecycle reference](../../../docs/reference/shell-runner.md).
Source fix only; not released/deployed. Production remains on PostgreSQL.

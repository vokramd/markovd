# Report shell process exits

Fix missing terminal status without replaying work. Persist API run before
launch, connect API and scheduled shell exits, atomically fail only unfinished
runs and record a safe diagnostic. Cover early/zero exits, launch failures,
terminal states, scheduled launches and oversized logs. Source-only work in
`fix/shell-runner-exit-status`; no production deployment or release.

## Resolution and verification — 2026-11-17

- API records its run and ID before launch; shell launch errors and exits
  report via a callback attached to API and scheduled requests.
- One atomic SQL statement fails only pending/running rows and inserts a safe
  `run_failed` diagnostic. Completed, failed, paused and cancelled states survive.
  Zero exit without a terminal callback is also failure, never assumed success.
- Reaped processes leave the registry; oversized log lines are drained to avoid
  blocking process exit. No external effects are retried.
- Original-source negative control fails the early-exit test; fixed source passes.
- `go test -race -p 1 ./...` passed with MARKOVD_TEST_DATABASE_URL pointing to an
  isolated loopback PostgreSQL 16 container. Covers synchronous exit before Start
  returns, real exit 17, failed launch, safe diagnostics, scheduled exit, zero
  exit, oversized stdout and idempotent preservation of all terminal states.
- `go vet ./...`, `make build` and `git diff --check` passed.

[Shell lifecycle reference](../../../docs/reference/shell-runner.md).
Source fixed locally, not released or deployed. Restart/DB-outage reconciliation
and Kubernetes job reconciliation are outside this fix.


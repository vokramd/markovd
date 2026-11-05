# Make runs help available locally

Requested 2026-10-17: make Breadboard's `./bin/markovd-cli runs --help` work.
Keep the existing run timestamp changes on `feat/cli-run-start-times`.

- [x] Handle run-group help before loading credentials or creating the API client.
- [x] Describe all run commands and creation/logging options.
- [x] Verify offline help, existing CLI behavior, and the rebuilt local wrapper.
- [x] Record evidence and close the task and bug.

## Verification

`go test ./cmd/markovd-cli` passes, including regression coverage for
`runs --help`, `runs -h`, `runs help`, `help runs`, and bare `runs`. Each
case includes missing config/CA files, an unreachable server, and a stdin
reader that fails the test if accessed. Existing configuration precedence and
run-operation tests pass after separating global flag parsing from config
loading.

Rebuilt the ignored binary with `make markovd-cli`; repeated with module-cache
access after the sandboxed build emitted a cache-write warning. Breadboard's
exact `./bin/markovd-cli runs --help` command exits 0 and displays commands,
creation flags, log following, examples, and global flag placement. The wrapper
also exits 0 with an unreachable server, missing explicit config and CA paths,
and `--password-stdin runs -h`. `git diff --check` passes in both repositories.
No service deployment or model calls are needed. Changes remain uncommitted on
`feat/cli-run-start-times`, alongside the earlier timestamp work.

# Show when runs started in CLI tables

Requested 2026-10-16: `runs list` lacks a started date/time, making benchmark
runs difficult to distinguish. Work on `feat/cli-run-start-times`.

The API already returns `started_at`; no server or storage change is needed.
Add a labeled, aligned `STARTED` column using the CLI host's local time with an
explicit UTC offset. Missing start timestamps show `-`; unparseable timestamps
remain visible as supplied. Reuse the layout for single-run table output and
keep JSON/YAML fields unchanged. Document it and rebuild the local CLI so
Breadboard's `./bin/markovd-cli` wrapper picks up the fix.

- [x] Add the table column and document time-zone/missing-value behavior.
- [x] Run the existing CLI package tests and build the CLI.
- [x] Verify live `runs list`, a run detail, and unchanged structured output.
- [x] Record checks and move this task to done.

## Verification

`go test ./cmd/markovd-cli` passes. `make markovd-cli` rebuilt the ignored
checkout binary; the first sandboxed build warned about the module stat cache,
so the build was repeated successfully with cache access. No server rebuild or
deployment is required.

Through Breadboard's `./bin/markovd-cli`, live list and detail output show
`markov-run-64a6b4f4` started at `2026-10-16 18:54:29 -04:00`.
`--output json runs get` retains `2026-10-16T22:54:29.951977Z` and the separate
creation timestamp. The API already supplies these fields; only table
rendering changed. The existing CLI suite and real service reads were used;
no model inference or new runs were submitted. `gofmt -d` is empty and
`git diff --check` passes. Changes are on `feat/cli-run-start-times`, uncommitted.

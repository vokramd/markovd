# `runs logs --follow` stops early on long log lines

## Summary

`markovd-cli runs logs <run-id> --follow` could exit while the run was still
running. Both the server's log-stream handlers and the CLI's SSE reader used
`bufio.Scanner` with its default 64 KiB line limit. Agent output lines can be
larger, for example a 130 KB tool result.

On the server, an oversized line ended the scan, and the handler still sent
`event: done`. The client could not tell a scanner error from a finished
stream. On the client, the same limit failed the read. The CLI also returned as
soon as the stream closed, so a stream that ended before the run finished
ended the command too.

## Fix

- `internal/api/runs.go`: both stream handlers accept lines up to 16 MiB. A
  scanner error is now sent as `event: error` instead of `event: done`.
- `cmd/markovd-cli`: the SSE reader accepts the same line size. It parses
  `event:` and `data:` fields by event, so multi-line data is joined, comments
  are skipped, and `done` is not printed. An `error` event fails the command.
  After the stream ends, `--follow` polls the run until it reaches a terminal
  status, the same way `--wait` does.

## Verification

`cmd/markovd-cli/runs_logs_test.go` streams a 130,196-byte line from a test
server, ends the stream while the run is still `running`, and checks that the
CLI prints the line and waits for the run to complete. `go test ./...` passes
(2026-11-04). The server half takes effect only after markovd is rebuilt and
deployed.

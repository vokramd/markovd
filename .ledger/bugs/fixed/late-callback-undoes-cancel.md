# A cancelled run shows as failed

Cancelling a running run set its status to `cancelled`, but the runner it killed still sent
`run_failed` on the way out, and the callback handler (`UpsertRunFromEvent`) overwrote the status.
Found 2026-11-08 cancelling a run from markov-ide: markovd listed it as failed.

Fixed: `UpsertRunFromEvent` keeps `cancelled` once set (a cancelled run cannot be resumed, so
nothing should move it back). `TestALateCallbackDoesNotUndoACancel` (Postgres). Live: a run
cancelled from the IDE stays `cancelled` after the pod's `run_failed` arrives.

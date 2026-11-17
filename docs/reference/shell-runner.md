# Shell runner lifecycle

For API launches, markovd records the run before starting the engine. Scheduled
runs already have a durable run ID before launch. A shell launch failure or
process exit without a terminal callback marks a pending/running run failed and
adds a `run_failed` event with `source: shell_runner` and an exit code. Code `-1`
means no normal exit code was available (for example a failed launch or signal).
Exit zero without a terminal callback also fails: it cannot prove completion.
Arguments, output and arbitrary launch errors are excluded from this event.

Completed, failed, paused and cancelled runs retain their status. This fallback
never restarts or replays a workflow: an external effect may already have
happened before a checkpoint failed. Oversized log lines are drained so the
engine can exit, and reaped processes are removed from the runner's registry.

This is in-process shell supervision. It does not reconcile processes after a
markovd restart or a database outage. Recording an exit has a five-second
context deadline; failures are logged for operator follow-up. Kubernetes job
reconciliation is unchanged. Production checkpoint storage can continue to use
PostgreSQL independently of this lifecycle fix.

# Runs help tries to authenticate

`./bin/markovd-cli runs --help` attempts to authenticate before recognizing
help, so it fails when the service is unavailable and never explains the run
commands. The resource command is dispatched only after configuration loading,
TLS setup, and authentication.

Help should print local usage and exit successfully without reading credentials
or contacting the server. Track the fix in
[the task](../../tasks/done/cli-runs-help.md).

Fixed by parsing global flags and handling local help before configuration
loading and API client creation. CLI tests pass with unavailable configuration,
credentials, and server. Rebuilt the local binary and confirmed the exact
Breadboard wrapper command exits 0. See the task for verification details.

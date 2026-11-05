package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunLogsFollowWaitsForRunAfterStreamEnds(t *testing.T) {
	var statusChecks atomic.Int32
	largeLogLine := strings.Repeat("x", 130196)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/runs/markov-run-test/logs/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, "data: final log line\n\n")
			_, _ = fmt.Fprintf(w, "data: %s\n\nevent: done\ndata: stream ended\n\n", largeLogLine)
		case "/api/v1/runs/markov-run-test":
			status := "running"
			if statusChecks.Add(1) > 1 {
				status = "completed"
			}
			writeTestJSON(w, run{RunID: "markov-run-test", Status: status})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &apiClient{baseURL: server.URL, client: server.Client()}
	var stdout bytes.Buffer
	err := runLogs(context.Background(), client, cliConfig{
		Output:       "table",
		PollInterval: time.Millisecond,
	}, []string{"markov-run-test", "--follow"}, &stdout)
	if err != nil {
		t.Fatalf("runLogs returned error: %v", err)
	}
	if got := statusChecks.Load(); got < 2 {
		t.Fatalf("run status was checked %d time(s), want at least 2", got)
	}
	if !strings.Contains(stdout.String(), "final log line") {
		t.Fatal("streamed log line missing from output")
	}
	if !strings.Contains(stdout.String(), largeLogLine) {
		t.Fatalf("large streamed log line was truncated (output bytes=%d)", stdout.Len())
	}
	if strings.Contains(stdout.String(), "\nstream ended\n") {
		t.Fatal("SSE completion event was printed as a log line")
	}
	if !strings.Contains(stdout.String(), "waiting for run markov-run-test to finish") || !strings.Contains(stdout.String(), "run markov-run-test completed") {
		t.Fatalf("run status progress missing from output: %q", stdout.String())
	}
}

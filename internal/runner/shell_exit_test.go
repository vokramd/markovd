package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShellRunnerReapsAndReportsExit(t *testing.T) {
	for _, tc := range []struct {
		name, script string
		failed       bool
	}{
		{"early_failure", "exit 17\n", true},
		{"clean_exit", "exit 0\n", false},
		{"oversized_output", "head -c 131072 /dev/zero | tr '\\000' x\nexit 19\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "markov")
			if err := os.WriteFile(path, []byte("#!/bin/sh\n"+tc.script), 0700); err != nil {
				t.Fatal(err)
			}
			r := NewShellRunner(path)
			exited := make(chan error, 2)
			id, err := r.Start(context.Background(), RunRequest{RunID: "markov-run-exit", WorkflowYAML: "entrypoint: main\n", OnExit: func(id string, err error) {
				if id != "markov-run-exit" {
					t.Errorf("wrong run ID %q", id)
				}
				exited <- err
			}})
			if err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-exited:
				if (err != nil) != tc.failed {
					t.Fatalf("exit error=%v", err)
				}
			case <-time.After(5 * time.Second):
				_ = r.Cancel(id)
				t.Fatal("child not reaped")
			}
			r.mu.Lock()
			_, present := r.procs[id]
			r.mu.Unlock()
			if present {
				t.Fatal("dead process retained")
			}
			select {
			case <-exited:
				t.Fatal("duplicate exit callback")
			default:
			}
		})
	}
}

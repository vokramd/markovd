package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os/exec"
	"time"
)

// Process exit is a fallback for missing callbacks, never a reason to replay a run.
func (s *Server) shellProcessExited(runID string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	code := 0
	if err != nil {
		code = -1
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code = exit.ExitCode()
		}
	}
	// Do not include arguments, output or arbitrary error text: they can contain secrets.
	payload, _ := json.Marshal(map[string]any{"event_type": "run_failed", "run_id": runID, "timestamp": time.Now().UTC(), "source": "shell_runner", "exit_code": code, "error": "engine exited without a terminal callback"})
	if err := s.db.FailUnfinishedRun(ctx, runID, string(payload)); err != nil {
		log.Printf("recording shell exit for %s: %v", runID, err)
	}
}

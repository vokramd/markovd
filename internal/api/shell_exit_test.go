package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vokramd/markovd/internal/auth"
	"github.com/vokramd/markovd/internal/db"
	"github.com/vokramd/markovd/internal/runner"
)

type immediateExitRunner struct{ runner.Runner }

func (*immediateExitRunner) Start(_ context.Context, req runner.RunRequest) (string, error) {
	req.OnExit(req.RunID, fmt.Errorf("secret must not appear in event"))
	return req.RunID, nil
}

func TestShellExitRecordsRunBeforeLaunch(t *testing.T) {
	url := os.Getenv("MARKOVD_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("MARKOVD_TEST_DATABASE_URL not set")
	}
	d, err := db.New(url)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	ctx := context.Background()
	user, err := d.CreateUser(ctx, fmt.Sprintf("exit-api-%d", time.Now().UnixNano()), "fixture")
	if err != nil {
		t.Fatal(err)
	}
	wf, err := d.CreateWorkflow(ctx, user.Username, "entrypoint: main\nworkflows: []\n", user.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		runs, _ := d.ListRuns(ctx)
		for _, r := range runs {
			if r.WorkflowName == wf.Name {
				_ = d.DeleteRun(ctx, r.RunID)
			}
		}
		_ = d.DeleteWorkflow(ctx, wf.Name)
		_, _ = d.Exec(`DELETE FROM users WHERE id=$1`, user.ID)
	}()
	jwt := auth.NewJWTManager("fixture-only", time.Hour)
	token, _ := jwt.Issue(user.ID, user.Username)
	script := filepath.Join(t.TempDir(), "markov")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 17\n"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		r        runner.Runner
		wantHTTP int
	}{
		{"synchronous_exit", &immediateExitRunner{}, http.StatusCreated},
		{"real_shell_exit", runner.NewShellRunner(script), http.StatusCreated},
		{"launch_failure", runner.NewShellRunner(script + "-missing"), http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewServer(d, nil, jwt, tc.r, "", "callback-secret", "", t.TempDir())
			body, _ := json.Marshal(createRunRequest{WorkflowName: wf.Name})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			if rec.Code != tc.wantHTTP {
				t.Fatalf("HTTP %d: %s", rec.Code, rec.Body.String())
			}
			var response struct {
				RunID string `json:"run_id"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.RunID == "" {
				t.Fatal("missing durable run ID")
			}
			deadline := time.Now().Add(5 * time.Second)
			for {
				run, err := d.GetRunByID(ctx, response.RunID)
				if err != nil {
					t.Fatal(err)
				}
				if run != nil && run.Status == "failed" {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("run still unfinished: %+v", run)
				}
				time.Sleep(10 * time.Millisecond)
			}
			events, err := d.GetEventsByRunID(ctx, response.RunID)
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 1 || events[0].EventType != "run_failed" {
				t.Fatalf("events=%+v", events)
			}
			if strings.Contains(events[0].Payload, "secret") {
				t.Fatal("error or credential leaked into event")
			}
		})
	}
}

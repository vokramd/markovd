package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/vokramd/markovd/internal/auth"
	"github.com/vokramd/markovd/internal/db"
)

func TestRunEventsReplayARunWithItsSubRuns(t *testing.T) {
	url := os.Getenv("MARKOVD_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("MARKOVD_TEST_DATABASE_URL not set")
	}
	database, err := db.New(url)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	ctx := context.Background()
	stamp := time.Now().UnixNano()
	user, err := database.CreateUser(ctx, fmt.Sprintf("run-events-%d", stamp), "not-a-credential")
	if err != nil {
		t.Fatal(err)
	}
	wf, err := database.CreateWorkflow(ctx, user.Username, "entrypoint: main\nworkflows: []\n", user.ID)
	if err != nil {
		t.Fatal(err)
	}
	runID := fmt.Sprintf("markov-run-%x", stamp)
	if _, err = database.CreateRun(ctx, runID, wf.Name, wf.ID, user.ID, "{}", "[]", "[]"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = database.Exec(`DELETE FROM events WHERE run_id LIKE $1`, runID+"%")
		_, _ = database.Exec(`DELETE FROM runs WHERE triggered_by=$1`, user.ID)
		_ = database.DeleteWorkflow(ctx, wf.Name)
		_, _ = database.Exec(`DELETE FROM users WHERE id=$1`, user.ID)
	}()
	insert := func(id, typ, step string) {
		payload := fmt.Sprintf(`{"run_id": %q, "event_type": %q, "workflow_name": "main", "step_name": %q}`, id, typ, step)
		if _, err := database.InsertEvent(ctx, id, typ, payload); err != nil {
			t.Fatal(err)
		}
	}
	insert(runID, "run_started", "")
	insert(runID, "step_started", "fan")
	insert(runID+"-fan-0", "sub_run_started", "")
	insert(runID+"-fan-0", "step_completed", "leaf")
	insert(runID+"0", "step_started", "another run sharing the prefix")
	insert(runID, "run_completed", "")

	jwt := auth.NewJWTManager("test-only", time.Hour)
	token, _ := jwt.Issue(user.ID, user.Username)
	router := NewServer(database, nil, jwt, nil, "", "callback-token", "http://callback", t.TempDir()).Router()
	get := func(path string) (int, runEventsResponse) {
		req := httptest.NewRequest("GET", "/api/v1"+path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		var body runEventsResponse
		_ = json.Unmarshal(res.Body.Bytes(), &body)
		return res.Code, body
	}
	types := func(page runEventsResponse) []string {
		var out []string
		for _, e := range page.Events {
			var ev struct {
				Type string `json:"event_type"`
			}
			_ = json.Unmarshal(e.Event, &ev)
			out = append(out, ev.Type)
		}
		return out
	}

	code, page := get("/runs/" + runID + "/events?limit=3")
	if code != http.StatusOK || page.Status != "running" {
		t.Fatalf("code=%d status=%q", code, page.Status)
	}
	if got := types(page); fmt.Sprint(got) != "[run_started step_started sub_run_started]" {
		t.Fatalf("first page = %v", got)
	}
	code, page = get(fmt.Sprintf("/runs/%s/events?after=%d", runID, page.Next))
	if got := types(page); code != http.StatusOK || fmt.Sprint(got) != "[step_completed run_completed]" {
		t.Fatalf("second page = %d %v: the sub-run's events are included, another run's are not", code, got)
	}
	_, empty := get(fmt.Sprintf("/runs/%s/events?after=%d", runID, page.Next))
	if len(empty.Events) != 0 || empty.Next != page.Next {
		t.Fatalf("polling past the end returns nothing and keeps the cursor: %+v", empty)
	}
	if code, _ := get("/runs/markov-run-nosuchrun/events"); code != http.StatusNotFound {
		t.Fatalf("unknown run: %d", code)
	}
	if code, _ := get("/runs/" + runID + "/events?after=-1"); code != http.StatusBadRequest {
		t.Fatalf("bad cursor: %d", code)
	}
}

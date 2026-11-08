package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

const (
	defaultProgressLimit = 500
	maxProgressLimit     = 2000
)

type progressItem struct {
	ID         int            `json:"id"`
	RunID      string         `json:"run_id"`
	Kind       string         `json:"kind"`
	Data       map[string]any `json:"data"`
	ReceivedAt string         `json:"received_at"`
}

type progressResponse struct {
	Events []progressItem `json:"events"`
}

// handleStepProgress returns the step_progress events a step emitted while it
// ran. Query: step (required), workflow, fork, after (last seen event id),
// limit. Clients poll with after= to receive only new events.
func (s *Server) handleStepProgress(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	q := r.URL.Query()
	step := q.Get("step")
	if runID == "" || step == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "run ID and step are required"})
		return
	}
	after, err := parseNonNegative(q.Get("after"), 0)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid after parameter"})
		return
	}
	limit, err := parseNonNegative(q.Get("limit"), defaultProgressLimit)
	if err != nil || limit == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit parameter"})
		return
	}
	if limit > maxProgressLimit {
		limit = maxProgressLimit
	}

	root := rootRunID(runID)
	rows, err := s.db.ListStepProgress(r.Context(), root, eventRunID(root, q.Get("fork")), q.Get("workflow"), step, after, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list progress"})
		return
	}

	items := make([]progressItem, 0, len(rows))
	for _, row := range rows {
		var payload struct {
			Kind string         `json:"kind"`
			Data map[string]any `json:"data"`
		}
		if json.Unmarshal([]byte(row.Payload), &payload) != nil {
			continue
		}
		items = append(items, progressItem{
			ID:         row.ID,
			RunID:      row.RunID,
			Kind:       payload.Kind,
			Data:       payload.Data,
			ReceivedAt: row.ReceivedAt.Format("2006-01-02T15:04:05.000Z07:00"),
		})
	}
	writeJSON(w, http.StatusOK, progressResponse{Events: items})
}

// eventRunID is the run_id markov stamps on events for a fork: the root run
// ID plus the fork suffix. An empty fork means the root run itself.
func eventRunID(root, fork string) string {
	if fork == "" {
		return ""
	}
	return root + "-" + fork
}

func parseNonNegative(s string, def int) (int, error) {
	if s == "" {
		return def, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, strconv.ErrSyntax
	}
	return n, nil
}

type runEvent struct {
	ID    int             `json:"id"`
	Event json.RawMessage `json:"event"`
}

type runEventsResponse struct {
	Events []runEvent `json:"events"`
	// Next is the after= value for the next poll: the last event's ID, or the request's after.
	Next int `json:"next"`
	// Status is the run's status as markovd records it, so a client knows when no more events
	// will come even if the run ended without reporting it (a pod killed mid-run).
	Status string `json:"status"`
}

// handleRunEvents returns a run's callback events, its sub-runs' included, as markov sent
// them. Query: after (last seen event ID), limit. Clients poll with after= to follow a run.
func (s *Server) handleRunEvents(w http.ResponseWriter, r *http.Request) {
	root := rootRunID(chi.URLParam(r, "runID"))
	q := r.URL.Query()
	after, err := parseNonNegative(q.Get("after"), 0)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid after parameter"})
		return
	}
	limit, err := parseNonNegative(q.Get("limit"), defaultProgressLimit)
	if err != nil || limit == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit parameter"})
		return
	}
	if limit > maxProgressLimit {
		limit = maxProgressLimit
	}
	run, err := s.db.GetRunByID(r.Context(), root)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get run"})
		return
	}
	if run == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "run not found"})
		return
	}
	rows, err := s.db.ListRunEvents(r.Context(), root, after, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list events"})
		return
	}
	resp := runEventsResponse{Events: make([]runEvent, 0, len(rows)), Next: after, Status: run.Status}
	for _, row := range rows {
		resp.Events = append(resp.Events, runEvent{ID: row.ID, Event: json.RawMessage(row.Payload)})
		resp.Next = row.ID
	}
	writeJSON(w, http.StatusOK, resp)
}

package db

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestFailUnfinishedRunPreservesFinalState(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	for _, status := range []string{"pending", "running", "completed", "failed", "paused", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			id := fmt.Sprintf("exit-%s-%d", status, time.Now().UnixNano())
			_, err := d.Exec(`INSERT INTO runs(run_id,workflow_name,status) VALUES($1,'fixture',$2)`, id, status)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = d.DeleteRun(ctx, id) })
			for i := 0; i < 2; i++ {
				if err := d.FailUnfinishedRun(ctx, id, `{"source":"shell_runner","exit_code":17}`); err != nil {
					t.Fatal(err)
				}
			}
			run, err := d.GetRunByID(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			events, err := d.GetEventsByRunID(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			want := status
			count := 0
			if status == "pending" || status == "running" {
				want = "failed"
				count = 1
				if run.CompletedAt == nil {
					t.Fatal("no completion time")
				}
			}
			if run.Status != want || len(events) != count {
				t.Fatalf("status=%s events=%d; want %s/%d", run.Status, len(events), want, count)
			}
		})
	}
}

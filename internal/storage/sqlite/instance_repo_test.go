package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/open-apime/apime/internal/storage/model"
)

func newTestInstanceRepo(t *testing.T) *instanceRepo {
	t.Helper()
	conn, err := sql.Open("sqlite3", "file:"+filepath.Join(t.TempDir(), "test.db")+"?_journal_mode=WAL")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	files, err := filepath.Glob("../../../db/migrations/sqlite/*.up.sql")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	sort.Strings(files)
	for _, f := range files {
		schema, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if _, err := conn.Exec(string(schema)); err != nil {
			t.Fatalf("migrate %s: %v", f, err)
		}
	}
	return NewInstanceRepository(&DB{Conn: conn})
}

func TestUpdateStatusSurvivesStaleWholeRowCopy(t *testing.T) {
	ctx := context.Background()
	repo := newTestInstanceRepo(t)
	inst, err := repo.Create(ctx, model.Instance{ID: "inst-1", Name: "one", Status: model.InstanceStatusError})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.UpdateStatus(ctx, inst.ID, model.InstanceStatusActive); err != nil {
		t.Fatalf("update status: %v", err)
	}
	if err := repo.UpdateHistorySync(ctx, inst.ID, model.HistorySyncStatusRunning, nil, time.Now()); err != nil {
		t.Fatalf("update history sync: %v", err)
	}

	got, err := repo.GetByID(ctx, inst.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != model.InstanceStatusActive {
		t.Fatalf("status = %q, want active", got.Status)
	}
	if got.HistorySyncStatus != model.HistorySyncStatusRunning {
		t.Fatalf("history sync status = %q, want running", got.HistorySyncStatus)
	}
}

func TestUpdateHistorySyncKeepsCycleWhenNil(t *testing.T) {
	ctx := context.Background()
	repo := newTestInstanceRepo(t)
	if _, err := repo.Create(ctx, model.Instance{ID: "inst-1", Name: "one", Status: model.InstanceStatusActive}); err != nil {
		t.Fatalf("create: %v", err)
	}

	cycle := "cycle-1"
	if err := repo.UpdateHistorySync(ctx, "inst-1", model.HistorySyncStatusRunning, &cycle, time.Now()); err != nil {
		t.Fatalf("start cycle: %v", err)
	}
	if err := repo.UpdateHistorySync(ctx, "inst-1", model.HistorySyncStatusCompleted, nil, time.Now()); err != nil {
		t.Fatalf("finish cycle: %v", err)
	}

	got, _ := repo.GetByID(ctx, "inst-1")
	if got.HistorySyncCycleID != cycle {
		t.Fatalf("cycle = %q, want %q", got.HistorySyncCycleID, cycle)
	}
}

func TestUpdateStatusUnknownInstance(t *testing.T) {
	if err := newTestInstanceRepo(t).UpdateStatus(context.Background(), "missing", model.InstanceStatusActive); err == nil {
		t.Fatal("expected an error for a missing instance")
	}
}

package repo

import (
	"testing"

	"github.com/black-eleven/stock-monitor/internal/db"
	"github.com/black-eleven/stock-monitor/internal/model"
)

func setupAlertRepo(t *testing.T) *AlertRepo {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewAlertRepo(database)
}

func TestAlertCRUD(t *testing.T) {
	r := setupAlertRepo(t)

	rule := model.AlertRule{
		Symbol: "HK:700", Type: "above", Value: 500,
		Enabled: true, CreatedAt: nowISO(),
	}
	id, err := r.Add(rule)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if id != 1 {
		t.Errorf("expected id=1, got %d", id)
	}

	all, err := r.GetAll()
	if err != nil {
		t.Fatalf("getall: %v", err)
	}
	if len(all) != 1 || all[0].ID != 1 {
		t.Errorf("unexpected all: %+v", all)
	}

	err = r.Update(1, func(a *model.AlertRule) {
		a.Value = 600
		a.Enabled = false
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	updated, err := r.GetBySymbol("HK:700")
	if err != nil {
		t.Fatalf("getBySymbol: %v", err)
	}
	if updated[0].Value != 600 || updated[0].Enabled {
		t.Errorf("unexpected updated: %+v", updated[0])
	}

	if err := r.Remove(1); err != nil {
		t.Fatalf("remove: %v", err)
	}
	all, _ = r.GetAll()
	if len(all) != 0 {
		t.Errorf("expected empty after delete, got %d", len(all))
	}
}

func TestAlertLog(t *testing.T) {
	r := setupAlertRepo(t)
	err := r.AppendLog(model.AlertLog{
		AlertID: 1, Symbol: "HK:700", Price: 495,
		Message: "跌破 500", TriggeredAt: nowISO(),
	})
	if err != nil {
		t.Fatalf("appendLog: %v", err)
	}
	logs, err := r.GetLogs(50)
	if err != nil {
		t.Fatalf("getLogs: %v", err)
	}
	if len(logs) != 1 || logs[0].Message != "跌破 500" {
		t.Errorf("unexpected logs: %+v", logs)
	}
}

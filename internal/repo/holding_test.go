package repo

import (
	"testing"

	"github.com/black-eleven/stock-monitor/internal/db"
	"github.com/black-eleven/stock-monitor/internal/model"
)

func setupHoldingRepo(t *testing.T) *HoldingRepo {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewHoldingRepo(database)
}

func TestHoldingCRUD(t *testing.T) {
	r := setupHoldingRepo(t)

	h := model.Holding{Symbol: "HK:1810", Name: "Xiaomi", Shares: 600, AvgCost: 44.4, BuyDate: "2026-01-15"}
	if err := r.Add(1, h); err != nil {
		t.Fatalf("add: %v", err)
	}

	all, err := r.GetAll(1)
	if err != nil {
		t.Fatalf("getall: %v", err)
	}
	if len(all) != 1 || all[0].Shares != 600 {
		t.Errorf("unexpected all: %+v", all)
	}

	err = r.Update(1, "HK:1810", func(h *model.Holding) {
		h.Shares = 800
		h.AvgCost = 42.5
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	all, _ = r.GetAll(1)
	if all[0].Shares != 800 || all[0].AvgCost != 42.5 {
		t.Errorf("unexpected updated: %+v", all[0])
	}

	if err := r.Remove(1, "HK:1810"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	all, _ = r.GetAll(1)
	if len(all) != 0 {
		t.Errorf("expected empty after delete")
	}
}

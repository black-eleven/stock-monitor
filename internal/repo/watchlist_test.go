package repo

import (
	"testing"
	"time"

	"github.com/black-eleven/stock-monitor/internal/db"
	"github.com/black-eleven/stock-monitor/internal/model"
)

func setupWatchlistRepo(t *testing.T) *WatchlistRepo {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewWatchlistRepo(database)
}

func TestAddAndGetAll(t *testing.T) {
	r := setupWatchlistRepo(t)
	item := model.WatchlistItem{Symbol: "HK:700", Name: "Tencent", AddedAt: nowISO()}
	if err := r.Add(1, item); err != nil {
		t.Fatalf("add: %v", err)
	}
	all, err := r.GetAll(1)
	if err != nil {
		t.Fatalf("getall: %v", err)
	}
	if len(all) != 1 || all[0].Symbol != "HK:700" {
		t.Errorf("unexpected result: %+v", all)
	}
}

func TestAddDuplicate(t *testing.T) {
	r := setupWatchlistRepo(t)
	item := model.WatchlistItem{Symbol: "HK:700", Name: "Tencent", AddedAt: nowISO()}
	r.Add(1, item)
	err := r.Add(1, item)
	if err != ErrDuplicate {
		t.Errorf("expected ErrDuplicate, got %v", err)
	}
}

func TestRemove(t *testing.T) {
	r := setupWatchlistRepo(t)
	r.Add(1, model.WatchlistItem{Symbol: "HK:700", Name: "Tencent", AddedAt: nowISO()})
	if err := r.Remove(1, "HK:700"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	all, _ := r.GetAll(1)
	if len(all) != 0 {
		t.Errorf("expected empty, got %+v", all)
	}
}

func TestRemoveNotFound(t *testing.T) {
	r := setupWatchlistRepo(t)
	err := r.Remove(1, "NONEXIST")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

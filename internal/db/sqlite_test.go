package db

import (
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	defer rows.Close()

	tables := make(map[string]bool)
	for rows.Next() {
		var name string
		rows.Scan(&name)
		tables[name] = true
	}
	for _, want := range []string{"alerts", "alert_logs", "holdings", "watchlist"} {
		if !tables[want] {
			t.Errorf("table %s not found", want)
		}
	}
}

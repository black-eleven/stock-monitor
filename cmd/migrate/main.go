package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/black-eleven/stock-monitor/internal/config"
	"github.com/black-eleven/stock-monitor/internal/db"
	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	cfg := config.Load()

	database, err := db.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	migrateUserIDColumn(database)
	migrateWatchlist(database)
	migrateHoldings(database)
	migrateAlerts(database)

	fmt.Println("Migration complete!")
}

func migrateWatchlist(database *sql.DB) {
	r := repo.NewWatchlistRepo(database)
	data := readJSON("watchlist.json")
	if data == nil {
		return
	}
	var items []model.WatchlistItem
	if err := json.Unmarshal(data, &items); err != nil {
		fmt.Fprintf(os.Stderr, "watchlist parse error: %v\n", err)
		return
	}
	for _, item := range items {
		if err := r.Add(0, item); err != nil {
			fmt.Printf("watchlist skip %s: %v\n", item.Symbol, err)
		} else {
			fmt.Printf("watchlist added: %s\n", item.Symbol)
		}
	}
}

func migrateHoldings(database *sql.DB) {
	r := repo.NewHoldingRepo(database)
	data := readJSON("holdings.json")
	if data == nil {
		return
	}
	var items []model.Holding
	if err := json.Unmarshal(data, &items); err != nil {
		fmt.Fprintf(os.Stderr, "holdings parse error: %v\n", err)
		return
	}
	for _, h := range items {
		if err := r.Add(0, h); err != nil {
			fmt.Printf("holdings skip %s: %v\n", h.Symbol, err)
		} else {
			fmt.Printf("holdings added: %s\n", h.Symbol)
		}
	}
}

func migrateAlerts(database *sql.DB) {
	r := repo.NewAlertRepo(database)
	data := readJSON("alerts.json")
	if data == nil {
		return
	}
	var items []model.AlertRule
	if err := json.Unmarshal(data, &items); err != nil {
		fmt.Fprintf(os.Stderr, "alerts parse error: %v\n", err)
		return
	}
	for _, a := range items {
		if _, err := r.Add(0, a); err != nil {
			fmt.Printf("alerts skip %s: %v\n", a.Symbol, err)
		} else {
			fmt.Printf("alerts added: %s (%s)\n", a.Symbol, a.Type)
		}
	}
}

func migrateUserIDColumn(database *sql.DB) {
	alterStatements := []string{
		"ALTER TABLE watchlist ADD COLUMN user_id INTEGER DEFAULT 0",
		"ALTER TABLE holdings ADD COLUMN user_id INTEGER DEFAULT 0",
		"ALTER TABLE alerts ADD COLUMN user_id INTEGER DEFAULT 0",
	}
	for _, stmt := range alterStatements {
		_, err := database.Exec(stmt)
		if err != nil {
			fmt.Printf("migration note: %v (column may already exist)\n", err)
		} else {
			fmt.Printf("migrated: %s\n", stmt)
		}
	}
}

func readJSON(name string) []byte {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	path := filepath.Join(dataDir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("skip %s: %v\n", name, err)
		return nil
	}
	return data
}

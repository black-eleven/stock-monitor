package db

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const schema = `
CREATE TABLE IF NOT EXISTS watchlist (
    symbol   TEXT PRIMARY KEY,
    name     TEXT NOT NULL,
    added_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS holdings (
    symbol   TEXT PRIMARY KEY,
    name     TEXT NOT NULL,
    shares   REAL NOT NULL,
    avg_cost REAL NOT NULL,
    buy_date TEXT
);

CREATE TABLE IF NOT EXISTS alerts (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol             TEXT NOT NULL,
    type               TEXT NOT NULL,
    value              REAL NOT NULL,
    enabled            INTEGER DEFAULT 1,
    created_at         TEXT NOT NULL,
    last_triggered_at  TEXT
);

CREATE TABLE IF NOT EXISTS alert_logs (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    alert_id     INTEGER NOT NULL,
    symbol       TEXT NOT NULL,
    price        REAL NOT NULL,
    message      TEXT NOT NULL,
    triggered_at TEXT NOT NULL
);
`

func Open(dataDir string) (*sql.DB, error) {
	if dataDir != ":memory:" {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, err
		}
	}
	dbPath := filepath.Join(dataDir, "stock-monitor.db")
	if dataDir == ":memory:" {
		dbPath = ":memory:"
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

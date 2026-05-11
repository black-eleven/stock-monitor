package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"

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

CREATE TABLE IF NOT EXISTS users (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    username   TEXT NOT NULL UNIQUE,
    password   TEXT NOT NULL,
    role       TEXT DEFAULT 'user',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS invite_codes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    code       TEXT NOT NULL UNIQUE,
    max_uses   INTEGER DEFAULT 1,
    used_count INTEGER DEFAULT 0,
    created_by INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    is_active  INTEGER DEFAULT 1
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

	// Migration: add user_id to existing tables (SQLite ADD COLUMN is idempotent-safe)
	migrations := []string{
		"ALTER TABLE watchlist ADD COLUMN user_id INTEGER DEFAULT 0",
		"ALTER TABLE holdings ADD COLUMN user_id INTEGER DEFAULT 0",
		"ALTER TABLE alerts ADD COLUMN user_id INTEGER DEFAULT 0",
	}
	for _, m := range migrations {
		db.Exec(m) // ignore errors (column may already exist)
	}

	return db, nil
}

func generateInviteCode() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func InitAdmin(database *sql.DB, password string, explicitPassword bool) (int, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return 0, fmt.Errorf("bcrypt: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	var count int
	database.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)

	if count > 0 {
		// Admin exists — update password if explicitly set via env
		if explicitPassword {
			database.Exec("UPDATE users SET password = ? WHERE role = 'admin'", string(hash))
			log.Printf("[DB] Admin password updated from ADMIN_PASSWORD env var")
		}
		return 0, nil
	}

	result, err := database.Exec(
		"INSERT INTO users (username, password, role, created_at) VALUES (?, ?, 'admin', ?)",
		"admin", string(hash), now,
	)
	if err != nil {
		return 0, fmt.Errorf("create admin: %w", err)
	}
	id, _ := result.LastInsertId()

	code := generateInviteCode()
	database.Exec(
		"INSERT INTO invite_codes (code, max_uses, used_count, created_by, created_at, is_active) VALUES (?, 1, 0, ?, ?, 1)",
		code, id, now,
	)
	log.Printf("[DB] Created admin user. Initial invite code: %s", code)

	return int(id), nil
}

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
    symbol   TEXT NOT NULL,
    name     TEXT NOT NULL,
    added_at TEXT NOT NULL,
    user_id  INTEGER DEFAULT 0,
    PRIMARY KEY (symbol, user_id)
);

CREATE TABLE IF NOT EXISTS holdings (
    symbol   TEXT NOT NULL,
    name     TEXT NOT NULL,
    shares   REAL NOT NULL,
    avg_cost REAL NOT NULL,
    buy_date TEXT,
    user_id  INTEGER DEFAULT 0,
    PRIMARY KEY (symbol, user_id)
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


		// Migration: rebuild tables with composite PK (symbol, user_id) for multi-user
		migrateCompositePK(db, "watchlist",
			"CREATE TABLE watchlist_new (symbol TEXT NOT NULL, name TEXT NOT NULL, added_at TEXT NOT NULL, user_id INTEGER DEFAULT 0, PRIMARY KEY (symbol, user_id))",
			"symbol, name, added_at, COALESCE(user_id, 0)",
		)
		migrateCompositePK(db, "holdings",
			"CREATE TABLE holdings_new (symbol TEXT NOT NULL, name TEXT NOT NULL, shares REAL NOT NULL, avg_cost REAL NOT NULL, buy_date TEXT, user_id INTEGER DEFAULT 0, PRIMARY KEY (symbol, user_id))",
			"symbol, name, shares, avg_cost, buy_date, COALESCE(user_id, 0)",
		)
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

// migrateCompositePK rebuilds a table with a composite primary key (col, user_id).
// SQLite does not support ALTER TABLE to change PKs, so we recreate the table.
func migrateCompositePK(database *sql.DB, tableName, createNewSQL, copyColumns string) {
	// Check if migration is needed by looking for user_id in the PK
	rows, err := database.Query("SELECT name FROM pragma_table_info(? || '_new')", tableName)
	if err == nil {
		rows.Close()
		return // new table already exists, migration done
	}

	// Check if current table already has composite PK
	pkCount := 0
	infoRows, err := database.Query("SELECT pk FROM pragma_table_info(?) WHERE pk > 0", tableName)
	if err != nil {
		return
	}
	for infoRows.Next() {
		pkCount++
	}
	infoRows.Close()
	if pkCount > 1 {
		return // already composite PK
	}

	// Rebuild: create new table -> copy data -> drop old -> rename new
	if _, err := database.Exec(createNewSQL); err != nil {
		log.Printf("[DB] migrate %s: create new table: %v", tableName, err)
		return
	}
	database.Exec("INSERT INTO " + tableName + "_new (" + copyColumns + ") SELECT " + copyColumns + " FROM " + tableName)
	database.Exec("DROP TABLE " + tableName)
	database.Exec("ALTER TABLE " + tableName + "_new RENAME TO " + tableName)
	log.Printf("[DB] Migrated %s to composite PK (col, user_id)", tableName)
}

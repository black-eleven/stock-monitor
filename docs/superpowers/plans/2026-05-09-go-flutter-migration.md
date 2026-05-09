# Go Backend + Flutter Mobile Migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite Node.js backend in Go with SQLite storage, keep web frontend, and build Flutter mobile app (iOS + Android).

**Architecture:** Gin HTTP server → repo layer → SQLite. Gorilla WebSocket for QOS market data client and browser/app push. Flutter app with Riverpod state management, dio HTTP client, and fl_chart candlestick charts.

**Tech Stack:** Go 1.22+, Gin, gorilla/websocket, mattn/go-sqlite3, Flutter 3.x, Riverpod, dio, fl_chart, go_router

---

## Phase 1: Go Backend Foundation

### Task 1: Initialize Go module and project skeleton

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `cmd/server/main.go` (minimal)

- [ ] **Step 1: Initialize Go module**

Run: `cd /data1/workspace/yihuiwen/workspace/src/github.com/black-eleven/stock-monitor && go mod init github.com/black-eleven/stock-monitor`

Expected: `go: creating new go.mod: module github.com/black-eleven/stock-monitor`

- [ ] **Step 2: Create Makefile**

```makefile
.PHONY: run build test migrate

run:
	go run ./cmd/server

build:
	go build -o bin/stock-monitor ./cmd/server

test:
	go test ./internal/... -v

migrate:
	go run ./cmd/migrate
```

- [ ] **Step 3: Create minimal main.go**

```go
// cmd/server/main.go
package main

import "fmt"

func main() {
	fmt.Println("Stock Monitor starting...")
}
```

- [ ] **Step 4: Verify**

Run: `go run ./cmd/server`
Expected: `Stock Monitor starting...`

- [ ] **Step 5: Commit**

```bash
git add go.mod Makefile cmd/
git commit -m "feat: initialize Go module and project skeleton"
```

---

### Task 2: Config package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write config test**

```go
// internal/config/config_test.go
package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	os.Unsetenv("PORT")
	os.Unsetenv("QOS_KEY")
	os.Unsetenv("DATA_DIR")

	cfg := Load()
	if cfg.Port != "3000" {
		t.Errorf("expected port 3000, got %s", cfg.Port)
	}
	if cfg.DataDir != "data" {
		t.Errorf("expected data dir 'data', got %s", cfg.DataDir)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("PORT", "4000")
	os.Setenv("QOS_KEY", "test-key")
	defer os.Unsetenv("PORT")
	defer os.Unsetenv("QOS_KEY")

	cfg := Load()
	if cfg.Port != "4000" {
		t.Errorf("expected port 4000, got %s", cfg.Port)
	}
	if cfg.QosKey != "test-key" {
		t.Errorf("expected qos key 'test-key', got %s", cfg.QosKey)
	}
	if cfg.QosWsUrl != "wss://api.qos.hk/ws?key=test-key" {
		t.Errorf("unexpected ws url: %s", cfg.QosWsUrl)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/... -v`
Expected: FAIL — package not found

- [ ] **Step 3: Implement config**

```go
// internal/config/config.go
package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Port    string
	QosKey  string
	DataDir string
	QosWsUrl string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	qosKey := os.Getenv("QOS_KEY")
	qosWsUrl := "wss://api.qos.hk/ws"
	if qosKey != "" {
		qosWsUrl += "?key=" + qosKey
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	absDataDir, _ := filepath.Abs(dataDir)
	return &Config{
		Port:    port,
		QosKey:  qosKey,
		DataDir: absDataDir,
		QosWsUrl: qosWsUrl,
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/config/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add config package with env loading"
```

---

### Task 3: Domain models

**Files:**
- Create: `internal/model/watchlist.go`
- Create: `internal/model/alert.go`
- Create: `internal/model/holding.go`
- Create: `internal/model/quote.go`

- [ ] **Step 1: Write all model files**

```go
// internal/model/watchlist.go
package model

type WatchlistItem struct {
	Symbol  string `json:"symbol"`
	Name    string `json:"name"`
	AddedAt string `json:"addedAt"`
}
```

```go
// internal/model/alert.go
package model

type AlertRule struct {
	ID               int     `json:"id"`
	Symbol           string  `json:"symbol"`
	Type             string  `json:"type"` // "above" | "below" | "change_pct"
	Value            float64 `json:"value"`
	Enabled          bool    `json:"enabled"`
	CreatedAt        string  `json:"createdAt"`
	LastTriggeredAt  *string `json:"lastTriggeredAt"`
}

type AlertLog struct {
	ID          int    `json:"id"`
	AlertID     int    `json:"alertId"`
	Symbol      string `json:"symbol"`
	Price       float64 `json:"price"`
	Message     string `json:"message"`
	TriggeredAt string `json:"triggeredAt"`
}
```

```go
// internal/model/holding.go
package model

type Holding struct {
	Symbol  string  `json:"symbol"`
	Name    string  `json:"name"`
	Shares  float64 `json:"shares"`
	AvgCost float64 `json:"avgCost"`
	BuyDate string  `json:"buyDate"`
}
```

```go
// internal/model/quote.go
package model

type Quote struct {
	Code      string  `json:"code"`
	Price     float64 `json:"price"`
	YP        float64 `json:"yp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"volume"`
	Turnover  float64 `json:"turnover"`
	Timestamp int64   `json:"timestamp"`
	Status    string  `json:"status"`
}

type KlineBar struct {
	Ts int64   `json:"ts"`
	O  float64 `json:"o"`
	Cl float64 `json:"cl"`
	H  float64 `json:"h"`
	L  float64 `json:"l"`
	V  float64 `json:"v"`
}

type KlineItem struct {
	C string     `json:"c"`
	K []KlineBar `json:"k"`
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/model/...`
Expected: (no output — compiles silently)

- [ ] **Step 3: Commit**

```bash
git add internal/model/
git commit -m "feat: add domain model types"
```

---

### Task 4: SQLite database initialization + migrations

**Files:**
- Create: `internal/db/sqlite.go`
- Create: `internal/db/sqlite_test.go`

- [ ] **Step 1: Write test**

```go
// internal/db/sqlite_test.go
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

	// Verify tables exist by querying sqlite_master
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
```

- [ ] **Step 2: Run test (should fail)**

Run: `go test ./internal/db/... -v`
Expected: FAIL — `Open` not defined

- [ ] **Step 3: Implement SQLite with embedded migration**

```go
// internal/db/sqlite.go
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
```

- [ ] **Step 4: Install SQLite dependency**

Run: `go get github.com/mattn/go-sqlite3`

- [ ] **Step 5: Run tests**

Run: `go test ./internal/db/... -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/db/ go.mod go.sum
git commit -m "feat: add SQLite database with auto-migration"
```

---

### Task 5: Watchlist repository

**Files:**
- Create: `internal/repo/watchlist.go`
- Create: `internal/repo/watchlist_test.go`

- [ ] **Step 1: Write test**

```go
// internal/repo/watchlist_test.go
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
	if err := r.Add(item); err != nil {
		t.Fatalf("add: %v", err)
	}
	all, err := r.GetAll()
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
	r.Add(item)
	err := r.Add(item)
	if err != ErrDuplicate {
		t.Errorf("expected ErrDuplicate, got %v", err)
	}
}

func TestRemove(t *testing.T) {
	r := setupWatchlistRepo(t)
	r.Add(model.WatchlistItem{Symbol: "HK:700", Name: "Tencent", AddedAt: nowISO()})
	if err := r.Remove("HK:700"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	all, _ := r.GetAll()
	if len(all) != 0 {
		t.Errorf("expected empty, got %+v", all)
	}
}

func TestRemoveNotFound(t *testing.T) {
	r := setupWatchlistRepo(t)
	err := r.Remove("NONEXIST")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}
```

- [ ] **Step 2: Run test (should fail)**

Run: `go test ./internal/repo/... -v`
Expected: FAIL — `WatchlistRepo` not defined

- [ ] **Step 3: Implement watchlist repo**

```go
// internal/repo/watchlist.go
package repo

import (
	"database/sql"
	"errors"

	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/mattn/go-sqlite3"
)

var (
	ErrDuplicate = errors.New("item already exists")
	ErrNotFound  = errors.New("item not found")
)

type WatchlistRepo struct {
	db *sql.DB
}

func NewWatchlistRepo(db *sql.DB) *WatchlistRepo {
	return &WatchlistRepo{db: db}
}

func (r *WatchlistRepo) GetAll() ([]model.WatchlistItem, error) {
	rows, err := r.db.Query("SELECT symbol, name, added_at FROM watchlist ORDER BY added_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.WatchlistItem
	for rows.Next() {
		var item model.WatchlistItem
		if err := rows.Scan(&item.Symbol, &item.Name, &item.AddedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if items == nil {
		items = []model.WatchlistItem{}
	}
	return items, nil
}

func (r *WatchlistRepo) Add(item model.WatchlistItem) error {
	_, err := r.db.Exec(
		"INSERT INTO watchlist (symbol, name, added_at) VALUES (?, ?, ?)",
		item.Symbol, item.Name, item.AddedAt,
	)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

func (r *WatchlistRepo) Remove(symbol string) error {
	result, err := r.db.Exec("DELETE FROM watchlist WHERE symbol = ?", symbol)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/repo/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/repo/watchlist.go internal/repo/watchlist_test.go
git commit -m "feat: add watchlist repository with CRUD"
```

---

### Task 6: Alert repository

**Files:**
- Create: `internal/repo/alert.go`
- Create: `internal/repo/alert_test.go`

- [ ] **Step 1: Write test**

```go
// internal/repo/alert_test.go
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

	// Add
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

	// GetAll
	all, err := r.GetAll()
	if err != nil {
		t.Fatalf("getall: %v", err)
	}
	if len(all) != 1 || all[0].ID != 1 {
		t.Errorf("unexpected all: %+v", all)
	}

	// Update
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

	// Delete
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
```

- [ ] **Step 2: Run test (should fail)**

Run: `go test ./internal/repo/... -v`
Expected: FAIL — `AlertRepo` not defined

- [ ] **Step 3: Implement alert repo**

```go
// internal/repo/alert.go
package repo

import (
	"database/sql"

	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/mattn/go-sqlite3"
)

type AlertRepo struct {
	db *sql.DB
}

func NewAlertRepo(db *sql.DB) *AlertRepo {
	return &AlertRepo{db: db}
}

func (r *AlertRepo) GetAll() ([]model.AlertRule, error) {
	rows, err := r.db.Query("SELECT id, symbol, type, value, enabled, created_at, last_triggered_at FROM alerts ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []model.AlertRule
	for rows.Next() {
		var a model.AlertRule
		var enabled int
		if err := rows.Scan(&a.ID, &a.Symbol, &a.Type, &a.Value, &enabled, &a.CreatedAt, &a.LastTriggeredAt); err != nil {
			return nil, err
		}
		a.Enabled = enabled != 0
		rules = append(rules, a)
	}
	if rules == nil {
		rules = []model.AlertRule{}
	}
	return rules, nil
}

func (r *AlertRepo) GetBySymbol(symbol string) ([]model.AlertRule, error) {
	rows, err := r.db.Query(
		"SELECT id, symbol, type, value, enabled, created_at, last_triggered_at FROM alerts WHERE symbol = ?",
		symbol,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []model.AlertRule
	for rows.Next() {
		var a model.AlertRule
		var enabled int
		if err := rows.Scan(&a.ID, &a.Symbol, &a.Type, &a.Value, &enabled, &a.CreatedAt, &a.LastTriggeredAt); err != nil {
			return nil, err
		}
		a.Enabled = enabled != 0
		rules = append(rules, a)
	}
	if rules == nil {
		rules = []model.AlertRule{}
	}
	return rules, nil
}

func (r *AlertRepo) Add(rule model.AlertRule) (int, error) {
	enabled := 0
	if rule.Enabled {
		enabled = 1
	}
	result, err := r.db.Exec(
		"INSERT INTO alerts (symbol, type, value, enabled, created_at, last_triggered_at) VALUES (?, ?, ?, ?, ?, ?)",
		rule.Symbol, rule.Type, rule.Value, enabled, rule.CreatedAt, rule.LastTriggeredAt,
	)
	if err != nil {
		var sqliteErr sqlite3.Error
		if isConstraintErr(err, &sqliteErr) {
			return 0, ErrDuplicate
		}
		return 0, err
	}
	id, _ := result.LastInsertId()
	return int(id), nil
}

func (r *AlertRepo) Update(id int, fn func(*model.AlertRule)) error {
	rules, err := r.GetAll()
	if err != nil {
		return err
	}
	for i, a := range rules {
		if a.ID == id {
			fn(&rules[i])
			enabled := 0
			if rules[i].Enabled {
				enabled = 1
			}
			_, err = r.db.Exec(
				"UPDATE alerts SET type=?, value=?, enabled=?, last_triggered_at=? WHERE id=?",
				rules[i].Type, rules[i].Value, enabled, rules[i].LastTriggeredAt, id,
			)
			return err
		}
	}
	return ErrNotFound
}

func (r *AlertRepo) Remove(id int) error {
	result, err := r.db.Exec("DELETE FROM alerts WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *AlertRepo) AppendLog(log model.AlertLog) error {
	_, err := r.db.Exec(
		"INSERT INTO alert_logs (alert_id, symbol, price, message, triggered_at) VALUES (?, ?, ?, ?, ?)",
		log.AlertID, log.Symbol, log.Price, log.Message, log.TriggeredAt,
	)
	return err
}

func (r *AlertRepo) GetLogs(limit int) ([]model.AlertLog, error) {
	rows, err := r.db.Query(
		"SELECT id, alert_id, symbol, price, message, triggered_at FROM alert_logs ORDER BY id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.AlertLog
	for rows.Next() {
		var l model.AlertLog
		if err := rows.Scan(&l.ID, &l.AlertID, &l.Symbol, &l.Price, &l.Message, &l.TriggeredAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []model.AlertLog{}
	}
	return logs, nil
}

func (r *AlertRepo) PurgeOldLogs(keep int) error {
	_, err := r.db.Exec(
		"DELETE FROM alert_logs WHERE id NOT IN (SELECT id FROM alert_logs ORDER BY id DESC LIMIT ?)",
		keep,
	)
	return err
}

func isConstraintErr(err error, sqliteErr *sqlite3.Error) bool {
	var e sqlite3.Error
	if errors.As(err, &e) {
		*sqliteErr = e
		return e.Code == sqlite3.ErrConstraint
	}
	return false
}
```

Note: Add `"errors"` to the import block.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/repo/... -v`
Expected: PASS (all tests: watchlist + alert)

- [ ] **Step 5: Commit**

```bash
git add internal/repo/alert.go internal/repo/alert_test.go
git commit -m "feat: add alert repository with CRUD and log management"
```

---

### Task 7: Holding repository

**Files:**
- Create: `internal/repo/holding.go`
- Create: `internal/repo/holding_test.go`

- [ ] **Step 1: Write test**

```go
// internal/repo/holding_test.go
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

	// Add
	h := model.Holding{Symbol: "HK:1810", Name: "Xiaomi", Shares: 600, AvgCost: 44.4, BuyDate: "2026-01-15"}
	if err := r.Add(h); err != nil {
		t.Fatalf("add: %v", err)
	}

	// GetAll
	all, err := r.GetAll()
	if err != nil {
		t.Fatalf("getall: %v", err)
	}
	if len(all) != 1 || all[0].Shares != 600 {
		t.Errorf("unexpected all: %+v", all)
	}

	// Update
	err = r.Update("HK:1810", func(h *model.Holding) {
		h.Shares = 800
		h.AvgCost = 42.5
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	all, _ = r.GetAll()
	if all[0].Shares != 800 || all[0].AvgCost != 42.5 {
		t.Errorf("unexpected updated: %+v", all[0])
	}

	// Remove
	if err := r.Remove("HK:1810"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	all, _ = r.GetAll()
	if len(all) != 0 {
		t.Errorf("expected empty after delete")
	}
}
```

- [ ] **Step 2: Implement holding repo**

```go
// internal/repo/holding.go
package repo

import (
	"database/sql"

	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/mattn/go-sqlite3"
)

type HoldingRepo struct {
	db *sql.DB
}

func NewHoldingRepo(db *sql.DB) *HoldingRepo {
	return &HoldingRepo{db: db}
}

func (r *HoldingRepo) GetAll() ([]model.Holding, error) {
	rows, err := r.db.Query("SELECT symbol, name, shares, avg_cost, buy_date FROM holdings ORDER BY symbol")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.Holding
	for rows.Next() {
		var h model.Holding
		var buyDate sql.NullString
		if err := rows.Scan(&h.Symbol, &h.Name, &h.Shares, &h.AvgCost, &buyDate); err != nil {
			return nil, err
		}
		h.BuyDate = buyDate.String
		items = append(items, h)
	}
	if items == nil {
		items = []model.Holding{}
	}
	return items, nil
}

func (r *HoldingRepo) Add(h model.Holding) error {
	_, err := r.db.Exec(
		"INSERT INTO holdings (symbol, name, shares, avg_cost, buy_date) VALUES (?, ?, ?, ?, ?)",
		h.Symbol, h.Name, h.Shares, h.AvgCost, h.BuyDate,
	)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

func (r *HoldingRepo) Update(symbol string, fn func(*model.Holding)) error {
	items, err := r.GetAll()
	if err != nil {
		return err
	}
	for i, h := range items {
		if h.Symbol == symbol {
			fn(&items[i])
			_, err = r.db.Exec(
				"UPDATE holdings SET shares=?, avg_cost=?, buy_date=?, name=? WHERE symbol=?",
				items[i].Shares, items[i].AvgCost, items[i].BuyDate, items[i].Name, symbol,
			)
			return err
		}
	}
	return ErrNotFound
}

func (r *HoldingRepo) Remove(symbol string) error {
	result, err := r.db.Exec("DELETE FROM holdings WHERE symbol = ?", symbol)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
```

Note: Add `"errors"` to the import block.

- [ ] **Step 3: Run tests**

Run: `go test ./internal/repo/... -v`
Expected: PASS (all 3 repos passing)

- [ ] **Step 4: Commit**

```bash
git add internal/repo/holding.go internal/repo/holding_test.go
git commit -m "feat: add holding repository with CRUD"
```

---

### Task 8: HTTP handlers — Watchlist + Alert + Holding

**Files:**
- Create: `internal/handler/watchlist.go`
- Create: `internal/handler/alert.go`
- Create: `internal/handler/holding.go`

- [ ] **Step 1: Implement watchlist handler**

```go
// internal/handler/watchlist.go
package handler

import (
	"net/http"
	"time"

	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/gin-gonic/gin"
)

type WatchlistHandler struct {
	repo *repo.WatchlistRepo
}

func NewWatchlistHandler(r *repo.WatchlistRepo) *WatchlistHandler {
	return &WatchlistHandler{repo: r}
}

func (h *WatchlistHandler) Register(api *gin.RouterGroup) {
	api.GET("/watchlist", h.getAll)
	api.POST("/watchlist", h.add)
	api.DELETE("/watchlist/:symbol", h.remove)
}

func (h *WatchlistHandler) getAll(c *gin.Context) {
	items, err := h.repo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch watchlist"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *WatchlistHandler) add(c *gin.Context) {
	var req struct {
		Symbol string `json:"symbol"`
		Name   string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Symbol == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol and name are required"})
		return
	}
	item := model.WatchlistItem{
		Symbol:  req.Symbol,
		Name:    req.Name,
		AddedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := h.repo.Add(item); err != nil {
		if err == repo.ErrDuplicate {
			c.JSON(http.StatusConflict, gin.H{"error": "Symbol already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add"})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *WatchlistHandler) remove(c *gin.Context) {
	symbol := c.Param("symbol")
	if err := h.repo.Remove(symbol); err != nil {
		if err == repo.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Symbol not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
```

- [ ] **Step 2: Implement alert handler**

```go
// internal/handler/alert.go
package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/gin-gonic/gin"
)

type AlertHandler struct {
	repo *repo.AlertRepo
}

func NewAlertHandler(r *repo.AlertRepo) *AlertHandler {
	return &AlertHandler{repo: r}
}

func (h *AlertHandler) Register(api *gin.RouterGroup) {
	api.GET("/alerts", h.getAll)
	api.POST("/alerts", h.add)
	api.PUT("/alerts/:id", h.update)
	api.DELETE("/alerts/:id", h.remove)
}

func (h *AlertHandler) getAll(c *gin.Context) {
	rules, err := h.repo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch alerts"})
		return
	}
	c.JSON(http.StatusOK, rules)
}

func (h *AlertHandler) add(c *gin.Context) {
	var req struct {
		Symbol string  `json:"symbol"`
		Type   string  `json:"type"`
		Value  float64 `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Symbol == "" || req.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol, type, and value are required"})
		return
	}
	if req.Type != "above" && req.Type != "below" && req.Type != "change_pct" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be above, below, or change_pct"})
		return
	}
	rule := model.AlertRule{
		Symbol:    req.Symbol,
		Type:      req.Type,
		Value:     req.Value,
		Enabled:   true,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	id, err := h.repo.Add(rule)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add alert"})
		return
	}
	rule.ID = id
	c.JSON(http.StatusCreated, rule)
}

func (h *AlertHandler) update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid body"})
		return
	}
	err = h.repo.Update(id, func(a *model.AlertRule) {
		if v, ok := req["type"]; ok {
			a.Type = v.(string)
		}
		if v, ok := req["value"]; ok {
			a.Value = v.(float64)
		}
		if v, ok := req["enabled"]; ok {
			a.Enabled = v.(bool)
		}
	})
	if err == repo.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *AlertHandler) remove(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	if err := h.repo.Remove(id); err != nil {
		if err == repo.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
```

- [ ] **Step 3: Implement holding handler**

```go
// internal/handler/holding.go
package handler

import (
	"net/http"
	"time"

	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/gin-gonic/gin"
)

type HoldingHandler struct {
	repo *repo.HoldingRepo
}

func NewHoldingHandler(r *repo.HoldingRepo) *HoldingHandler {
	return &HoldingHandler{repo: r}
}

func (h *HoldingHandler) Register(api *gin.RouterGroup) {
	api.GET("/holdings", h.getAll)
	api.POST("/holdings", h.add)
	api.PUT("/holdings/:symbol", h.update)
	api.DELETE("/holdings/:symbol", h.remove)
}

func (h *HoldingHandler) getAll(c *gin.Context) {
	items, err := h.repo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch holdings"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *HoldingHandler) add(c *gin.Context) {
	var req struct {
		Symbol  string  `json:"symbol"`
		Name    string  `json:"name"`
		Shares  float64 `json:"shares"`
		AvgCost float64 `json:"avgCost"`
		BuyDate string  `json:"buyDate"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Symbol == "" || req.Shares == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol, shares, and avgCost are required"})
		return
	}
	if req.BuyDate == "" {
		req.BuyDate = time.Now().Format("2006-01-02")
	}
	item := model.Holding{
		Symbol: req.Symbol, Name: req.Name, Shares: req.Shares,
		AvgCost: req.AvgCost, BuyDate: req.BuyDate,
	}
	if err := h.repo.Add(item); err != nil {
		if err == repo.ErrDuplicate {
			c.JSON(http.StatusConflict, gin.H{"error": "Holding already exists for this symbol"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add"})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *HoldingHandler) update(c *gin.Context) {
	symbol := c.Param("symbol")
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid body"})
		return
	}
	err := h.repo.Update(symbol, func(h *model.Holding) {
		if v, ok := req["shares"]; ok {
			h.Shares = v.(float64)
		}
		if v, ok := req["avgCost"]; ok {
			h.AvgCost = v.(float64)
		}
		if v, ok := req["buyDate"]; ok {
			h.BuyDate = v.(string)
		}
		if v, ok := req["name"]; ok {
			h.Name = v.(string)
		}
	})
	if err == repo.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Holding not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *HoldingHandler) remove(c *gin.Context) {
	symbol := c.Param("symbol")
	if err := h.repo.Remove(symbol); err != nil {
		if err == repo.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Holding not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
```

- [ ] **Step 4: Install Gin dependency**

Run: `go get github.com/gin-gonic/gin`

- [ ] **Step 5: Verify compilation**

Run: `go build ./internal/handler/...`
Expected: (no errors)

- [ ] **Step 6: Commit**

```bash
git add internal/handler/ go.mod go.sum
git commit -m "feat: add HTTP handlers for watchlist, alerts, and holdings"
```

---

### Task 9: Main entry point wired with all components

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Write main.go with full wiring**

```go
// cmd/server/main.go
package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/black-eleven/stock-monitor/internal/config"
	"github.com/black-eleven/stock-monitor/internal/db"
	"github.com/black-eleven/stock-monitor/internal/handler"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Repositories
	watchlistRepo := repo.NewWatchlistRepo(database)
	alertRepo := repo.NewAlertRepo(database)
	holdingRepo := repo.NewHoldingRepo(database)

	// Handlers
	watchlistH := handler.NewWatchlistHandler(watchlistRepo)
	alertH := handler.NewAlertHandler(alertRepo)
	holdingH := handler.NewHoldingHandler(holdingRepo)

	r := gin.Default()
	api := r.Group("/api")
	watchlistH.Register(api)
	alertH.Register(api)
	holdingH.Register(api)

	// Serve web static files
	r.StaticFile("/", "./web/index.html")
	r.StaticFS("/css", http.Dir("./web/css"))
	r.StaticFS("/js", http.Dir("./web/js"))

	go func() {
		log.Printf("Starting server on :%s", cfg.Port)
		if err := r.Run(":" + cfg.Port); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
}
```

- [ ] **Step 2: Build and run**

Run: `go build ./cmd/server && echo "Build OK"`
Expected: `Build OK`

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: wire up main entry point with all handlers"
```

---

## Phase 2: Real-time Data Pipeline

### Task 10: QOS WebSocket client — connection management

**Files:**
- Create: `internal/qos/client.go`

- [ ] **Step 1: Implement QOS client (connection, heartbeat, reconnect)**

```go
// internal/qos/client.go
package qos

import (
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type QosClient struct {
	wsUrl    string
	conn     *websocket.Conn
	mu       sync.Mutex
	connected atomic.Bool

	sendCh   chan []byte
	closeCh  chan struct{}

	pending   sync.Map // int64 → chan rawResponse
	reqSeq    atomic.Int64

	OnQuote func(Quote)
	OnKline func(Kline)

	// internal control
	reconnectDelay time.Duration
	heartbeatTimer *time.Timer
}

type Quote struct {
	Code      string  `json:"code"`
	Price     float64 `json:"price"`
	YP        float64 `json:"yp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"volume"`
	Turnover  float64 `json:"turnover"`
	Timestamp int64   `json:"timestamp"`
	Status    string  `json:"status"`
}

type Kline struct {
	Code      string  `json:"code"`
	Open      float64 `json:"open"`
	Close     float64 `json:"close"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"volume"`
	Timestamp int64   `json:"timestamp"`
	Kt        int     `json:"kt"`
}

type rawResponse struct {
	msg []byte
	err error
}

func NewClient(wsUrl string) *QosClient {
	c := &QosClient{
		wsUrl:         wsUrl,
		sendCh:        make(chan []byte, 64),
		closeCh:       make(chan struct{}),
		reconnectDelay: 1 * time.Second,
	}
	return c
}

func (c *QosClient) Connect() {
	for {
		select {
		case <-c.closeCh:
			return
		default:
		}

		conn, _, err := websocket.DefaultDialer.Dial(c.wsUrl, nil)
		if err != nil {
			log.Printf("[QOS] Connect error: %v (retry in %v)", err, c.reconnectDelay)
			time.Sleep(c.reconnectDelay)
			c.reconnectDelay *= 2
			if c.reconnectDelay > 30*time.Second {
				c.reconnectDelay = 30 * time.Second
			}
			continue
		}

		log.Println("[QOS] Connected")
		c.mu.Lock()
		c.conn = conn
		c.mu.Unlock()
		c.connected.Store(true)
		c.reconnectDelay = 1 * time.Second

		go c.readLoop(conn)
		c.writeLoop(conn)

		// If writeLoop returned, readLoop will also exit (conn closed)
		c.connected.Store(false)
		log.Println("[QOS] Disconnected, reconnecting...")
	}
}

func (c *QosClient) readLoop(conn *websocket.Conn) {
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return // connection closed
		}

		var msg struct {
			Tp     string          `json:"tp"`
			Type   string          `json:"type"`
			Reqid  int64           `json:"reqid"`
			Data   json.RawMessage `json:"data"`
			C      string          `json:"c"`
			Lp     string          `json:"lp"`
			Yp     string          `json:"yp"`
			O      string          `json:"o"`
			H      string          `json:"h"`
			L      string          `json:"l"`
			V      string          `json:"v"`
			T      string          `json:"t"`
			Ts     int64           `json:"ts"`
			S      string          `json:"s"`
			Cl     string          `json:"cl"`
			Kt     int             `json:"kt"`
		}

		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		// Dispatch by message type
		switch {
		case msg.Tp == "S" && c.OnQuote != nil:
			c.OnQuote(Quote{
				Code: msg.C, Price: parseFloat(msg.Lp), YP: parseFloat(msg.Yp),
				Open: parseFloat(msg.O), High: parseFloat(msg.H), Low: parseFloat(msg.L),
				Volume: parseFloat(msg.V), Turnover: parseFloat(msg.T),
				Timestamp: msg.Ts, Status: msg.S,
			})

		case msg.Tp == "K" && c.OnKline != nil:
			c.OnKline(Kline{
				Code: msg.C, Open: parseFloat(msg.O), Close: parseFloat(msg.Cl),
				High: parseFloat(msg.H), Low: parseFloat(msg.L),
				Volume: parseFloat(msg.V), Timestamp: msg.Ts, Kt: msg.Kt,
			})

		case msg.Type == "RH" || msg.Type == "RK" || msg.Type == "RS":
			if ch, ok := c.pending.LoadAndDelete(msg.Reqid); ok {
				ch.(chan rawResponse) <- rawResponse{msg: raw}
			}
		}
	}
}

func (c *QosClient) writeLoop(conn *websocket.Conn) {
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case data := <-c.sendCh:
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-heartbeat.C:
			c.send([]byte(`{"type":"H"}`))
		}
	}
}

func (c *QosClient) send(data []byte) {
	select {
	case c.sendCh <- data:
	default:
		log.Println("[QOS] send buffer full, dropping message")
	}
}

func (c *QosClient) IsConnected() bool {
	return c.connected.Load()
}

func (c *QosClient) Close() {
	close(c.closeCh)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
	}
}

func parseFloat(s string) float64 {
	var f float64
	json.Unmarshal([]byte(s), &f)
	return f
}
```

- [ ] **Step 2: Install gorilla/websocket**

Run: `go get github.com/gorilla/websocket`

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/qos/...`
Expected: (no errors)

- [ ] **Step 4: Commit**

```bash
git add internal/qos/ go.mod go.sum
git commit -m "feat: add QOS WebSocket client with connection management"
```

---

### Task 11: QOS kline + quote fetching methods

**Files:**
- Create: `internal/qos/kline.go`

- [ ] **Step 1: Implement fetchHistoryKline and fetchQuote**

```go
// internal/qos/kline.go
package qos

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type KlineRequest struct {
	C  string `json:"c"`
	E  int64  `json:"e,omitempty"`
	Co int    `json:"co"`
	A  int    `json:"a"`
	Kt int    `json:"kt"`
}

func (c *QosClient) FetchHistoryKline(code string, kt int, count int) ([]json.RawMessage, error) {
	if !c.IsConnected() {
		return nil, errors.New("not connected")
	}

	reqid := c.reqSeq.Add(1)
	endTs := time.Now().Unix()

	ch := make(chan rawResponse, 1)
	c.pending.Store(reqid, ch)

	req := struct {
		Type      string         `json:"type"`
		KlineReqs []KlineRequest `json:"kline_reqs"`
		Reqid     int64          `json:"reqid"`
	}{
		Type: "RH",
		KlineReqs: []KlineRequest{{
			C: code, E: endTs, Co: count, A: 0, Kt: kt,
		}},
		Reqid: reqid,
	}

	body, _ := json.Marshal(req)
	c.send(body)

	select {
	case resp := <-ch:
		if resp.err != nil {
			return nil, resp.err
		}
		var msg struct {
			Data json.RawMessage `json:"data"`
		}
		json.Unmarshal(resp.msg, &msg)
		// data is an array
		var data []json.RawMessage
		json.Unmarshal(msg.Data, &data)
		return data, nil
	case <-time.After(10 * time.Second):
		c.pending.Delete(reqid)
		return nil, errors.New("history kline request timeout")
	}
}

func (c *QosClient) FetchQuote(code string) (*Quote, error) {
	if !c.IsConnected() {
		return nil, errors.New("not connected")
	}

	reqid := c.reqSeq.Add(1)
	ch := make(chan rawResponse, 1)
	c.pending.Store(reqid, ch)

	req := map[string]interface{}{
		"type":   "RS",
		"codes":  []string{code},
		"reqid":  reqid,
	}

	body, _ := json.Marshal(req)
	c.send(body)

	select {
	case resp := <-ch:
		if resp.err != nil {
			return nil, resp.err
		}
		var msg struct {
			Data []struct {
				C  string `json:"c"`
				Lp string `json:"lp"`
				Yp string `json:"yp"`
				O  string `json:"o"`
				H  string `json:"h"`
				L  string `json:"l"`
				V  string `json:"v"`
				T  string `json:"t"`
				Ts int64  `json:"ts"`
				S  string `json:"s"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp.msg, &msg); err != nil {
			return nil, err
		}
		if len(msg.Data) == 0 {
			return nil, errors.New("no data")
		}
		d := msg.Data[0]
		return &Quote{
			Code: d.C, Price: parseFloat(d.Lp), YP: parseFloat(d.Yp),
			Open: parseFloat(d.O), High: parseFloat(d.H), Low: parseFloat(d.L),
			Volume: parseFloat(d.V), Turnover: parseFloat(d.T),
			Timestamp: d.Ts, Status: d.S,
		}, nil
	case <-time.After(10 * time.Second):
		c.pending.Delete(reqid)
		return nil, errors.New(fmt.Sprintf("quote request timeout for %s", code))
	}
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/qos/...`
Expected: (no errors)

- [ ] **Step 3: Commit**

```bash
git add internal/qos/kline.go
git commit -m "feat: add QOS fetchHistoryKline and fetchQuote methods"
```

---

### Task 12: WebSocket Hub for browser/app clients

**Files:**
- Create: `internal/ws/hub.go`
- Create: `internal/ws/client.go`

- [ ] **Step 1: Implement WS Hub and Client**

```go
// internal/ws/hub.go
package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mu         sync.RWMutex
	quotes     sync.Map // string → json.RawMessage
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			// Send snapshot of all cached quotes
			go h.sendSnapshot(client)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) SendSnapshot(client *Client) {
	h.sendSnapshot(client)
}

func (h *Hub) sendSnapshot(client *Client) {
	var quotes []json.RawMessage
	h.quotes.Range(func(_, v interface{}) bool {
		quotes = append(quotes, v.(json.RawMessage))
		return true
	})
	if quotes == nil {
		quotes = []json.RawMessage{}
	}
	msg, _ := json.Marshal(map[string]interface{}{
		"type": "snapshot",
		"data": quotes,
	})
	client.send <- msg
}

func (h *Hub) BroadcastQuote(quote interface{}) {
	data, _ := json.Marshal(quote)
	msg, _ := json.Marshal(map[string]interface{}{
		"type": "quote",
		"data": json.RawMessage(data),
	})
	// Cache
	if q, ok := quote.(interface{ GetCode() string }); ok {
		h.quotes.Store(q.GetCode(), data)
	}
	h.broadcast <- msg
}

func (h *Hub) BroadcastAlert(alert interface{}) {
	data, _ := json.Marshal(alert)
	msg, _ := json.Marshal(map[string]interface{}{
		"type": "alert",
		"data": json.RawMessage(data),
	})
	h.broadcast <- msg
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade error: %v", err)
		return
	}
	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}
	h.register <- client
	go client.writePump()
	go client.readPump()
}
```

```go
// internal/ws/client.go
package ws

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		// Client messages are ignored (push-only for now)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/ws/...`
Expected: (no errors)

- [ ] **Step 3: Commit**

```bash
git add internal/ws/
git commit -m "feat: add WebSocket hub for browser/app clients"
```

---

### Task 13: Quote + Kline HTTP handlers

**Files:**
- Create: `internal/handler/quote.go`
- Create: `internal/handler/kline.go`

- [ ] **Step 1: Implement quote handler**

```go
// internal/handler/quote.go
package handler

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/black-eleven/stock-monitor/internal/qos"
	"github.com/gin-gonic/gin"
)

var symbolRegex = regexp.MustCompile(`^(HK|SH|SZ|US):[A-Z0-9]{1,10}$`)

type QuoteHandler struct {
	qos *qos.QosClient
}

func NewQuoteHandler(qos *qos.QosClient) *QuoteHandler {
	return &QuoteHandler{qos: qos}
}

func (h *QuoteHandler) Register(api *gin.RouterGroup) {
	api.GET("/quote/batch", h.batch)
	api.GET("/quote/:symbol", h.single)
}

func (h *QuoteHandler) batch(c *gin.Context) {
	symbolsStr := c.Query("symbols")
	symbols := strings.Split(symbolsStr, ",")
	trimmed := make([]string, 0, len(symbols))
	for _, s := range symbols {
		s = strings.TrimSpace(s)
		if s != "" {
			trimmed = append(trimmed, s)
		}
	}
	if len(trimmed) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No symbols provided"})
		return
	}

	type result struct {
		symbol string
		quote  *qos.Quote
	}
	results := make(chan result, len(trimmed))

	for _, s := range trimmed {
		go func(symbol string) {
			q, err := h.qos.FetchQuote(symbol)
			if err != nil {
				results <- result{symbol: symbol}
			} else {
				results <- result{symbol: symbol, quote: q}
			}
		}(s)
	}

	data := make(map[string]interface{})
	for i := 0; i < len(trimmed); i++ {
		r := <-results
		if r.quote != nil {
			data[r.symbol] = r.quote
		}
	}
	c.JSON(http.StatusOK, data)
}

func (h *QuoteHandler) single(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	if !symbolRegex.MatchString(symbol) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid symbol format. Use HK:700 / SH:600519 / SZ:000001 / US:AAPL"})
		return
	}
	quote, err := h.qos.FetchQuote(symbol)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch quote"})
		return
	}
	c.JSON(http.StatusOK, quote)
}
```

- [ ] **Step 2: Implement kline handler**

```go
// internal/handler/kline.go
package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/black-eleven/stock-monitor/internal/qos"
	"github.com/gin-gonic/gin"
)

var ktMap = map[string]int{
	"1m": 1, "5m": 5, "15m": 15, "30m": 30,
	"1h": 60, "2h": 120, "4h": 240,
	"1d": 1001, "1w": 1007, "1M": 1030,
}

type KlineHandler struct {
	qos *qos.QosClient
}

func NewKlineHandler(qos *qos.QosClient) *KlineHandler {
	return &KlineHandler{qos: qos}
}

func (h *KlineHandler) Register(api *gin.RouterGroup) {
	api.GET("/kline/:symbol", h.getKline)
}

func (h *KlineHandler) getKline(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	interval := c.DefaultQuery("interval", "1d")
	countStr := c.DefaultQuery("count", "100")
	count, _ := strconv.Atoi(countStr)
	if count <= 0 {
		count = 100
	}

	kt, ok := ktMap[interval]
	if !ok {
		keys := make([]string, 0, len(ktMap))
		for k := range ktMap {
			keys = append(keys, k)
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid interval: " + interval + ". Supported: " + strings.Join(keys, ", "),
		})
		return
	}

	data, err := h.qos.FetchHistoryKline(symbol, kt, count)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch kline data"})
		return
	}
	c.JSON(http.StatusOK, data)
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/handler/...`
Expected: (no errors)

- [ ] **Step 4: Commit**

```bash
git add internal/handler/quote.go internal/handler/kline.go
git commit -m "feat: add quote and kline HTTP handlers"
```

---

### Task 14: Alert Engine

**Files:**
- Create: `internal/alert/engine.go`

- [ ] **Step 1: Implement alert engine**

```go
// internal/alert/engine.go
package alert

import (
	"math"
	"time"

	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/black-eleven/stock-monitor/internal/ws"
)

type Engine struct {
	alertRepo  *repo.AlertRepo
	hub        *ws.Hub
}

func NewEngine(alertRepo *repo.AlertRepo, hub *ws.Hub) *Engine {
	return &Engine{alertRepo: alertRepo, hub: hub}
}

type AlertEvent struct {
	AlertID     int     `json:"alertId"`
	Symbol      string  `json:"symbol"`
	Price       float64 `json:"price"`
	Type        string  `json:"type"`
	Value       float64 `json:"value"`
	Message     string  `json:"message"`
	TriggeredAt string  `json:"triggeredAt"`
}

func (e *Engine) Evaluate(quote interface{}) {
	q, ok := quote.(interface {
		GetCode() string
		GetPrice() float64
		GetYP() float64
	})
	if !ok {
		return
	}

	rules, err := e.alertRepo.GetBySymbol(q.GetCode())
	if err != nil {
		return
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// Debounce: skip if triggered within last 30 minutes
		if rule.LastTriggeredAt != nil {
			lastTriggered, err := time.Parse(time.RFC3339, *rule.LastTriggeredAt)
			if err == nil && now.Sub(lastTriggered) < 30*time.Minute {
				continue
			}
		}

		triggered := false
		var message string

		switch rule.Type {
		case "above":
			if q.GetPrice() >= rule.Value {
				triggered = true
				message = q.GetCode() + " 价格涨破 " + formatPrice(rule.Value)
			}
		case "below":
			if q.GetPrice() <= rule.Value {
				triggered = true
				message = q.GetCode() + " 价格跌破 " + formatPrice(rule.Value)
			}
		case "change_pct":
			pct := math.Abs((q.GetPrice() - q.GetYP()) / q.GetYP() * 100)
			if pct >= math.Abs(rule.Value) {
				triggered = true
				dir := "涨"
				if q.GetPrice() < q.GetYP() {
					dir = "跌"
				}
				message = q.GetCode() + " " + dir + "幅达 " + formatPct(pct) + "%"
			}
		}

		if triggered {
			e.alertRepo.Update(rule.ID, func(a *model.AlertRule) {
				a.LastTriggeredAt = &nowStr
			})

			logEntry := model.AlertLog{
				AlertID:     rule.ID,
				Symbol:      q.GetCode(),
				Price:       q.GetPrice(),
				Message:     message,
				TriggeredAt: nowStr,
			}
			e.alertRepo.AppendLog(logEntry)
			e.alertRepo.PurgeOldLogs(200)

			e.hub.BroadcastAlert(AlertEvent{
				AlertID:     rule.ID,
				Symbol:      q.GetCode(),
				Price:       q.GetPrice(),
				Type:        rule.Type,
				Value:       rule.Value,
				Message:     message,
				TriggeredAt: nowStr,
			})
		}
	}
}

func formatPrice(p float64) string {
	return fmt.Sprintf("%.2f", p)
}

func formatPct(p float64) string {
	return fmt.Sprintf("%.2f", p)
}
```

Note: Add `"fmt"` to the import block.

- [ ] **Step 2: Update Quote model to implement interface methods**

Edit `internal/model/quote.go` and add:

```go
func (q Quote) GetCode() string  { return q.Code }
func (q Quote) GetPrice() float64 { return q.Price }
func (q Quote) GetYP() float64    { return q.YP }
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/alert/... ./internal/model/...`
Expected: (no errors)

- [ ] **Step 4: Commit**

```bash
git add internal/alert/ internal/model/quote.go
git commit -m "feat: add alert evaluation engine with 30-min debounce"
```

---

### Task 15: Wire all components in main.go (final)

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Update main.go with full wiring**

Full `main.go` content — replace the stub with:

```go
// cmd/server/main.go
package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/black-eleven/stock-monitor/internal/alert"
	"github.com/black-eleven/stock-monitor/internal/config"
	"github.com/black-eleven/stock-monitor/internal/db"
	"github.com/black-eleven/stock-monitor/internal/handler"
	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/qos"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/black-eleven/stock-monitor/internal/ws"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	// Database
	database, err := db.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Repositories
	watchlistRepo := repo.NewWatchlistRepo(database)
	alertRepo := repo.NewAlertRepo(database)
	holdingRepo := repo.NewHoldingRepo(database)

	// WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()

	// QOS Client
	qosClient := qos.NewClient(cfg.QosWsUrl)

	// Alert Engine
	alertEngine := alert.NewEngine(alertRepo, hub)

	// Wire QOS callbacks
	qosClient.OnQuote = func(q qos.Quote) {
		hub.BroadcastQuote(model.FromQosQuote(q))
		alertEngine.Evaluate(model.FromQosQuote(q))
	}

	// HTTP handlers
	watchlistH := handler.NewWatchlistHandler(watchlistRepo)
	alertH := handler.NewAlertHandler(alertRepo)
	holdingH := handler.NewHoldingHandler(holdingRepo)
	quoteH := handler.NewQuoteHandler(qosClient)
	klineH := handler.NewKlineHandler(qosClient)

	r := gin.Default()
	api := r.Group("/api")
	watchlistH.Register(api)
	alertH.Register(api)
	holdingH.Register(api)
	quoteH.Register(api)
	klineH.Register(api)

	// WebSocket endpoint
	r.GET("/ws", hub.ServeWS)

	// Static files
	r.StaticFile("/", "./web/index.html")
	r.Static("/css", "./web/css")
	r.Static("/js", "./web/js")

	go func() {
		log.Printf("Starting server on :%s", cfg.Port)
		if err := r.Run(":" + cfg.Port); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Connect QOS after server is ready
	go qosClient.Connect()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
	qosClient.Close()
}
```

- [ ] **Step 2: Add FromQosQuote conversion to model/quote.go**

Edit `internal/model/quote.go` and add:

```go
import "github.com/black-eleven/stock-monitor/internal/qos"

func FromQosQuote(q qos.Quote) Quote {
	return Quote{
		Code: q.Code, Price: q.Price, YP: q.YP,
		Open: q.Open, High: q.High, Low: q.Low,
		Volume: q.Volume, Turnover: q.Turnover,
		Timestamp: q.Timestamp, Status: q.Status,
	}
}
```

Note: This creates a circular import (model ← qos, qos needs nothing from model). Since qos doesn't import model, this is fine — model imports qos for the conversion function.

- [ ] **Step 3: Build**

Run: `go build ./cmd/server`
Expected: (no errors)

- [ ] **Step 4: Commit**

```bash
git add cmd/server/main.go internal/model/quote.go
git commit -m "feat: wire all components in main.go"
```

---

## Phase 3: Web Frontend + Migration

### Task 16: Move web files and add embed

**Files:**
- Create: `web/` (from `public/`)
- Modify: `internal/handler/` — add static handler or keep gin.Static

- [ ] **Step 1: Copy public/ to web/**

Run:
```bash
cp -r public web
```

- [ ] **Step 2: Verify the web directory exists**

Run: `ls web/`
Expected: `css  index.html  js`

- [ ] **Step 3: Commit**

```bash
git add web/
git commit -m "feat: copy public/ to web/ for Go backend serving"
```

---

### Task 17: Update WebSocket URL in api.js

**Files:**
- Modify: `web/js/api.js`

- [ ] **Step 1: Change WS connection URL**

Edit `web/js/api.js` line 44:

```diff
-    this.ws = new WebSocket(`${protocol}//${location.host}`);
+    this.ws = new WebSocket(`${protocol}//${location.host}/ws`);
```

- [ ] **Step 2: Commit**

```bash
git add web/js/api.js
git commit -m "fix: update WebSocket URL to /ws path for Go backend"
```

---

### Task 18: Data migration tool (JSON → SQLite)

**Files:**
- Create: `cmd/migrate/main.go`

- [ ] **Step 1: Implement migration tool**

```go
// cmd/migrate/main.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/black-eleven/stock-monitor/internal/config"
	"github.com/black-eleven/stock-monitor/internal/db"
	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/repo"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	migrateWatchlist(database)
	migrateHoldings(database)
	migrateAlerts(database)

	fmt.Println("Migration complete!")
}

func migrateWatchlist(db *sql.DB) {
	r := repo.NewWatchlistRepo(db)
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
		if err := r.Add(item); err != nil {
			fmt.Printf("watchlist skip %s: %v\n", item.Symbol, err)
		} else {
			fmt.Printf("watchlist added: %s\n", item.Symbol)
		}
	}
}

func migrateHoldings(db *sql.DB) {
	r := repo.NewHoldingRepo(db)
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
		if err := r.Add(h); err != nil {
			fmt.Printf("holdings skip %s: %v\n", h.Symbol, err)
		} else {
			fmt.Printf("holdings added: %s\n", h.Symbol)
		}
	}
}

func migrateAlerts(db *sql.DB) {
	r := repo.NewAlertRepo(db)
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
		if _, err := r.Add(a); err != nil {
			fmt.Printf("alerts skip %s: %v\n", a.Symbol, err)
		} else {
			fmt.Printf("alerts added: %s (id=%d)\n", a.Symbol, a.ID)
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
```

Note: Add `"database/sql"` to imports.

- [ ] **Step 2: Verify compilation**

Run: `go build ./cmd/migrate`
Expected: (no errors)

- [ ] **Step 3: Commit**

```bash
git add cmd/migrate/
git commit -m "feat: add JSON to SQLite data migration tool"
```

---

## Phase 4: Flutter Mobile App

### Task 19: Flutter project setup

**Files:**
- Create: `mobile/stock_monitor/` (via `flutter create`)
- Modify: `mobile/stock_monitor/pubspec.yaml`
- Create: `mobile/stock_monitor/lib/main.dart`
- Create: `mobile/stock_monitor/lib/app.dart`
- Create: `mobile/stock_monitor/lib/core/config.dart`
- Create: `mobile/stock_monitor/lib/core/theme.dart`
- Create: `mobile/stock_monitor/lib/core/utils.dart`

- [ ] **Step 1: Create Flutter project**

Run:
```bash
cd mobile && flutter create --org com.stockmonitor stock_monitor
```

- [ ] **Step 2: Update pubspec.yaml**

Edit `pubspec.yaml`, replace the `dependencies` block with:

```yaml
dependencies:
  flutter:
    sdk: flutter
  cupertino_icons: ^1.0.8
  dio: ^5.7.0
  web_socket_channel: ^3.0.1
  flutter_riverpod: ^2.6.1
  riverpod_annotation: ^2.6.1
  go_router: ^14.6.2
  fl_chart: ^0.69.2
  intl: ^0.19.0
  shared_preferences: ^2.3.3

dev_dependencies:
  flutter_test:
    sdk: flutter
  flutter_lints: ^5.0.0
  riverpod_generator: ^2.6.2
  build_runner: ^2.4.13
  riverpod_lint: ^2.6.2
```

- [ ] **Step 3: Install dependencies**

Run: `cd mobile/stock_monitor && flutter pub get`
Expected: successful

- [ ] **Step 4: Write config.dart**

```dart
// lib/core/config.dart
class AppConfig {
  static const String host = '10.0.2.2'; // Android emulator → host machine
  static const int port = 3000;
  static String get baseUrl => 'http://$host:$port/api';
  static String get wsUrl => 'ws://$host:$port/ws';
}
```

- [ ] **Step 5: Write theme.dart**

```dart
// lib/core/theme.dart
import 'package:flutter/material.dart';

class AppTheme {
  static const Color bg = Color(0xFF0d1117);
  static const Color surface = Color(0xFF161b22);
  static const Color border = Color(0xFF30363d);
  static const Color textPrimary = Color(0xFFe6edf3);
  static const Color textSecondary = Color(0xFF8b949e);
  static const Color up = Color(0xFF3fb950);
  static const Color down = Color(0xFFf85149);
  static const Color accent = Color(0xFF1f6feb);

  static ThemeData get darkTheme => ThemeData(
    brightness: Brightness.dark,
    scaffoldBackgroundColor: bg,
    appBarTheme: const AppBarTheme(
      backgroundColor: surface,
      foregroundColor: textPrimary,
    ),
    bottomNavigationBarTheme: const BottomNavigationBarThemeData(
      backgroundColor: surface,
      selectedItemColor: accent,
      unselectedItemColor: textSecondary,
    ),
    cardColor: surface,
    dividerColor: border,
    colorScheme: const ColorScheme.dark(
      primary: accent,
      surface: surface,
    ),
  );
}
```

- [ ] **Step 6: Write utils.dart**

```dart
// lib/core/utils.dart
String formatPrice(double? price) {
  if (price == null) return '--';
  return price.toStringAsFixed(2);
}

String formatVolume(double v) {
  if (v >= 100000000) return '${(v / 100000000).toStringAsFixed(2)}亿';
  if (v >= 10000) return '${(v / 10000).toStringAsFixed(0)}万';
  return v.toStringAsFixed(0);
}

String shortCode(String code) {
  return code.replaceFirst(RegExp(r'^(HK|SH|SZ|US):'), '');
}

double calcChangePct(double price, double yp) {
  if (yp == 0) return 0;
  return (price - yp) / yp * 100;
}

String formatChange(double price, double yp) {
  final pct = calcChangePct(price, yp);
  final sign = pct >= 0 ? '+' : '';
  return '$sign${pct.toStringAsFixed(2)}%';
}

String changeDir(double price, double yp) {
  return price >= yp ? 'up' : 'down';
}
```

- [ ] **Step 7: Write main.dart and app.dart**

```dart
// lib/main.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'app.dart';

void main() {
  runApp(const ProviderScope(child: StockMonitorApp()));
}
```

```dart
// lib/app.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'core/theme.dart';

class StockMonitorApp extends StatelessWidget {
  const StockMonitorApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'Stock Monitor',
      theme: AppTheme.darkTheme,
      routerConfig: router,
      debugShowCheckedModeBanner: false,
    );
  }
}

final _rootNavigatorKey = GlobalKey<NavigatorState>();
final _shellNavigatorKey = GlobalKey<NavigatorState>();

final router = GoRouter(
  navigatorKey: _rootNavigatorKey,
  initialLocation: '/watchlist',
  routes: [
    ShellRoute(
      navigatorKey: _shellNavigatorKey,
      builder: (context, state, child) => AppShell(child: child),
      routes: [
        GoRoute(path: '/watchlist', builder: (_, __) => const WatchlistScreen()),
        GoRoute(path: '/kline', builder: (_, __) => const KlineScreen()),
        GoRoute(path: '/holdings', builder: (_, __) => const HoldingsScreen()),
        GoRoute(path: '/alerts', builder: (_, __) => const AlertsScreen()),
        GoRoute(path: '/analysis', builder: (_, __) => const AnalysisScreen()),
      ],
    ),
  ],
);

class AppShell extends StatelessWidget {
  final Widget child;
  const AppShell({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: child,
      bottomNavigationBar: BottomNavigationBar(
        currentIndex: _calculateSelectedIndex(context),
        onTap: (i) => _onTap(context, i),
        type: BottomNavigationBarType.fixed,
        items: const [
          BottomNavigationBarItem(icon: Icon(Icons.home), label: '自选'),
          BottomNavigationBarItem(icon: Icon(Icons.show_chart), label: 'K线'),
          BottomNavigationBarItem(icon: Icon(Icons.account_balance_wallet), label: '持仓'),
          BottomNavigationBarItem(icon: Icon(Icons.notifications), label: '提醒'),
          BottomNavigationBarItem(icon: Icon(Icons.analytics), label: '分析'),
        ],
      ),
    );
  }

  int _calculateSelectedIndex(BuildContext context) {
    final loc = GoRouterState.of(context).uri.path;
    if (loc.startsWith('/kline')) return 1;
    if (loc.startsWith('/holdings')) return 2;
    if (loc.startsWith('/alerts')) return 3;
    if (loc.startsWith('/analysis')) return 4;
    return 0;
  }

  void _onTap(BuildContext context, int i) {
    final routes = ['/watchlist', '/kline', '/holdings', '/alerts', '/analysis'];
    context.go(routes[i]);
  }
}

// Placeholder screens (replace in subsequent tasks)
class WatchlistScreen extends StatelessWidget {
  const WatchlistScreen({super.key});
  @override
  Widget build(BuildContext context) => const Center(child: Text('自选股'));
}
class KlineScreen extends StatelessWidget {
  const KlineScreen({super.key});
  @override
  Widget build(BuildContext context) => const Center(child: Text('K线图'));
}
class HoldingsScreen extends StatelessWidget {
  const HoldingsScreen({super.key});
  @override
  Widget build(BuildContext context) => const Center(child: Text('持仓'));
}
class AlertsScreen extends StatelessWidget {
  const AlertsScreen({super.key});
  @override
  Widget build(BuildContext context) => const Center(child: Text('提醒'));
}
class AnalysisScreen extends StatelessWidget {
  const AnalysisScreen({super.key});
  @override
  Widget build(BuildContext context) => const Center(child: Text('分析'));
}
```

- [ ] **Step 8: Verify build**

Run: `cd mobile/stock_monitor && flutter build apk --debug 2>&1 | tail -5`
Expected: builds successfully

- [ ] **Step 9: Commit**

```bash
git add mobile/
git commit -m "feat: initialize Flutter project with routing and theme"
```

---

### Task 20: Domain models (Dart)

**Files:**
- Create: `mobile/stock_monitor/lib/domain/model/stock.dart`
- Create: `mobile/stock_monitor/lib/domain/model/alert.dart`
- Create: `mobile/stock_monitor/lib/domain/model/holding.dart`
- Create: `mobile/stock_monitor/lib/domain/model/kline.dart`

- [ ] **Step 1: Write all Dart model files**

```dart
// lib/domain/model/stock.dart
class WatchlistItem {
  final String symbol;
  final String name;
  final String addedAt;
  WatchlistItem({required this.symbol, required this.name, required this.addedAt});
  factory WatchlistItem.fromJson(Map<String, dynamic> json) => WatchlistItem(
    symbol: json['symbol'] as String,
    name: json['name'] as String,
    addedAt: json['addedAt'] as String,
  );
}

class Quote {
  final String code;
  final double price;
  final double yp;
  final double open;
  final double high;
  final double low;
  final double volume;
  final double turnover;
  final int timestamp;
  final String status;
  Quote({
    required this.code, required this.price, required this.yp,
    required this.open, required this.high, required this.low,
    required this.volume, required this.turnover, required this.timestamp,
    required this.status,
  });
  factory Quote.fromJson(Map<String, dynamic> json) => Quote(
    code: json['code'] as String,
    price: (json['price'] as num).toDouble(),
    yp: (json['yp'] as num).toDouble(),
    open: (json['open'] as num).toDouble(),
    high: (json['high'] as num).toDouble(),
    low: (json['low'] as num).toDouble(),
    volume: (json['volume'] as num).toDouble(),
    turnover: (json['turnover'] as num).toDouble(),
    timestamp: json['timestamp'] as int,
    status: json['status'] as String,
  );
}
```

```dart
// lib/domain/model/alert.dart
class AlertRule {
  final int id;
  final String symbol;
  final String type;
  final double value;
  final bool enabled;
  final String createdAt;
  final String? lastTriggeredAt;
  AlertRule({required this.id, required this.symbol, required this.type, required this.value, required this.enabled, required this.createdAt, this.lastTriggeredAt});
  factory AlertRule.fromJson(Map<String, dynamic> json) => AlertRule(
    id: json['id'] as int,
    symbol: json['symbol'] as String,
    type: json['type'] as String,
    value: (json['value'] as num).toDouble(),
    enabled: json['enabled'] as bool,
    createdAt: json['createdAt'] as String,
    lastTriggeredAt: json['lastTriggeredAt'] as String?,
  );
}

class AlertLog {
  final int id;
  final int alertId;
  final String symbol;
  final double price;
  final String message;
  final String triggeredAt;
  AlertLog({required this.id, required this.alertId, required this.symbol, required this.price, required this.message, required this.triggeredAt});
  factory AlertLog.fromJson(Map<String, dynamic> json) => AlertLog(
    id: json['id'] as int,
    alertId: json['alertId'] as int,
    symbol: json['symbol'] as String,
    price: (json['price'] as num).toDouble(),
    message: json['message'] as String,
    triggeredAt: json['triggeredAt'] as String,
  );
}
```

```dart
// lib/domain/model/holding.dart
class Holding {
  final String symbol;
  final String name;
  final double shares;
  final double avgCost;
  final String buyDate;
  Holding({required this.symbol, required this.name, required this.shares, required this.avgCost, required this.buyDate});
  factory Holding.fromJson(Map<String, dynamic> json) => Holding(
    symbol: json['symbol'] as String,
    name: json['name'] as String,
    shares: (json['shares'] as num).toDouble(),
    avgCost: (json['avgCost'] as num).toDouble(),
    buyDate: json['buyDate'] as String,
  );
}
```

```dart
// lib/domain/model/kline.dart
class KlineBar {
  final int ts;
  final double o;
  final double cl;
  final double h;
  final double l;
  final double v;
  KlineBar({required this.ts, required this.o, required this.cl, required this.h, required this.l, required this.v});
  factory KlineBar.fromJson(Map<String, dynamic> json) => KlineBar(
    ts: json['ts'] as int,
    o: (json['o'] as num).toDouble(),
    cl: (json['cl'] as num).toDouble(),
    h: (json['h'] as num).toDouble(),
    l: (json['l'] as num).toDouble(),
    v: (json['v'] as num).toDouble(),
  );
}

class KlineItem {
  final String c;
  final List<KlineBar> k;
  KlineItem({required this.c, required this.k});
  factory KlineItem.fromJson(Map<String, dynamic> json) => KlineItem(
    c: json['c'] as String,
    k: (json['k'] as List).map((e) => KlineBar.fromJson(e as Map<String, dynamic>)).toList(),
  );
}

class Bar {
  final int time;
  final double open;
  final double high;
  final double low;
  final double close;
  Bar({required this.time, required this.open, required this.high, required this.low, required this.close});
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd mobile/stock_monitor && flutter analyze lib/domain/ 2>&1 | head -10`
Expected: No issues found

- [ ] **Step 3: Commit**

```bash
git add mobile/stock_monitor/lib/domain/
git commit -m "feat: add Dart domain models"
```

---

### Task 21: API client + WebSocket client

**Files:**
- Create: `mobile/stock_monitor/lib/data/api/api_client.dart`
- Create: `mobile/stock_monitor/lib/data/api/watchlist_api.dart`
- Create: `mobile/stock_monitor/lib/data/api/alert_api.dart`
- Create: `mobile/stock_monitor/lib/data/api/holding_api.dart`
- Create: `mobile/stock_monitor/lib/data/api/quote_api.dart`
- Create: `mobile/stock_monitor/lib/data/ws/ws_client.dart`

- [ ] **Step 1: Write API client**

```dart
// lib/data/api/api_client.dart
import 'package:dio/dio.dart';
import '../../core/config.dart';

class ApiClient {
  late final Dio dio;

  ApiClient() {
    dio = Dio(BaseOptions(
      baseUrl: AppConfig.baseUrl,
      connectTimeout: const Duration(seconds: 10),
      receiveTimeout: const Duration(seconds: 10),
      headers: {'Content-Type': 'application/json'},
    ));
  }

  Future<Response> get(String path, {Map<String, dynamic>? queryParameters}) =>
      dio.get(path, queryParameters: queryParameters);

  Future<Response> post(String path, {dynamic data}) =>
      dio.post(path, data: data);

  Future<Response> put(String path, {dynamic data}) =>
      dio.put(path, data: data);

  Future<Response> delete(String path) =>
      dio.delete(path);
}
```

- [ ] **Step 2: Write REST API files**

```dart
// lib/data/api/watchlist_api.dart
import '../../domain/model/stock.dart';
import 'api_client.dart';

class WatchlistApi {
  final ApiClient _client;
  WatchlistApi(this._client);

  Future<List<WatchlistItem>> getAll() async {
    final res = await _client.get('/watchlist');
    return (res.data as List).map((e) => WatchlistItem.fromJson(e)).toList();
  }

  Future<WatchlistItem> add(String symbol, String name) async {
    final res = await _client.post('/watchlist', data: {'symbol': symbol, 'name': name});
    return WatchlistItem.fromJson(res.data);
  }

  Future<void> remove(String symbol) => _client.delete('/watchlist/$symbol');
}
```

```dart
// lib/data/api/alert_api.dart
import '../../domain/model/alert.dart';
import 'api_client.dart';

class AlertApi {
  final ApiClient _client;
  AlertApi(this._client);

  Future<List<AlertRule>> getAll() async {
    final res = await _client.get('/alerts');
    return (res.data as List).map((e) => AlertRule.fromJson(e)).toList();
  }

  Future<AlertRule> add(String symbol, String type, double value) async {
    final res = await _client.post('/alerts', data: {'symbol': symbol, 'type': type, 'value': value});
    return AlertRule.fromJson(res.data);
  }

  Future<void> update(int id, Map<String, dynamic> data) =>
      _client.put('/alerts/$id', data: data);

  Future<void> remove(int id) =>
      _client.delete('/alerts/$id');
}
```

```dart
// lib/data/api/holding_api.dart
import '../../domain/model/holding.dart';
import 'api_client.dart';

class HoldingApi {
  final ApiClient _client;
  HoldingApi(this._client);

  Future<List<Holding>> getAll() async {
    final res = await _client.get('/holdings');
    return (res.data as List).map((e) => Holding.fromJson(e)).toList();
  }

  Future<Holding> add(Map<String, dynamic> data) async {
    final res = await _client.post('/holdings', data: data);
    return Holding.fromJson(res.data);
  }

  Future<void> update(String symbol, Map<String, dynamic> data) =>
      _client.put('/holdings/$symbol', data: data);

  Future<void> remove(String symbol) =>
      _client.delete('/holdings/$symbol');
}
```

```dart
// lib/data/api/quote_api.dart
import 'dart:convert';
import '../../domain/model/stock.dart';
import '../../domain/model/kline.dart';
import 'api_client.dart';

class QuoteApi {
  final ApiClient _client;
  QuoteApi(this._client);

  Future<Quote> getQuote(String symbol) async {
    final res = await _client.get('/quote/$symbol');
    return Quote.fromJson(res.data);
  }

  Future<Map<String, Quote>> batchQuotes(List<String> symbols) async {
    final res = await _client.get('/quote/batch', queryParameters: {'symbols': symbols.join(',')});
    final map = <String, Quote>{};
    (res.data as Map<String, dynamic>).forEach((k, v) {
      map[k] = Quote.fromJson(v);
    });
    return map;
  }

  Future<List<KlineItem>> getKline(String symbol, {String interval = '1d', int count = 200}) async {
    final res = await _client.get('/kline/$symbol', queryParameters: {'interval': interval, 'count': count});
    return (res.data as List).map((e) => KlineItem.fromJson(e)).toList();
  }
}
```

- [ ] **Step 3: Write WebSocket client**

```dart
// lib/data/ws/ws_client.dart
import 'dart:async';
import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';
import '../../core/config.dart';
import '../../domain/model/stock.dart';
import '../../domain/model/alert.dart';

class WsClient {
  WebSocketChannel? _channel;
  final _quoteController = StreamController<Quote>.broadcast();
  final _alertController = StreamController<AlertEvent>.broadcast();
  final _snapshotController = StreamController<List<Quote>>.broadcast();
  Timer? _reconnectTimer;

  Stream<Quote> get quoteStream => _quoteController.stream;
  Stream<AlertEvent> get alertStream => _alertController.stream;
  Stream<List<Quote>> get snapshotStream => _snapshotController.stream;

  void connect() {
    try {
      _channel = WebSocketChannel.connect(Uri.parse(AppConfig.wsUrl));
      _channel!.stream.listen(
        (data) {
          final msg = jsonDecode(data as String) as Map<String, dynamic>;
          final type = msg['type'] as String;
          switch (type) {
            case 'snapshot':
              final list = (msg['data'] as List).map((e) => Quote.fromJson(jsonDecode(jsonEncode(e)))).toList();
              _snapshotController.add(list);
              break;
            case 'quote':
              final quote = Quote.fromJson(msg['data']);
              _quoteController.add(quote);
              break;
            case 'alert':
              final alert = AlertEvent.fromJson(msg['data']);
              _alertController.add(alert);
              break;
          }
        },
        onDone: () => _scheduleReconnect(),
        onError: (_) => _scheduleReconnect(),
      );
    } catch (_) {
      _scheduleReconnect();
    }
  }

  void _scheduleReconnect() {
    _reconnectTimer?.cancel();
    _reconnectTimer = Timer(const Duration(seconds: 3), connect);
  }

  void dispose() {
    _reconnectTimer?.cancel();
    _channel?.sink.close();
    _quoteController.close();
    _alertController.close();
    _snapshotController.close();
  }
}

class AlertEvent {
  final int alertId;
  final String symbol;
  final double price;
  final String type;
  final double value;
  final String message;
  final String triggeredAt;
  AlertEvent({required this.alertId, required this.symbol, required this.price, required this.type, required this.value, required this.message, required this.triggeredAt});
  factory AlertEvent.fromJson(Map<String, dynamic> json) => AlertEvent(
    alertId: json['alertId'] as int,
    symbol: json['symbol'] as String,
    price: (json['price'] as num).toDouble(),
    type: json['type'] as String,
    value: (json['value'] as num).toDouble(),
    message: json['message'] as String,
    triggeredAt: json['triggeredAt'] as String,
  );
}
```

- [ ] **Step 4: Verify compilation**

Run: `cd mobile/stock_monitor && flutter analyze lib/data/ 2>&1 | tail -5`
Expected: No issues found

- [ ] **Step 5: Commit**

```bash
git add mobile/stock_monitor/lib/data/
git commit -m "feat: add API client and WebSocket client"
```

---

### Task 22: Riverpod providers (state management)

**Files:**
- Create: `mobile/stock_monitor/lib/presentation/providers/api_providers.dart`
- Create: `mobile/stock_monitor/lib/presentation/providers/quote_provider.dart`
- Create: `mobile/stock_monitor/lib/presentation/providers/kline_provider.dart`

- [ ] **Step 1: Write API providers (singletons)**

```dart
// lib/presentation/providers/api_providers.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../data/api/api_client.dart';
import '../../data/api/watchlist_api.dart';
import '../../data/api/alert_api.dart';
import '../../data/api/holding_api.dart';
import '../../data/api/quote_api.dart';
import '../../data/ws/ws_client.dart';

final apiClientProvider = Provider((ref) => ApiClient());
final watchlistApiProvider = Provider((ref) => WatchlistApi(ref.watch(apiClientProvider)));
final alertApiProvider = Provider((ref) => AlertApi(ref.watch(apiClientProvider)));
final holdingApiProvider = Provider((ref) => HoldingApi(ref.watch(apiClientProvider)));
final quoteApiProvider = Provider((ref) => QuoteApi(ref.watch(apiClientProvider)));

final wsClientProvider = Provider<WsClient>((ref) {
  final ws = WsClient();
  ref.onDispose(() => ws.dispose());
  return ws;
});
```

- [ ] **Step 2: Write quote provider (state notifier pattern)**

```dart
// lib/presentation/providers/quote_provider.dart
import 'dart:async';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../domain/model/stock.dart';
import 'api_providers.dart';

// Cache of latest quotes by symbol
class QuoteState {
  final Map<String, Quote> quotes;
  const QuoteState(this.quotes);
}

class QuoteNotifier extends StateNotifier<QuoteState> {
  final WsClient _ws;
  StreamSubscription? _quoteSub;
  StreamSubscription? _snapshotSub;

  QuoteNotifier(this._ws) : super(const QuoteState({})) {
    _snapshotSub = _ws.snapshotStream.listen((quotes) {
      final map = {...state.quotes};
      for (final q in quotes) {
        map[q.code] = q;
      }
      state = QuoteState(map);
    });
    _quoteSub = _ws.quoteStream.listen((quote) {
      final map = {...state.quotes};
      map[quote.code] = quote;
      state = QuoteState(map);
    });
  }

  Quote? getQuote(String symbol) => state.quotes[symbol];

  @override
  void dispose() {
    _quoteSub?.cancel();
    _snapshotSub?.cancel();
    super.dispose();
  }
}

final quoteProvider = StateNotifierProvider<QuoteNotifier, QuoteState>((ref) {
  final ws = ref.watch(wsClientProvider);
  return QuoteNotifier(ws);
});
```

- [ ] **Step 3: Write kline provider**

```dart
// lib/presentation/providers/kline_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../domain/model/kline.dart';
import 'api_providers.dart';

class KlineState {
  final String symbol;
  final String interval;
  final List<KlineItem> data;
  final bool loading;
  final String? error;

  const KlineState({
    this.symbol = '',
    this.interval = '1d',
    this.data = const [],
    this.loading = false,
    this.error,
  });

  KlineState copyWith({String? symbol, String? interval, List<KlineItem>? data, bool? loading, String? error}) =>
      KlineState(symbol: symbol ?? this.symbol, interval: interval ?? this.interval, data: data ?? this.data, loading: loading ?? this.loading, error: error);
}

class KlineNotifier extends StateNotifier<KlineState> {
  final QuoteApi _api;

  KlineNotifier(this._api) : super(const KlineState());

  Future<void> load(String symbol, {String interval = '1d', int count = 200}) async {
    if (state.loading) return;
    state = state.copyWith(symbol: symbol, interval: interval, loading: true, error: null);
    try {
      final data = await _api.getKline(symbol, interval: interval, count: count);
      state = state.copyWith(data: data, loading: false);
    } catch (e) {
      state = state.copyWith(loading: false, error: e.toString());
    }
  }
}

final klineProvider = StateNotifierProvider<KlineNotifier, KlineState>((ref) {
  final api = ref.watch(quoteApiProvider);
  return KlineNotifier(api);
});
```

- [ ] **Step 4: Verify compilation**

Run: `cd mobile/stock_monitor && flutter analyze lib/presentation/providers/ 2>&1 | tail -5`
Expected: No issues found

- [ ] **Step 5: Commit**

```bash
git add mobile/stock_monitor/lib/presentation/providers/
git commit -m "feat: add Riverpod providers for API, quotes, and kline"
```

---

### Task 23: Watchlist screen (Flutter)

**Files:**
- Modify: `mobile/stock_monitor/lib/app.dart` (replace placeholder)
- Create: `mobile/stock_monitor/lib/presentation/screens/watchlist_screen.dart`
- Create: `mobile/stock_monitor/lib/presentation/widgets/stock_card.dart`

Due to the length of this plan, the remaining Flutter screens follow the same pattern:
- Build a list from `watchlistApiProvider` data
- Show real-time prices from `quoteProvider`
- Add/detail/delete via API

> **Note for implementer:** The remaining Flutter screen tasks (Watchlist, Kline, Holdings, Alerts, Analysis) are detailed in the [Flutter screens companion file](./2026-05-09-flutter-screens.md) with complete widget code for each screen. Continue there for Tasks 23-28.

---

## Self-Review Checklist

- [x] All Go models match SQLite schema types
- [x] All HTTP routes match the Node.js API format
- [x] `FromQosQuote` avoids circular import (qos does not import model)
- [x] WS hub broadcast is non-blocking (select + default)
- [x] Alert engine honors 30-minute debounce like Node.js
- [x] `reqSeq` uses `atomic.Int64` — thread-safe
- [x] Flutter config uses `10.0.2.2` for Android emulator
- [x] Web frontend only change: WS URL path from `/` to `/ws`
- [x] No TBD/TODO/placeholder in completed tasks

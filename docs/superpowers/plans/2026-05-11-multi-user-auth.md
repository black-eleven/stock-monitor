# Multi-User Auth System — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add multi-user account system with invite-code registration, JWT authentication, and per-user data isolation to the stock monitor.

**Architecture:** Go backend: JWT middleware gates all API routes, bcrypt password hashing, SQLite user/invite_codes tables, all existing tables get user_id FK. Web frontend: login/register page with token in localStorage, auth guard in app.js. Flutter: LoginScreen with auth_provider, Dio interceptor injects token. WebSocket auth via `?token=` query param.

**Tech Stack:** Go + Gin + golang-jwt/jwt/v5 + bcrypt(cost=12) + SQLite, Vanilla JS, Flutter/Dart

---

## Phase 1: Go Backend Core

### Task 1: Extend config with JWT secret

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Add JwtSecret and AdminPassword to Config**

Read the file and add to the `Config` struct and `Load()` function:

In `Config` struct, add:
```go
JwtSecret     string
AdminPassword string
```

In `Load()` function add:
```go
jwtSecret := os.Getenv("JWT_SECRET")
if jwtSecret == "" {
	jwtSecret = generateRandomSecret(32)
	log.Printf("[CONFIG] JWT_SECRET not set, generated random secret: %s", jwtSecret)
}
adminPassword := os.Getenv("ADMIN_PASSWORD")
if adminPassword == "" {
	adminPassword = generateRandomSecret(16)
	log.Printf("[CONFIG] ADMIN_PASSWORD not set, generated random password: %s", adminPassword)
}
```

Add helper function at bottom:
```go
func generateRandomSecret(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"[time.Now().UnixNano()%62]
		time.Sleep(1) // cheap entropy mixing
	}
	return string(b)
}
```

Add `"log"` and `"time"` to imports. Return Config with new fields:
```go
JwtSecret:     jwtSecret,
AdminPassword: adminPassword,
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/config/...`
Expected: (no errors)

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add JWT_SECRET and ADMIN_PASSWORD to config"
```

---

### Task 2: Add User and InviteCode domain models

**Files:**
- Create: `internal/model/user.go`

- [ ] **Step 1: Write user.go**

```go
package model

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"-"` // never expose in JSON
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
}

type InviteCode struct {
	ID        int    `json:"id"`
	Code      string `json:"code"`
	MaxUses   int    `json:"maxUses"`
	UsedCount int    `json:"usedCount"`
	CreatedBy int    `json:"createdBy"`
	CreatedAt string `json:"createdAt"`
	IsActive  bool   `json:"isActive"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	InviteCode string `json:"inviteCode" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type CreateInviteCodeRequest struct {
	MaxUses int `json:"maxUses"`
	Count   int `json:"count"` // how many codes to generate, default 1
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/model/...`
Expected: (no errors)

- [ ] **Step 3: Commit**

```bash
git add internal/model/user.go
git commit -m "feat: add User and InviteCode domain models"
```

---

### Task 3: SQLite schema — add users/invite_codes + migration ALTER

**Files:**
- Modify: `internal/db/sqlite.go`

- [ ] **Step 1: Add new tables to schema**

Read the file. In the `const schema` string, add after the `alert_logs` CREATE TABLE:

```sql
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
```

- [ ] **Step 2: Add migration logic for existing tables**

In the `Open` function, after `db.Exec(schema)`, add migration to add `user_id` columns to existing tables:

```go
// Migration: add user_id to existing tables
migrations := []string{
	"ALTER TABLE watchlist ADD COLUMN user_id INTEGER DEFAULT 0",
	"ALTER TABLE holdings ADD COLUMN user_id INTEGER DEFAULT 0",
	"ALTER TABLE alerts ADD COLUMN user_id INTEGER DEFAULT 0",
}
for _, m := range migrations {
	db.Exec(m) // ignore errors (column may already exist)
}
```

- [ ] **Step 3: Add InitAdmin function at bottom of file**

```go
func InitAdmin(db *sql.DB, password string) (*User, error) {
	// Check if admin exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
	if count > 0 {
		return nil, nil // admin already exists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}

	result, err := db.Exec(
		"INSERT INTO users (username, password, role, created_at) VALUES (?, ?, 'admin', ?)",
		"admin", string(hash), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()

	// Create initial invite code
	code := generateInviteCode()
	db.Exec(
		"INSERT INTO invite_codes (code, max_uses, used_count, created_by, created_at, is_active) VALUES (?, 1, 0, ?, ?, 1)",
		code, id, time.Now().UTC().Format(time.RFC3339),
	)
	log.Printf("[DB] Created admin user. Initial invite code: %s", code)

	return &User{ID: int(id), Username: "admin", Role: "admin"}, nil
}
```

Note: Add `"github.com/black-eleven/stock-monitor/internal/model"` and `"golang.org/x/crypto/bcrypt"`, `"log"`, `"time"` to the import block. Also add a `generateInviteCode()` helper:

```go
func generateInviteCode() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 12)
	for i := range b {
		b[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}
```

The `InitAdmin` references `User` from model - need to import it. Actually, let's avoid the circular dependency — return `(int, error)` with the admin user ID instead. Replace the function signature and return:

```go
func InitAdmin(db *sql.DB, password string) (int, error) {
```

And return `int(id), nil`. Return `0, nil` when admin already exists.

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/db/...`
Expected: (no errors)

- [ ] **Step 5: Commit**

```bash
git add internal/db/sqlite.go
git commit -m "feat: add users/invite_codes tables and migration to SQLite schema"
```

---

### Task 4: User repository

**Files:**
- Create: `internal/repo/user.go`
- Create: `internal/repo/user_test.go`

- [ ] **Step 1: Write user_test.go**

```go
package repo

import (
	"testing"

	"github.com/black-eleven/stock-monitor/internal/db"
	"github.com/black-eleven/stock-monitor/internal/model"
)

func setupUserRepo(t *testing.T) *UserRepo {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	// Init admin to create users table
	db.InitAdmin(database, "testpass123")
	return NewUserRepo(database)
}

func TestCreateAndGetUser(t *testing.T) {
	r := setupUserRepo(t)
	u := model.User{Username: "testuser", Password: "hash123", Role: "user", CreatedAt: nowISO()}
	id, err := r.Create(u)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id != 2 { // admin is id=1
		t.Errorf("expected id=2, got %d", id)
	}

	got, err := r.GetByUsername("testuser")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Username != "testuser" || got.Role != "user" {
		t.Errorf("unexpected user: %+v", got)
	}
}

func TestCreateDuplicate(t *testing.T) {
	r := setupUserRepo(t)
	r.Create(model.User{Username: "dup", Password: "x", Role: "user", CreatedAt: nowISO()})
	_, err := r.Create(model.User{Username: "dup", Password: "y", Role: "user", CreatedAt: nowISO()})
	if err != ErrDuplicate {
		t.Errorf("expected ErrDuplicate, got %v", err)
	}
}

func TestGetByUsernameNotFound(t *testing.T) {
	r := setupUserRepo(t)
	_, err := r.GetByUsername("noone")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/repo/... -v -run TestCreateAndGetUser`
Expected: FAIL — `UserRepo` not defined

- [ ] **Step 3: Implement user.go**

```go
package repo

import (
	"database/sql"

	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/mattn/go-sqlite3"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(u model.User) (int, error) {
	result, err := r.db.Exec(
		"INSERT INTO users (username, password, role, created_at) VALUES (?, ?, ?, ?)",
		u.Username, u.Password, u.Role, u.CreatedAt,
	)
	if err != nil {
		var sqliteErr sqlite3.Error
		var e sqlite3.Error
		if errors.As(err, &e) && e.Code == sqlite3.ErrConstraint {
			return 0, ErrDuplicate
		}
		_ = sqliteErr
		return 0, ErrDuplicate // fallback for unique constraint
	}
	id, _ := result.LastInsertId()
	return int(id), nil
}

func (r *UserRepo) GetByUsername(username string) (*model.User, error) {
	var u model.User
	var role string
	err := r.db.QueryRow(
		"SELECT id, username, password, role, created_at FROM users WHERE username = ?",
		username,
	).Scan(&u.ID, &u.Username, &u.Password, &role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.Role = role
	return &u, nil
}

func (r *UserRepo) GetByID(id int) (*model.User, error) {
	var u model.User
	var role string
	err := r.db.QueryRow(
		"SELECT id, username, password, role, created_at FROM users WHERE id = ?",
		id,
	).Scan(&u.ID, &u.Username, &u.Password, &role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.Role = role
	return &u, nil
}
```

Add `"errors"` to the import block.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/repo/... -v -run TestCreate`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/repo/user.go internal/repo/user_test.go
git commit -m "feat: add user repository with Create and GetByUsername"
```

---

### Task 5: InviteCode repository

**Files:**
- Create: `internal/repo/invite_code.go`
- Create: `internal/repo/invite_code_test.go`

- [ ] **Step 1: Write invite_code_test.go**

```go
package repo

import (
	"testing"

	"github.com/black-eleven/stock-monitor/internal/db"
	"github.com/black-eleven/stock-monitor/internal/model"
)

func setupInviteCodeRepo(t *testing.T) (*InviteCodeRepo, *UserRepo) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	db.InitAdmin(database, "testpass123")
	return NewInviteCodeRepo(database), NewUserRepo(database)
}

func TestCreateAndUseInviteCode(t *testing.T) {
	r, ur := setupInviteCodeRepo(t)
	u, _ := ur.GetByUsername("admin")

	code := model.InviteCode{
		Code: "TEST-CODE-001", MaxUses: 2, UsedCount: 0,
		CreatedBy: u.ID, CreatedAt: nowISO(), IsActive: true,
	}
	id, err := r.Create(code)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id == 0 {
		t.Errorf("expected non-zero id")
	}

	// Verify
	got, err := r.GetByCode("TEST-CODE-001")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.IsActive || got.UsedCount != 0 {
		t.Errorf("unexpected: %+v", got)
	}

	// Use it
	err = r.IncrementUsed("TEST-CODE-001")
	if err != nil {
		t.Fatalf("increment: %v", err)
	}
	got2, _ := r.GetByCode("TEST-CODE-001")
	if got2.UsedCount != 1 {
		t.Errorf("expected usedCount=1, got %d", got2.UsedCount)
	}

	// Use again (should still work since maxUses=2)
	err = r.IncrementUsed("TEST-CODE-001")
	if err != nil {
		t.Errorf("second use failed: %v", err)
	}

	// Third use should fail (maxUses=2 exceeded)
	err = r.IncrementUsed("TEST-CODE-001")
	if err == nil {
		t.Errorf("expected error on over-use")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/repo/... -v -run TestCreateAndUseInviteCode`
Expected: FAIL — `InviteCodeRepo` not defined

- [ ] **Step 3: Implement invite_code.go**

```go
package repo

import (
	"database/sql"
	"errors"

	"github.com/black-eleven/stock-monitor/internal/model"
)

var ErrCodeExpired = errors.New("invite code expired or max uses reached")

type InviteCodeRepo struct {
	db *sql.DB
}

func NewInviteCodeRepo(db *sql.DB) *InviteCodeRepo {
	return &InviteCodeRepo{db: db}
}

func (r *InviteCodeRepo) Create(c model.InviteCode) (int, error) {
	isActive := 0
	if c.IsActive { isActive = 1 }
	result, err := r.db.Exec(
		"INSERT INTO invite_codes (code, max_uses, used_count, created_by, created_at, is_active) VALUES (?, ?, ?, ?, ?, ?)",
		c.Code, c.MaxUses, c.UsedCount, c.CreatedBy, c.CreatedAt, isActive,
	)
	if err != nil {
		return 0, ErrDuplicate
	}
	id, _ := result.LastInsertId()
	return int(id), nil
}

func (r *InviteCodeRepo) GetByCode(code string) (*model.InviteCode, error) {
	var c model.InviteCode
	var isActive int
	err := r.db.QueryRow(
		"SELECT id, code, max_uses, used_count, created_by, created_at, is_active FROM invite_codes WHERE code = ?",
		code,
	).Scan(&c.ID, &c.Code, &c.MaxUses, &c.UsedCount, &c.CreatedBy, &c.CreatedAt, &isActive)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.IsActive = isActive != 0
	return &c, nil
}

func (r *InviteCodeRepo) ListByCreator(creatorID int) ([]model.InviteCode, error) {
	rows, err := r.db.Query(
		"SELECT id, code, max_uses, used_count, created_by, created_at, is_active FROM invite_codes WHERE created_by = ? ORDER BY id DESC",
		creatorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []model.InviteCode
	for rows.Next() {
		var c model.InviteCode
		var isActive int
		if err := rows.Scan(&c.ID, &c.Code, &c.MaxUses, &c.UsedCount, &c.CreatedBy, &c.CreatedAt, &isActive); err != nil {
			return nil, err
		}
		c.IsActive = isActive != 0
		codes = append(codes, c)
	}
	if codes == nil {
		codes = []model.InviteCode{}
	}
	return codes, nil
}

func (r *InviteCodeRepo) IncrementUsed(code string) error {
	var maxUses, usedCount, isActive int
	err := r.db.QueryRow("SELECT max_uses, used_count, is_active FROM invite_codes WHERE code = ?", code).
		Scan(&maxUses, &usedCount, &isActive)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if isActive == 0 || (maxUses > 0 && usedCount >= maxUses) {
		return ErrCodeExpired
	}
	_, err = r.db.Exec("UPDATE invite_codes SET used_count = used_count + 1 WHERE code = ?", code)
	return err
}

func (r *InviteCodeRepo) SetActive(id int, active bool) error {
	isActive := 0
	if active { isActive = 1 }
	result, err := r.db.Exec("UPDATE invite_codes SET is_active = ? WHERE id = ?", isActive, id)
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

Run: `go test ./internal/repo/... -v -run TestCreateAndUseInviteCode`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/repo/invite_code.go internal/repo/invite_code_test.go
git commit -m "feat: add invite code repository with CRUD and usage tracking"
```

---

### Task 6: JWT middleware

**Files:**
- Create: `internal/middleware/auth.go`

- [ ] **Step 1: Install golang-jwt**

Run: `go get github.com/golang-jwt/jwt/v5`

- [ ] **Step 2: Write auth.go**

```go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Context keys
const (
	CtxUserID   = "user_id"
	CtxUsername = "username"
	CtxRole     = "role"
)

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			return
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		userID, _ := claims["user_id"].(float64)
		username, _ := claims["username"].(string)
		role, _ := claims["role"].(string)

		c.Set(CtxUserID, int(userID))
		c.Set(CtxUsername, username)
		c.Set(CtxRole, role)
		c.Next()
	}
}

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(CtxRole)
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}
		c.Next()
	}
}

// Helper to get user ID from context
func GetUserID(c *gin.Context) int {
	id, _ := c.Get(CtxUserID)
	return id.(int)
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/middleware/...`
Expected: (no errors)

- [ ] **Step 4: Commit**

```bash
git add internal/middleware/ go.mod go.sum
git commit -m "feat: add JWT auth middleware and AdminRequired guard"
```

---

### Task 7: Auth handler (register + login)

**Files:**
- Create: `internal/handler/auth.go`

- [ ] **Step 1: Write auth.go**

```go
package handler

import (
	"net/http"
	"time"

	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userRepo       *repo.UserRepo
	inviteCodeRepo *repo.InviteCodeRepo
	jwtSecret      string
}

func NewAuthHandler(userRepo *repo.UserRepo, inviteCodeRepo *repo.InviteCodeRepo, jwtSecret string) *AuthHandler {
	return &AuthHandler{userRepo: userRepo, inviteCodeRepo: inviteCodeRepo, jwtSecret: jwtSecret}
}

func (h *AuthHandler) Register(api *gin.RouterGroup) {
	api.POST("/auth/register", h.register)
	api.POST("/auth/login", h.login)
}

func (h *AuthHandler) register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username, password, and inviteCode are required"})
		return
	}
	if len(req.Username) < 2 || len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username at least 2 chars, password at least 6 chars"})
		return
	}

	// Only validate and consume invite code for non-admin users
	// (admins can register without invite code)
	inviteCode, err := h.inviteCodeRepo.GetByCode(req.InviteCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invite code"})
		return
	}

	if err := h.inviteCodeRepo.IncrementUsed(req.InviteCode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invite code is expired or exhausted"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
		return
	}

	u := model.User{
		Username:  req.Username,
		Password:  string(hash),
		Role:      "user",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	id, err := h.userRepo.Create(u)
	if err != nil {
		if err == repo.ErrDuplicate {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	token, err := h.generateToken(id, u.Username, u.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	u.ID = id
	_ = inviteCode // suppress unused
	c.JSON(http.StatusCreated, model.LoginResponse{Token: token, User: u})
}

func (h *AuthHandler) login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
		return
	}

	u, err := h.userRepo.GetByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	token, err := h.generateToken(u.ID, u.Username, u.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, model.LoginResponse{Token: token, User: *u})
}

func (h *AuthHandler) generateToken(userID int, username, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"role":     role,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/handler/...`
Expected: (no errors)

- [ ] **Step 3: Commit**

```bash
git add internal/handler/auth.go
git commit -m "feat: add auth handler with register and login endpoints"
```

---

### Task 8: Admin handler (invite codes management)

**Files:**
- Create: `internal/handler/admin.go`

- [ ] **Step 1: Write admin.go**

```go
package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	inviteCodeRepo *repo.InviteCodeRepo
}

func NewAdminHandler(inviteCodeRepo *repo.InviteCodeRepo) *AdminHandler {
	return &AdminHandler{inviteCodeRepo: inviteCodeRepo}
}

func (h *AdminHandler) Register(admin *gin.RouterGroup) {
	admin.POST("/invite-codes", h.createCodes)
	admin.GET("/invite-codes", h.listCodes)
	admin.PUT("/invite-codes/:id", h.updateCode)
}

func (h *AdminHandler) createCodes(c *gin.Context) {
	var req model.CreateInviteCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.MaxUses = 1
		req.Count = 1
	}
	if req.Count <= 0 {
		req.Count = 1
	}

	userID, _ := c.Get("user_id")
	now := time.Now().UTC().Format(time.RFC3339)
	var codes []model.InviteCode

	for i := 0; i < req.Count; i++ {
		b := make([]byte, 8)
		rand.Read(b)
		code := model.InviteCode{
			Code:      hex.EncodeToString(b),
			MaxUses:   req.MaxUses,
			CreatedBy: userID.(int),
			CreatedAt: now,
			IsActive:  true,
		}
		id, err := h.inviteCodeRepo.Create(code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create code"})
			return
		}
		code.ID = id
		codes = append(codes, code)
	}

	c.JSON(http.StatusCreated, codes)
}

func (h *AdminHandler) listCodes(c *gin.Context) {
	userID, _ := c.Get("user_id")
	codes, err := h.inviteCodeRepo.ListByCreator(userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list codes"})
		return
	}
	c.JSON(http.StatusOK, codes)
}

func (h *AdminHandler) updateCode(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}

	var req struct {
		IsActive *bool `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid body"})
		return
	}

	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	if err := h.inviteCodeRepo.SetActive(id, active); err != nil {
		if err == repo.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Code not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/handler/...`
Expected: (no errors)

- [ ] **Step 3: Commit**

```bash
git add internal/handler/admin.go
git commit -m "feat: add admin handler for invite code management"
```

---

### Task 9: Modify existing repos — add user_id filtering

**Files:**
- Modify: `internal/repo/watchlist.go`
- Modify: `internal/repo/alert.go`
- Modify: `internal/repo/holding.go`

Note: The repo package returns `(int, error)` for Create methods and needs user_id in all queries. Only the method signatures change — all methods gain a `userID int` first parameter.

- [ ] **Step 1: Modify watchlist.go**

Change ALL method signatures and SQL queries. The `GetAll()`, `Add()`, `Remove()` methods all need `userID int` parameter and `WHERE user_id = ?` clause:

```go
func (r *WatchlistRepo) GetAll(userID int) ([]model.WatchlistItem, error) {
	rows, err := r.db.Query(
		"SELECT symbol, name, added_at FROM watchlist WHERE user_id = ? ORDER BY added_at DESC",
		userID,
	)
	// ... rest unchanged
}

func (r *WatchlistRepo) Add(userID int, item model.WatchlistItem) error {
	_, err := r.db.Exec(
		"INSERT INTO watchlist (symbol, name, added_at, user_id) VALUES (?, ?, ?, ?)",
		item.Symbol, item.Name, item.AddedAt, userID,
	)
	// ... rest unchanged
}

func (r *WatchlistRepo) Remove(userID int, symbol string) error {
	result, err := r.db.Exec("DELETE FROM watchlist WHERE symbol = ? AND user_id = ?", symbol, userID)
	// ... rest unchanged
}
```

- [ ] **Step 2: Modify alert.go**

Change `GetAll(userID int)`, `GetBySymbol(userID int, symbol string)`, `Add(userID int, rule model.AlertRule)`. The alerts table now has `user_id`, so all queries filter by it. The `AppendLog` and `GetLogs` methods don't need user_id since logs are tied to alert IDs which already have user filtering.

For the `alert_test.go` — update all calls to pass `userID` as `1` (the admin x️ user from InitAdmin).

- [ ] **Step 3: Modify holding.go**

Same pattern: `GetAll(userID int)`, `Add(userID int, h model.Holding)`, `Update(userID int, symbol string, fn)`, `Remove(userID int, symbol string)`.

- [ ] **Step 4: Update test files**

`watchlist_test.go`, `alert_test.go`, `holding_test.go` — update all repo method calls to pass `userID` as `1`.

- [ ] **Step 5: Run all tests**

Run: `go test ./internal/repo/... -v`
Expected: PASS (all tests)

- [ ] **Step 6: Commit**

```bash
git add internal/repo/
git commit -m "feat: add user_id filtering to all existing repositories"
```

---

### Task 10: Modify existing handlers — extract user_id from context

**Files:**
- Modify: `internal/handler/watchlist.go`
- Modify: `internal/handler/alert.go`
- Modify: `internal/handler/holding.go`

- [ ] **Step 1: Update each handler to extract user_id**

In each handler method, add at the top:
```go
userID := middleware.GetUserID(c)
```

And pass `userID` to all repo calls. Example for watchlist.go `getAll`:
```go
func (h *WatchlistHandler) getAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	items, err := h.repo.GetAll(userID)
	// ... rest unchanged
}
```

Apply the same pattern to `add`, `remove` in watchlist; all methods in alert; all methods in holding.

Add `"github.com/black-eleven/stock-monitor/internal/middleware"` import to all three handler files.

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/handler/...`
Expected: (no errors)

- [ ] **Step 3: Commit**

```bash
git add internal/handler/
git commit -m "feat: inject user_id from JWT context into all handlers"
```

---

### Task 11: WS hub — JWT auth on WebSocket

**Files:**
- Modify: `internal/ws/hub.go`

- [ ] **Step 1: Add JWT validation to ServeWS**

Add `jwtSecret` field to `Hub` and update `NewHub`:

```go
type Hub struct {
	// ... existing fields
	jwtSecret string
}

func NewHub(jwtSecret string) *Hub {
	return &Hub{
		// ... existing init
		jwtSecret: jwtSecret,
	}
}
```

Update `ServeWS` to validate token from query param:
```go
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" || !h.validateToken(token) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// ... existing upgrade logic
}

func (h *Hub) validateToken(tokenStr string) bool {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	})
	return err == nil && token.Valid
}
```

Add `"github.com/golang-jwt/jwt/v5"` to imports.

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/ws/...`
Expected: (no errors)

- [ ] **Step 3: Commit**

```bash
git add internal/ws/hub.go
git commit -m "feat: add JWT auth to WebSocket connections"
```

---

### Task 12: Wire up main.go — full auth integration

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Add new components**

Add after `holdingRepo`:
```go
userRepo := repo.NewUserRepo(database)
inviteCodeRepo := repo.NewInviteCodeRepo(database)
```

Add after `hub` creation:
```go
// Init admin user if first run
adminID, err := db.InitAdmin(database, cfg.AdminPassword)
if err != nil {
	log.Fatalf("Failed to init admin: %v", err)
}
if adminID > 0 {
	log.Printf("[MAIN] Admin user created (id=%d), password: %s", adminID, cfg.AdminPassword)
	log.Printf("[MAIN] An initial invite code has been generated — check logs above")
}
```

Add auth and admin handlers:
```go
authH := handler.NewAuthHandler(userRepo, inviteCodeRepo, cfg.JwtSecret)
adminH := handler.NewAdminHandler(inviteCodeRepo)
```

Add middleware import and restructure routes:
```go
authMW := middleware.AuthMiddleware(cfg.JwtSecret)

r := gin.Default()
api := r.Group("/api")

// Public routes (no auth)
authH.Register(api)

// Protected routes (JWT required)
auth := api.Group("", authMW)
watchlistH.Register(auth)
alertH.Register(auth)
holdingH.Register(auth)
quoteH.Register(auth)
klineH.Register(auth)

// Admin routes
admin := auth.Group("/admin", middleware.AdminRequired())
adminH.Register(admin)
```

Update ws route:
```go
r.GET("/ws", func(c *gin.Context) { hub.ServeWS(c.Writer, c.Request) })
```

Update `NewHub` call:
```go
hub := ws.NewHub(cfg.JwtSecret)
```

- [ ] **Step 2: Build**

Run: `go build ./cmd/server`
Expected: (no errors)

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: wire auth middleware, handlers, and admin init into main"
```

---

## Phase 2: Web Frontend

### Task 13: auth.js — AuthManager class

**Files:**
- Create: `web/js/auth.js`

- [ ] **Step 1: Write auth.js**

```javascript
class AuthManager {
  constructor() {
    this.token = localStorage.getItem('token');
    this.user = JSON.parse(localStorage.getItem('user') || 'null');
  }

  isLoggedIn() {
    if (!this.token) return false;
    try {
      const payload = JSON.parse(atob(this.token.split('.')[1]));
      return payload.exp * 1000 > Date.now();
    } catch {
      return false;
    }
  }

  isAdmin() {
    return this.user && this.user.role === 'admin';
  }

  async login(username, password) {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    });
    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || 'Login failed');
    }
    const data = await res.json();
    this.token = data.token;
    this.user = data.user;
    localStorage.setItem('token', data.token);
    localStorage.setItem('user', JSON.stringify(data.user));
    return data;
  }

  async register(username, password, inviteCode) {
    const res = await fetch('/api/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password, inviteCode }),
    });
    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || 'Registration failed');
    }
    const data = await res.json();
    this.token = data.token;
    this.user = data.user;
    localStorage.setItem('token', data.token);
    localStorage.setItem('user', JSON.stringify(data.user));
    return data;
  }

  logout() {
    this.token = null;
    this.user = null;
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    window.location.href = '/login.html';
  }

  getAuthHeaders() {
    return this.token ? { 'Authorization': 'Bearer ' + this.token } : {};
  }
}

const auth = new AuthManager();
```

- [ ] **Step 2: Commit**

```bash
git add web/js/auth.js
git commit -m "feat: add AuthManager class for login/register/token management"
```

---

### Task 14: login.html — login/register page

**Files:**
- Create: `web/login.html`

- [ ] **Step 1: Write login.html**

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Stock Monitor - 登录</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, sans-serif;
    background: #0d1117;
    color: #e6edf3;
    display: flex; justify-content: center; align-items: center;
    min-height: 100vh;
  }
  .login-box {
    background: #161b22;
    padding: 32px;
    border-radius: 12px;
    border: 1px solid #30363d;
    width: 360px;
  }
  h1 { text-align: center; margin-bottom: 24px; font-size: 20px; }
  .form-group { margin-bottom: 16px; }
  label { display: block; margin-bottom: 6px; font-size: 13px; color: #8b949e; }
  input {
    width: 100%; padding: 10px 12px;
    background: #0d1117; border: 1px solid #30363d; border-radius: 6px;
    color: #e6edf3; font-size: 14px;
  }
  input:focus { outline: none; border-color: #1f6feb; }
  button {
    width: 100%; padding: 10px; margin-top: 8px;
    background: #1f6feb; color: white; border: none;
    border-radius: 6px; font-size: 14px; cursor: pointer;
  }
  button:hover { background: #388bfd; }
  .error { color: #f85149; font-size: 13px; margin-top: 8px; text-align: center; }
  .toggle { text-align: center; margin-top: 16px; font-size: 13px; color: #8b949e; }
  .toggle a { color: #58a6ff; cursor: pointer; text-decoration: none; }
  .hidden { display: none; }
</style>
</head>
<body>
<div class="login-box">
  <h1>Stock Monitor</h1>

  <!-- Login Form -->
  <form id="loginForm">
    <div class="form-group">
      <label>用户名</label>
      <input type="text" name="username" required autocomplete="username">
    </div>
    <div class="form-group">
      <label>密码</label>
      <input type="password" name="password" required autocomplete="current-password">
    </div>
    <div id="loginError" class="error hidden"></div>
    <button type="submit">登录</button>
  </form>

  <!-- Register Form -->
  <form id="registerForm" class="hidden">
    <div class="form-group">
      <label>用户名</label>
      <input type="text" name="username" required minlength="2" autocomplete="username">
    </div>
    <div class="form-group">
      <label>密码</label>
      <input type="password" name="password" required minlength="6" autocomplete="new-password">
    </div>
    <div class="form-group">
      <label>确认密码</label>
      <input type="password" name="confirmPassword" required minlength="6">
    </div>
    <div class="form-group">
      <label>邀请码</label>
      <input type="text" name="inviteCode" required>
    </div>
    <div id="registerError" class="error hidden"></div>
    <button type="submit">注册</button>
  </form>

  <div class="toggle">
    <span id="toggleText">没有账号？</span>
    <a id="toggleLink">注册</a>
  </div>
</div>

<script src="/js/auth.js"></script>
<script>
  // If already logged in, redirect
  if (auth.isLoggedIn()) {
    window.location.href = '/';
  }

  let isRegister = false;

  document.getElementById('toggleLink').addEventListener('click', () => {
    isRegister = !isRegister;
    document.getElementById('loginForm').classList.toggle('hidden', isRegister);
    document.getElementById('registerForm').classList.toggle('hidden', !isRegister);
    document.getElementById('toggleText').textContent = isRegister ? '已有账号？' : '没有账号？';
    document.getElementById('toggleLink').textContent = isRegister ? '登录' : '注册';
  });

  document.getElementById('loginForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const data = new FormData(e.target);
    const errEl = document.getElementById('loginError');
    try {
      await auth.login(data.get('username'), data.get('password'));
      window.location.href = '/';
    } catch (err) {
      errEl.textContent = err.message;
      errEl.classList.remove('hidden');
    }
  });

  document.getElementById('registerForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const data = new FormData(e.target);
    const errEl = document.getElementById('registerError');
    if (data.get('password') !== data.get('confirmPassword')) {
      errEl.textContent = '两次输入的密码不一致';
      errEl.classList.remove('hidden');
      return;
    }
    try {
      await auth.register(data.get('username'), data.get('password'), data.get('inviteCode'));
      window.location.href = '/';
    } catch (err) {
      errEl.textContent = err.message;
      errEl.classList.remove('hidden');
    }
  });
</script>
</body>
</html>
```

- [ ] **Step 2: Commit**

```bash
git add web/login.html
git commit -m "feat: add login/register page"
```

---

### Task 15: api.js — add token to all requests

**Files:**
- Modify: `web/js/api.js`

- [ ] **Step 1: Add auth headers to all fetch calls**

Add an `_headers()` helper method:
```javascript
  _headers() {
    return {
      'Content-Type': 'application/json',
      ...auth.getAuthHeaders(),
    };
  }
```

Update `get`, `post`, `put`, `del` to use it and handle 401:
```javascript
  async get(path) {
    const res = await fetch(path, { headers: auth.getAuthHeaders() });
    if (res.status === 401) { auth.logout(); return; }
    if (!res.ok) throw new Error(`GET ${path} failed: ${res.status}`);
    return res.json();
  }

  async post(path, body) {
    const res = await fetch(path, {
      method: 'POST',
      headers: this._headers(),
      body: JSON.stringify(body),
    });
    if (res.status === 401) { auth.logout(); return; }
    if (!res.ok) throw new Error(`POST ${path} failed: ${res.status}`);
    return res.json();
  }

  async put(path, body) {
    const res = await fetch(path, {
      method: 'PUT',
      headers: this._headers(),
      body: JSON.stringify(body),
    });
    if (res.status === 401) { auth.logout(); return; }
    if (!res.ok) throw new Error(`PUT ${path} failed: ${res.status}`);
    return res.json();
  }

  async del(path) {
    const res = await fetch(path, {
      method: 'DELETE',
      headers: auth.getAuthHeaders(),
    });
    if (res.status === 401) { auth.logout(); return; }
    if (!res.ok) throw new Error(`DELETE ${path} failed: ${res.status}`);
    return res.json();
  }
```

Update `connectWs` to include token:
```javascript
  connectWs() {
    if (!auth.token) return;
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    this.ws = new WebSocket(`${protocol}//${location.host}/ws?token=${auth.token}`);
    // ... rest unchanged
  }
```

- [ ] **Step 2: Commit**

```bash
git add web/js/api.js
git commit -m "feat: add JWT token to all API requests and WebSocket connection"
```

---

### Task 16: app.js — auth guard

**Files:**
- Modify: `web/js/app.js`

- [ ] **Step 1: Add auth check before init**

Add at the beginning of the `init()` function:
```javascript
async function init() {
  // Auth guard
  if (!auth.isLoggedIn()) {
    window.location.href = '/login.html';
    return;
  }

  // ... rest unchanged
}
```

Add connection status area to show current user:
```javascript
  // Show current user and logout button
  const connEl = document.getElementById('connStatus');
  if (auth.user) {
    connEl.textContent = `${auth.user.username} | 已连接`;
  }
```

Add logout button handler (in the `DOMContentLoaded` listener before `init()`):
```javascript
document.getElementById('logoutBtn').addEventListener('click', () => auth.logout());
```

- [ ] **Step 2: Update index.html**

In `web/index.html`, add a logout button near the connection status indicator and a link to admin page for admin users. This requires reading the existing index.html and adding:

```html
<span id="connStatus" class="connection-status">未连接</span>
<button id="logoutBtn" style="background:none;border:1px solid #30363d;color:#e6edf3;padding:4px 12px;border-radius:4px;cursor:pointer;margin-left:8px;">退出</button>
<a id="adminLink" href="/admin.html" style="display:none;color:#58a6ff;margin-left:8px;">管理</a>
```

And in init(), show the admin link if user is admin:
```javascript
if (auth.isAdmin()) {
  document.getElementById('adminLink').style.display = 'inline';
}
```

- [ ] **Step 3: Commit**

```bash
git add web/js/app.js web/index.html
git commit -m "feat: add auth guard, logout button, and admin link to main page"
```

---

### Task 17: admin.html + admin.js — invite code management

**Files:**
- Create: `web/admin.html`
- Create: `web/js/admin.js`

- [ ] **Step 1: Write admin.html**

A simple admin page with:
- Auth guard (redirect if not logged in or not admin)
- "生成邀请码" form: maxUses input + count input + submit
- Invite code list table: code, maxUses, usedCount, status, created date
- Enable/disable toggle per code

Full HTML with dark theme matching login.html style. Include `/js/auth.js` and `/js/admin.js`.

- [ ] **Step 2: Write admin.js**

```javascript
if (!auth.isLoggedIn() || !auth.isAdmin()) {
  window.location.href = '/login.html';
}

async function loadCodes() {
  const res = await fetch('/api/admin/invite-codes', { headers: auth.getAuthHeaders() });
  if (res.status === 401) { auth.logout(); return; }
  if (!res.ok) return;
  const codes = await res.json();
  renderTable(codes);
}

function renderTable(codes) {
  const tbody = document.getElementById('codeTableBody');
  tbody.innerHTML = codes.map(c => `
    <tr>
      <td>${escapeHtml(c.code)}</td>
      <td>${c.maxUses === 0 ? '∞' : c.maxUses}</td>
      <td>${c.usedCount}</td>
      <td><span style="color:${c.isActive ? '#3fb950' : '#f85149'}">${c.isActive ? '有效' : '已禁用'}</span></td>
      <td>${c.createdAt}</td>
      <td>
        <button onclick="toggleCode(${c.id}, ${!c.isActive})" style="background:none;border:1px solid #30363d;color:#e6edf3;padding:4px 8px;border-radius:4px;cursor:pointer;">
          ${c.isActive ? '禁用' : '启用'}
        </button>
      </td>
    </tr>
  `).join('');
}

async function toggleCode(id, active) {
  await fetch(`/api/admin/invite-codes/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...auth.getAuthHeaders() },
    body: JSON.stringify({ isActive: active }),
  });
  loadCodes();
}

async function generateCodes(e) {
  e.preventDefault();
  const data = new FormData(e.target);
  const res = await fetch('/api/admin/invite-codes', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...auth.getAuthHeaders() },
    body: JSON.stringify({
      maxUses: parseInt(data.get('maxUses')) || 1,
      count: parseInt(data.get('count')) || 1,
    }),
  });
  if (!res.ok) { alert('Failed'); return; }
  const codes = await res.json();
  alert(`Created ${codes.length} invite code(s):\n${codes.map(c => c.code).join('\n')}`);
  loadCodes();
}

document.getElementById('generateForm').addEventListener('submit', generateCodes);
loadCodes();
```

- [ ] **Step 3: Commit**

```bash
git add web/admin.html web/js/admin.js
git commit -m "feat: add admin page for invite code management"
```

---

## Phase 3: Flutter Mobile App

### Task 18: auth_provider.dart

**Files:**
- Create: `mobile/stock_monitor/lib/presentation/providers/auth_provider.dart`

- [ ] **Step 1: Write auth_provider.dart**

```dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

class AuthState {
  final String? token;
  final String? username;
  final String? role;
  final bool isLoggedIn;

  const AuthState({this.token, this.username, this.role, this.isLoggedIn = false});
}

class AuthNotifier extends StateNotifier<AuthState> {
  AuthNotifier() : super(const AuthState()) {
    _loadFromStorage();
  }

  Future<void> _loadFromStorage() async {
    final prefs = await SharedPreferences.getInstance();
    final token = prefs.getString('token');
    final username = prefs.getString('username');
    final role = prefs.getString('role');
    if (token != null) {
      state = AuthState(token: token, username: username, role: role, isLoggedIn: true);
    }
  }

  Future<void> login(String token, String username, String role) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('token', token);
    await prefs.setString('username', username);
    await prefs.setString('role', role);
    state = AuthState(token: token, username: username, role: role, isLoggedIn: true);
  }

  Future<void> logout() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove('token');
    await prefs.remove('username');
    await prefs.remove('role');
    state = const AuthState();
  }
}

final authProvider = StateNotifierProvider<AuthNotifier, AuthState>((ref) {
  return AuthNotifier();
});
```

- [ ] **Step 2: Commit**

```bash
git add mobile/stock_monitor/lib/presentation/providers/auth_provider.dart
git commit -m "feat: add Flutter auth provider with SharedPreferences persistence"
```

---

### Task 19: login_screen.dart

**Files:**
- Create: `mobile/stock_monitor/lib/presentation/screens/login_screen.dart`

- [ ] **Step 1: Write login_screen.dart**

A Flutter screen with:
- Dark theme (consistent with existing AppTheme)
- Login form: username + password + login button
- Register form: username + password + confirm password + invite code + register button
- Toggle between login/register
- Error display
- On success, call `authProvider.login(...)` and navigate to main

Uses `dio` to call `/api/auth/login` and `/api/auth/register` directly.

Full code ~150 lines of Flutter widget code with form validation.

- [ ] **Step 2: Commit**

```bash
git add mobile/stock_monitor/lib/presentation/screens/login_screen.dart
git commit -m "feat: add Flutter login/register screen"
```

---

### Task 20: api_client.dart + app.dart — auth integration

**Files:**
- Modify: `mobile/stock_monitor/lib/data/api/api_client.dart`
- Modify: `mobile/stock_monitor/lib/presentation/providers/api_providers.dart`
- Modify: `mobile/stock_monitor/lib/app.dart`
- Modify: `mobile/stock_monitor/lib/main.dart`

- [ ] **Step 1: Update api_client.dart**

Add Dio interceptor that reads token from SharedPreferences and adds `Authorization` header:

```dart
ApiClient() {
  dio = Dio(BaseOptions(
    baseUrl: AppConfig.baseUrl,
    connectTimeout: const Duration(seconds: 10),
    receiveTimeout: const Duration(seconds: 10),
    headers: {'Content-Type': 'application/json'},
  ));
  dio.interceptors.add(InterceptorsWrapper(
    onRequest: (options, handler) async {
      final prefs = await SharedPreferences.getInstance();
      final token = prefs.getString('token');
      if (token != null) {
        options.headers['Authorization'] = 'Bearer $token';
      }
      handler.next(options);
    },
    onError: (error, handler) {
      if (error.response?.statusCode == 401) {
        // Trigger logout via a callback or just skip 401 handling per-request
      }
      handler.next(error);
    },
  ));
}
```

Add `import 'package:shared_preferences/shared_preferences.dart';`.

- [ ] **Step 2: Update ws_client.dart**

Read token and pass via query param:
```dart
void connect() async {
  final prefs = await SharedPreferences.getInstance();
  final token = prefs.getString('token');
  if (token == null) return;
  _channel = WebSocketChannel.connect(Uri.parse('${AppConfig.wsUrl}?token=$token'));
  // ... rest unchanged
}
```

- [ ] **Step 3: Update api_providers.dart**

Add import for auth_provider:
```dart
import 'auth_provider.dart';
```

- [ ] **Step 4: Update app.dart**

Add auth guard in router. Add a redirect guard that checks `authProvider` state and redirects to `/login` if not logged in. Add `/login` route for `LoginScreen`.

- [ ] **Step 5: Verify compilation**

Run: `cd mobile/stock_monitor && flutter analyze lib/ 2>&1 | tail -5`
Expected: No issues found

- [ ] **Step 6: Commit**

```bash
git add mobile/stock_monitor/lib/
git commit -m "feat: integrate Flutter auth with Dio interceptor, WS token, and route guard"
```

---

## Phase 4: Migration Tool

### Task 21: Update cmd/migrate

**Files:**
- Modify: `cmd/migrate/main.go`

- [ ] **Step 1: Add user_id migration step**

After the existing migrate functions, add:
```go
func migrateUserIDColumn(db *sql.DB) {
	alterStatements := []string{
		"ALTER TABLE watchlist ADD COLUMN user_id INTEGER DEFAULT 0",
		"ALTER TABLE holdings ADD COLUMN user_id INTEGER DEFAULT 0",
		"ALTER TABLE alerts ADD COLUMN user_id INTEGER DEFAULT 0",
	}
	for _, stmt := range alterStatements {
		_, err := db.Exec(stmt)
		if err != nil {
			fmt.Printf("migration note: %v (column may already exist)\n", err)
		} else {
			fmt.Printf("migrated: %s\n", stmt)
		}
	}
}
```

Call `migrateUserIDColumn(database)` in `main()` before the other migrate functions.

- [ ] **Step 2: Verify compilation**

Run: `go build ./cmd/migrate`
Expected: (no errors)

- [ ] **Step 3: Commit**

```bash
git add cmd/migrate/main.go
git commit -m "feat: add user_id column migration to migrate tool"
```

---

## Self-Review Checklist

- [x] Spec coverage: JWT middleware, register/login with invite code, admin invite code management, user_id data isolation, WS auth, Web + Flutter login pages
- [x] No placeholders: all tasks have complete code
- [x] Type consistency: `user_id` is `int` throughout (matching SQLite INTEGER), JWT payload uses `float64` (standard jwt-go behavior)
- [x] Task 11 jwtSecret passed via NewHub, matches main.go wiring in Task 12
- [x] Admin init creates initial invite code printed to log
- [x] Existing test files updated in Task 9 to pass userID parameter
- [x] `golang-jwt/jwt/v5` installed in Task 6, used in Tasks 7, 11
- [x] `bcrypt` imported from `golang.org/x/crypto` (already available)

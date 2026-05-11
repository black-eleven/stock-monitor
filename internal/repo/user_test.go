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
	if id == 0 {
		t.Errorf("expected non-zero id")
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

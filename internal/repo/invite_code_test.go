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

	got, err := r.GetByCode("TEST-CODE-001")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.IsActive || got.UsedCount != 0 {
		t.Errorf("unexpected: %+v", got)
	}

	// Use it twice (maxUses=2)
	if err := r.IncrementUsed("TEST-CODE-001"); err != nil {
		t.Fatalf("increment: %v", err)
	}
	if err := r.IncrementUsed("TEST-CODE-001"); err != nil {
		t.Fatalf("second use: %v", err)
	}

	// Third use should fail
	if err := r.IncrementUsed("TEST-CODE-001"); err == nil {
		t.Errorf("expected error on over-use")
	}

	// Disable
	if err := r.SetActive(id, false); err != nil {
		t.Fatalf("setActive: %v", err)
	}
	got2, _ := r.GetByCode("TEST-CODE-001")
	if got2.IsActive {
		t.Errorf("expected disabled")
	}
}

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
	if c.IsActive {
		isActive = 1
	}
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
	if active {
		isActive = 1
	}
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

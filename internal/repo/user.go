package repo

import (
	"database/sql"

	"github.com/black-eleven/stock-monitor/internal/model"
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
		return 0, ErrDuplicate
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

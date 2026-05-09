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

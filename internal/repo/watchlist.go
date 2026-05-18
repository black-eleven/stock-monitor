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

func (r *WatchlistRepo) GetAll(userID int) ([]model.WatchlistItem, error) {
	rows, err := r.db.Query("SELECT symbol, name, added_at FROM watchlist WHERE user_id = ? ORDER BY added_at DESC", userID)
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

func (r *WatchlistRepo) Add(userID int, item model.WatchlistItem) error {
	_, err := r.db.Exec(
		"INSERT INTO watchlist (symbol, name, added_at, user_id) VALUES (?, ?, ?, ?)",
		item.Symbol, item.Name, item.AddedAt, userID,
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

func (r *WatchlistRepo) Remove(userID int, symbol string) error {
	result, err := r.db.Exec("DELETE FROM watchlist WHERE symbol = ? AND user_id = ?", symbol, userID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetAllSymbols returns distinct symbols across all user watchlists.
func (r *WatchlistRepo) GetAllSymbols() ([]string, error) {
	rows, err := r.db.Query("SELECT DISTINCT symbol FROM watchlist")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, err
		}
		symbols = append(symbols, symbol)
	}
	return symbols, nil
}

package repo

import (
	"database/sql"
	"errors"

	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/mattn/go-sqlite3"
)

type HoldingRepo struct {
	db *sql.DB
}

func NewHoldingRepo(db *sql.DB) *HoldingRepo {
	return &HoldingRepo{db: db}
}

func (r *HoldingRepo) GetAll(userID int) ([]model.Holding, error) {
	rows, err := r.db.Query("SELECT symbol, name, shares, avg_cost, buy_date FROM holdings WHERE user_id = ? ORDER BY symbol", userID)
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

func (r *HoldingRepo) Add(userID int, h model.Holding) error {
	_, err := r.db.Exec(
		"INSERT INTO holdings (symbol, name, shares, avg_cost, buy_date, user_id) VALUES (?, ?, ?, ?, ?, ?)",
		h.Symbol, h.Name, h.Shares, h.AvgCost, h.BuyDate, userID,
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

func (r *HoldingRepo) Update(userID int, symbol string, fn func(*model.Holding)) error {
	items, err := r.GetAll(userID)
	if err != nil {
		return err
	}
	for i, h := range items {
		if h.Symbol == symbol {
			fn(&items[i])
			_, err = r.db.Exec(
				"UPDATE holdings SET shares=?, avg_cost=?, buy_date=?, name=? WHERE symbol=? AND user_id=?",
				items[i].Shares, items[i].AvgCost, items[i].BuyDate, items[i].Name, symbol, userID,
			)
			return err
		}
	}
	return ErrNotFound
}

func (r *HoldingRepo) Remove(userID int, symbol string) error {
	result, err := r.db.Exec("DELETE FROM holdings WHERE symbol = ? AND user_id = ?", symbol, userID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

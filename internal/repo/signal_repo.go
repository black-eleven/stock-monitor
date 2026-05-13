package repo

import (
	"database/sql"

	"github.com/black-eleven/stock-monitor/internal/model"
)

type SignalRepo struct {
	db *sql.DB
}

func NewSignalRepo(db *sql.DB) *SignalRepo {
	return &SignalRepo{db: db}
}

func (r *SignalRepo) Record(userID int, rec model.SignalRecord) error {
	_, err := r.db.Exec(
		`INSERT INTO signal_history (symbol, date, buy_score, buy_pct, sell_score, sell_pct, buy_count, sell_count, user_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(symbol, date, user_id) DO UPDATE SET
		   buy_score=excluded.buy_score, buy_pct=excluded.buy_pct,
		   sell_score=excluded.sell_score, sell_pct=excluded.sell_pct,
		   buy_count=excluded.buy_count, sell_count=excluded.sell_count`,
		rec.Symbol, rec.Date, rec.BuyScore, rec.BuyPct, rec.SellScore, rec.SellPct, rec.BuyCount, rec.SellCount, userID,
	)
	return err
}

func (r *SignalRepo) GetHistory(userID int, symbol string, days int) ([]model.SignalRecord, error) {
	rows, err := r.db.Query(
		`SELECT symbol, date, buy_score, buy_pct, sell_score, sell_pct, buy_count, sell_count
		 FROM signal_history WHERE symbol = ? AND user_id = ?
		 ORDER BY date DESC LIMIT ?`,
		symbol, userID, days,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []model.SignalRecord
	for rows.Next() {
		var rec model.SignalRecord
		if err := rows.Scan(&rec.Symbol, &rec.Date, &rec.BuyScore, &rec.BuyPct, &rec.SellScore, &rec.SellPct, &rec.BuyCount, &rec.SellCount); err != nil {
			return nil, err
		}
		recs = append(recs, rec)
	}
	if recs == nil {
		recs = []model.SignalRecord{}
	}
	return recs, nil
}

func (r *SignalRepo) GetLatest(userID int, symbol string) (*model.SignalRecord, error) {
	recs, err := r.GetHistory(userID, symbol, 1)
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, nil
	}
	return &recs[0], nil
}

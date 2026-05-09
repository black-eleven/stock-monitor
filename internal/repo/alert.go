package repo

import (
	"database/sql"
	"errors"

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
	rows, err := r.db.Query(
		"SELECT id, symbol, type, value, enabled, created_at, last_triggered_at FROM alerts ORDER BY id",
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

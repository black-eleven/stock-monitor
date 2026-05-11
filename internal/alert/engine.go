package alert

import (
	"fmt"
	"math"
	"time"

	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/black-eleven/stock-monitor/internal/ws"
)

type Engine struct {
	alertRepo *repo.AlertRepo
	hub       *ws.Hub
}

func NewEngine(alertRepo *repo.AlertRepo, hub *ws.Hub) *Engine {
	return &Engine{alertRepo: alertRepo, hub: hub}
}

type AlertEvent struct {
	AlertID     int     `json:"alertId"`
	Symbol      string  `json:"symbol"`
	Price       float64 `json:"price"`
	Type        string  `json:"type"`
	Value       float64 `json:"value"`
	Message     string  `json:"message"`
	TriggeredAt string  `json:"triggeredAt"`
}

type Quote interface {
	GetCode() string
	GetPrice() float64
	GetYP() float64
}

func (e *Engine) Evaluate(q Quote) {
	rules, err := e.alertRepo.GetBySymbolAll(q.GetCode())
	if err != nil {
		return
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		if rule.LastTriggeredAt != nil {
			lastTriggered, err := time.Parse(time.RFC3339, *rule.LastTriggeredAt)
			if err == nil && now.Sub(lastTriggered) < 30*time.Minute {
				continue
			}
		}

		triggered := false
		var message string

		switch rule.Type {
		case "above":
			if q.GetPrice() >= rule.Value {
				triggered = true
				message = fmt.Sprintf("%s 价格涨破 %.2f", q.GetCode(), rule.Value)
			}
		case "below":
			if q.GetPrice() <= rule.Value {
				triggered = true
				message = fmt.Sprintf("%s 价格跌破 %.2f", q.GetCode(), rule.Value)
			}
		case "change_pct":
			pct := math.Abs((q.GetPrice() - q.GetYP()) / q.GetYP() * 100)
			if pct >= math.Abs(rule.Value) {
				triggered = true
				dir := "涨"
				if q.GetPrice() < q.GetYP() {
					dir = "跌"
				}
				message = fmt.Sprintf("%s %s幅达 %.2f%%", q.GetCode(), dir, pct)
			}
		}

		if triggered {
			e.alertRepo.Update(rule.ID, func(a *model.AlertRule) {
				a.LastTriggeredAt = &nowStr
			})

			logEntry := model.AlertLog{
				AlertID:     rule.ID,
				Symbol:      q.GetCode(),
				Price:       q.GetPrice(),
				Message:     message,
				TriggeredAt: nowStr,
			}
			e.alertRepo.AppendLog(logEntry)
			e.alertRepo.PurgeOldLogs(200)

			e.hub.BroadcastAlert(AlertEvent{
				AlertID:     rule.ID,
				Symbol:      q.GetCode(),
				Price:       q.GetPrice(),
				Type:        rule.Type,
				Value:       rule.Value,
				Message:     message,
				TriggeredAt: nowStr,
			})
		}
	}
}

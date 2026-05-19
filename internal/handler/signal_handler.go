package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/black-eleven/stock-monitor/internal/ws"
	"github.com/gin-gonic/gin"
)

type SignalHandler struct {
	repo *repo.SignalRepo
	hub  *ws.Hub
}

func NewSignalHandler(repo *repo.SignalRepo, hub *ws.Hub) *SignalHandler {
	return &SignalHandler{repo: repo, hub: hub}
}

func (h *SignalHandler) Register(api *gin.RouterGroup) {
	api.POST("/signals/record", h.record)
	api.GET("/signals/:symbol/history", h.history)
}

func (h *SignalHandler) record(c *gin.Context) {
	userID := c.GetInt("userID")

	var req struct {
		Symbol    string  `json:"symbol"`
		BuyScore  float64 `json:"buyScore"`
		BuyPct    float64 `json:"buyPct"`
		SellScore float64 `json:"sellScore"`
		SellPct   float64 `json:"sellPct"`
		BuyCount  int     `json:"buyCount"`
		SellCount int     `json:"sellCount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	date := time.Now().Format("2006-01-02")
	rec := model.SignalRecord{
		Symbol:    req.Symbol,
		Date:      date,
		BuyScore:  req.BuyScore,
		BuyPct:    req.BuyPct,
		SellScore: req.SellScore,
		SellPct:   req.SellPct,
		BuyCount:  req.BuyCount,
		SellCount: req.SellCount,
	}

	if err := h.repo.Record(userID, rec); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record signal"})
		return
	}

	// Check threshold cross against previous record
	alert := h.checkThresholdCross(userID, req.Symbol, rec)
	if alert != nil && h.hub != nil {
		h.hub.BroadcastAlert(alert)
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"alert":  alert,
	})
}

func (h *SignalHandler) history(c *gin.Context) {
	userID := c.GetInt("userID")
	symbol := c.Param("symbol")
	daysStr := c.DefaultQuery("days", "30")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 || days > 90 {
		days = 30
	}

	recs, err := h.repo.GetHistory(userID, symbol, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch signal history"})
		return
	}

	c.JSON(http.StatusOK, recs)
}

func (h *SignalHandler) checkThresholdCross(userID int, symbol string, latest model.SignalRecord) *model.SignalAlert {
	// Look back 7 days to find the most recent previous record
	recs, err := h.repo.GetHistory(userID, symbol, 7)
	if err != nil || len(recs) < 2 {
		return nil
	}

	prev := recs[1] // most recent previous record (recs[0] is the latest we just inserted)
		now := time.Now().Format(time.RFC3339)

	// Threshold: cross from below 50% to >= 50%
	if prev.BuyPct < 50 && latest.BuyPct >= 50 {
		return &model.SignalAlert{
			Symbol:  symbol,
			Type:    "buy",
			OldPct:  prev.BuyPct,
			NewPct:  latest.BuyPct,
			TriggeredAt: now,
				Message: fmt.Sprintf("买入信号增强：从 %.0f%% 升至 %.0f%%（强烈买入）", prev.BuyPct, latest.BuyPct),
		}
	}
	if prev.SellPct < 50 && latest.SellPct >= 50 {
		return &model.SignalAlert{
			Symbol:  symbol,
			Type:    "sell",
			OldPct:  prev.SellPct,
			NewPct:  latest.SellPct,
			TriggeredAt: now,
				Message: fmt.Sprintf("卖出信号增强：从 %.0f%% 升至 %.0f%%（强烈卖出），注意风险", prev.SellPct, latest.SellPct),
		}
	}

	// Cross from 0% to >= 25% (new signal emerged)
	if prev.BuyPct < 25 && latest.BuyPct >= 25 && latest.BuyPct < 50 {
		return &model.SignalAlert{
			Symbol:  symbol,
			Type:    "buy",
			OldPct:  prev.BuyPct,
			NewPct:  latest.BuyPct,
			TriggeredAt: now,
				Message: fmt.Sprintf("出现买入信号：%.0f%%（值得关注）", latest.BuyPct),
		}
	}
	if prev.SellPct < 25 && latest.SellPct >= 25 && latest.SellPct < 50 {
		return &model.SignalAlert{
			Symbol:  symbol,
			Type:    "sell",
			OldPct:  prev.SellPct,
			NewPct:  latest.SellPct,
			TriggeredAt: now,
				Message: fmt.Sprintf("出现卖出信号：%.0f%%（注意风险）", latest.SellPct),
		}
	}

	return nil
}

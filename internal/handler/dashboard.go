package handler

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/black-eleven/stock-monitor/internal/middleware"
	"github.com/black-eleven/stock-monitor/internal/model"
	"github.com/black-eleven/stock-monitor/internal/repo"
	"github.com/black-eleven/stock-monitor/internal/ws"
	"github.com/gin-gonic/gin"
)

var indexSymbols = []struct{ Code, Name string }{
	{"SH:000001", "上证指数"},
	{"SZ:399001", "深证成指"},
	{"HK:HSI", "恒生指数"},
	{"US:IXIC", "纳斯达克"},
}

type DashboardHandler struct {
	hub           *ws.Hub
	watchlistRepo *repo.WatchlistRepo
	alertRepo     *repo.AlertRepo
	signalRepo    *repo.SignalRepo
}

func NewDashboardHandler(hub *ws.Hub, wr *repo.WatchlistRepo, ar *repo.AlertRepo, sr *repo.SignalRepo) *DashboardHandler {
	return &DashboardHandler{hub: hub, watchlistRepo: wr, alertRepo: ar, signalRepo: sr}
}

func (h *DashboardHandler) Register(api *gin.RouterGroup) {
	api.GET("/dashboard", h.serve)
}

type indexCard struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	ChangePct float64 `json:"changePct"`
}

type moverItem struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	ChangePct float64 `json:"changePct"`
}

type signalItem struct {
	Symbol   string  `json:"symbol"`
	Name     string  `json:"name"`
	BuyScore float64 `json:"buyScore"`
	BuyPct   float64 `json:"buyPct"`
}

func (h *DashboardHandler) serve(c *gin.Context) {
	userID := middleware.GetUserID(c)

	// 1. Index cards from hub cache
	indices := make([]indexCard, 0, len(indexSymbols))
	for _, idx := range indexSymbols {
		card := indexCard{Code: idx.Code, Name: idx.Name}
		if raw, ok := h.hub.GetQuote(idx.Code); ok {
			var q model.Quote
			if json.Unmarshal(raw, &q) == nil {
				card.Price = q.Price
				if q.YP > 0 {
					card.ChangePct = ((q.Price - q.YP) / q.YP) * 100
				}
			}
		}
		indices = append(indices, card)
	}

	// 2. Top gainers / losers from user's watchlist
	watchlist, err := h.watchlistRepo.GetAll(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch watchlist"})
		return
	}

	type watchlistQuote struct {
		symbol    string
		name      string
		price     float64
		changePct float64
	}
	var wlQuotes []watchlistQuote
	for _, w := range watchlist {
		wq := watchlistQuote{symbol: w.Symbol, name: w.Name}
		if raw, ok := h.hub.GetQuote(w.Symbol); ok {
			var q model.Quote
			if json.Unmarshal(raw, &q) == nil {
				wq.price = q.Price
				if q.YP > 0 {
					wq.changePct = ((q.Price - q.YP) / q.YP) * 100
				}
			}
		}
		wlQuotes = append(wlQuotes, wq)
	}

	// Build symbol→name lookup from watchlist
	nameMap := make(map[string]string, len(watchlist))
	for _, w := range watchlist {
		nameMap[w.Symbol] = w.Name
	}

	sort.Slice(wlQuotes, func(i, j int) bool { return wlQuotes[i].changePct > wlQuotes[j].changePct })

	topGainers := make([]moverItem, 0, 3)
	topLosers := make([]moverItem, 0, 3)
	for i, wq := range wlQuotes {
		if i < 3 {
			name := wq.name
			if name == "" {
				name = wq.symbol
			}
			topGainers = append(topGainers, moverItem{
				Symbol: wq.symbol, Name: name, Price: wq.price, ChangePct: wq.changePct,
			})
		}
	}
	for i := len(wlQuotes) - 1; i >= 0 && len(topLosers) < 3; i-- {
		name := wlQuotes[i].name
		if name == "" {
			name = wlQuotes[i].symbol
		}
		topLosers = append(topLosers, moverItem{
			Symbol: wlQuotes[i].symbol, Name: name, Price: wlQuotes[i].price, ChangePct: wlQuotes[i].changePct,
		})
	}

	// 3. Recent alert logs (enriched with watchlist names)
	rawLogs, err := h.alertRepo.GetLogs(3)
	if err != nil {
		rawLogs = []model.AlertLog{}
	}
	type alertLogWithName struct {
		ID          int     `json:"id"`
		AlertID     int     `json:"alertId"`
		Symbol      string  `json:"symbol"`
		Name        string  `json:"name"`
		Price       float64 `json:"price"`
		Message     string  `json:"message"`
		TriggeredAt string  `json:"triggeredAt"`
	}
	alertLogs := make([]alertLogWithName, 0, len(rawLogs))
	for _, l := range rawLogs {
		name := l.Symbol
		if n, ok := nameMap[l.Symbol]; ok {
			name = n
		}
		alertLogs = append(alertLogs, alertLogWithName{
			ID: l.ID, AlertID: l.AlertID, Symbol: l.Symbol, Name: name,
			Price: l.Price, Message: l.Message, TriggeredAt: l.TriggeredAt,
		})
	}

	// 4. Top buy signals from signal history
	signals, err := h.signalRepo.GetLatestBuySignals(userID, 3)
	if err != nil {
		signals = []model.SignalRecord{}
	}
	topSignals := make([]signalItem, 0, len(signals))
	for _, s := range signals {
		name := s.Symbol
		if n, ok := nameMap[s.Symbol]; ok {
			name = n
		}
		topSignals = append(topSignals, signalItem{
			Symbol: s.Symbol, Name: name, BuyScore: s.BuyScore, BuyPct: s.BuyPct,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"indices":      indices,
		"topGainers":   topGainers,
		"topLosers":    topLosers,
		"recentAlerts": alertLogs,
		"topSignals":   topSignals,
	})
}

// IndexSymbols returns the list of index symbols that should always be tracked.
func IndexSymbols() []string {
	symbols := make([]string, len(indexSymbols))
	for i, idx := range indexSymbols {
		symbols[i] = idx.Code
	}
	return symbols
}

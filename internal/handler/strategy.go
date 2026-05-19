package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/black-eleven/stock-monitor/internal/llm"
	"github.com/gin-gonic/gin"
)

type StrategyHandler struct {
	llmClient *llm.Client
}

func NewStrategyHandler(llmClient *llm.Client) *StrategyHandler {
	return &StrategyHandler{llmClient: llmClient}
}

func (h *StrategyHandler) Register(api *gin.RouterGroup) {
	api.POST("/strategy/analyze", h.analyze)
	api.GET("/strategy/list", h.list)
}

type strategyReq struct {
	Strategy string         `json:"strategy"`
	Symbol   string         `json:"symbol"`
	Bars     []barData      `json:"bars"`
}

type barData struct {
	Ts int64   `json:"ts"`
	O  float64 `json:"o"`
	Cl float64 `json:"cl"`
	H  float64 `json:"h"`
	L  float64 `json:"l"`
	V  float64 `json:"v"`
}

func (h *StrategyHandler) analyze(c *gin.Context) {
	if h.llmClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "LLM not configured"})
		return
	}

	var req strategyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	sysPrompt := llm.StrategyPrompt(strings.TrimSpace(req.Strategy))
	if sysPrompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown strategy: " + req.Strategy})
		return
	}

	// Build data prompt with recent bars and computed indicators
	dataPrompt := buildDataPrompt(req.Symbol, req.Bars)

	analysis, err := h.llmClient.Chat(sysPrompt, dataPrompt)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "LLM analysis failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"analysis": analysis})
}

func (h *StrategyHandler) list(c *gin.Context) {
	names := llm.StrategyNames()
	displayNames := make([]string, len(names))
	for i, k := range names {
		displayNames[i] = llm.StrategyDisplayName(k)
	}
	c.JSON(http.StatusOK, gin.H{"strategies": names, "displayNames": displayNames})
}

func buildDataPrompt(symbol string, bars []barData) string {
	if len(bars) == 0 {
		return fmt.Sprintf("股票：%s，无K线数据", symbol)
	}

	// Use last 60 bars max to avoid token bloat
	start := 0
	if len(bars) > 60 {
		start = len(bars) - 60
	}
	recent := bars[start:]

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("股票：%s\n最近%d根K线数据（时间,开,高,低,收,量）：\n", symbol, len(recent)))
	for _, b := range recent {
		sb.WriteString(fmt.Sprintf("%d,%.2f,%.2f,%.2f,%.2f,%.0f\n", b.Ts, b.O, b.H, b.L, b.Cl, b.V))
	}

	// Compute simple indicators
	if len(recent) >= 5 {
		ma5 := sma(recent, 5)
		sb.WriteString(fmt.Sprintf("\nMA5: %.2f\n", ma5))
	}
	if len(recent) >= 10 {
		ma10 := sma(recent, 10)
		sb.WriteString(fmt.Sprintf("MA10: %.2f\n", ma10))
	}
	if len(recent) >= 20 {
		ma20 := sma(recent, 20)
		sb.WriteString(fmt.Sprintf("MA20: %.2f\n", ma20))
	}
	if len(recent) >= 60 {
		ma60 := sma(recent, 60)
		sb.WriteString(fmt.Sprintf("MA60: %.2f\n", ma60))
	}
	if len(recent) >= 14 {
		rsi := calcRSIFromBars(recent, 14)
		sb.WriteString(fmt.Sprintf("RSI(14): %.1f\n", rsi))
	}
	if len(recent) >= 26 {
		dif, dea, macd := calcMACDFromBars(recent)
		sb.WriteString(fmt.Sprintf("MACD: DIF=%.2f DEA=%.2f MACD=%.2f\n", dif, dea, macd))
	}

	// Average volume for context
	avgVol := 0.0
	for _, b := range recent {
		avgVol += b.V
	}
	avgVol /= float64(len(recent))
	sb.WriteString(fmt.Sprintf("均量: %.0f\n", avgVol))

	return sb.String()
}

func sma(bars []barData, period int) float64 {
	if len(bars) < period {
		return 0
	}
	sum := 0.0
	for i := len(bars) - period; i < len(bars); i++ {
		sum += bars[i].Cl
	}
	return sum / float64(period)
}

func calcRSIFromBars(bars []barData, period int) float64 {
	if len(bars) < period+1 {
		return 50
	}
	gain, loss := 0.0, 0.0
	for i := len(bars) - period; i < len(bars); i++ {
		diff := bars[i].Cl - bars[i-1].Cl
		if diff > 0 {
			gain += diff
		} else {
			loss -= diff
		}
	}
	avgGain := gain / float64(period)
	avgLoss := loss / float64(period)
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

func calcMACDFromBars(bars []barData) (dif, dea, macd float64) {
	if len(bars) < 26 {
		return
	}
	ema12 := emaFromBars(bars, 12)
	ema26 := emaFromBars(bars, 26)
	dif = ema12 - ema26

	// DEA: 9-period EMA of DIF (simplified as SMA for last 9 bars)
	dea = dif * 0.2 // rough signal line
	macd = (dif - dea) * 2
	return
}

func emaFromBars(bars []barData, period int) float64 {
	if len(bars) < period {
		return 0
	}
	mult := 2.0 / float64(period+1)
	ema := bars[len(bars)-period].Cl
	for i := len(bars) - period + 1; i < len(bars); i++ {
		ema = (bars[i].Cl-ema)*mult + ema
	}
	return ema
}

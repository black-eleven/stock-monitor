package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/black-eleven/stock-monitor/internal/eastmoney"
	"github.com/gin-gonic/gin"
)

var ktMap = map[string]int{
	"1m": 1, "5m": 5, "15m": 15, "30m": 30,
	"1h": 60, "2h": 120, "4h": 240,
	"1d": 1001, "1w": 1007, "1M": 1030,
}

type KlineHandler struct {
	client eastmoney.QuoteClient
}

func NewKlineHandler(client eastmoney.QuoteClient) *KlineHandler {
	return &KlineHandler{client: client}
}

func (h *KlineHandler) Register(api *gin.RouterGroup) {
	api.GET("/kline/:symbol", h.getKline)
}

func (h *KlineHandler) getKline(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	interval := c.DefaultQuery("interval", "1d")
	countStr := c.DefaultQuery("count", "100")
	count, _ := strconv.Atoi(countStr)
	if count <= 0 {
		count = 100
	}

	kt, ok := ktMap[interval]
	if !ok {
		keys := make([]string, 0, len(ktMap))
		for k := range ktMap {
			keys = append(keys, k)
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid interval: " + interval + ". Supported: " + strings.Join(keys, ", "),
		})
		return
	}

	data, err := h.client.FetchHistoryKlineCached(symbol, kt, count)
	if err != nil {
		log.Printf("[Kline] Failed to fetch kline for %s (kt=%d): %v", symbol, kt, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch kline data"})
		return
	}
	c.JSON(http.StatusOK, data)
}

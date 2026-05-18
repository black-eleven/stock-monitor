package handler

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/black-eleven/stock-monitor/internal/eastmoney"
	"github.com/gin-gonic/gin"
)

var symbolRegex = regexp.MustCompile(`^(HK|SH|SZ|US):[A-Z0-9]{1,10}$`)

type QuoteHandler struct {
	client eastmoney.QuoteClient
}

func NewQuoteHandler(client eastmoney.QuoteClient) *QuoteHandler {
	return &QuoteHandler{client: client}
}

func (h *QuoteHandler) Register(api *gin.RouterGroup) {
	api.GET("/quote/batch", h.batch)
	api.GET("/quote/:symbol", h.single)
}

func (h *QuoteHandler) batch(c *gin.Context) {
	symbolsStr := c.Query("symbols")
	symbols := strings.Split(symbolsStr, ",")
	trimmed := make([]string, 0, len(symbols))
	for _, s := range symbols {
		s = strings.TrimSpace(s)
		if s != "" {
			trimmed = append(trimmed, s)
		}
	}
	if len(trimmed) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No symbols provided"})
		return
	}

	type result struct {
		symbol string
		quote  *eastmoney.Quote
	}
	results := make(chan result, len(trimmed))

	for _, s := range trimmed {
		go func(symbol string) {
			q, err := h.client.FetchQuoteCached(symbol)
			if err != nil {
				results <- result{symbol: symbol}
			} else {
				results <- result{symbol: symbol, quote: q}
			}
		}(s)
	}

	data := make(map[string]interface{})
	for i := 0; i < len(trimmed); i++ {
		r := <-results
		if r.quote != nil {
			data[r.symbol] = r.quote
		}
	}
	c.JSON(http.StatusOK, data)
}

func (h *QuoteHandler) single(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	if !symbolRegex.MatchString(symbol) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid symbol format. Use HK:700 / SH:600519 / SZ:000001 / US:AAPL"})
		return
	}
	quote, err := h.client.FetchQuoteCached(symbol)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch quote"})
		return
	}
	c.JSON(http.StatusOK, quote)
}
